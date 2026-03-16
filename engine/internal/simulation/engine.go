package simulation

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/welife-os/welife-os/engine/internal/graph"
	"github.com/welife-os/welife-os/engine/internal/llm"
	"github.com/welife-os/welife-os/engine/internal/storage"
	"github.com/welife-os/welife-os/engine/internal/task"
)

// Engine orchestrates parallel-life simulations.
type Engine struct {
	llm        llm.LLMClient
	store      *storage.Store
	tasks      *task.Manager
	profiler   *ProfileBuilder
	graphStore *graph.GraphStore
}

// NewEngine creates a simulation engine.
func NewEngine(llmClient llm.LLMClient, store *storage.Store, tasks *task.Manager, profiler *ProfileBuilder, graphStore *graph.GraphStore) *Engine {
	return &Engine{
		llm:        llmClient,
		store:      store,
		tasks:      tasks,
		profiler:   profiler,
		graphStore: graphStore,
	}
}

// RunSimulation starts an async simulation run. Returns the session ID and task ID.
func (e *Engine) RunSimulation(ctx context.Context, config SimulationConfig) (string, string, error) {
	if strings.TrimSpace(config.ConversationID) == "" {
		return "", "", fmt.Errorf("conversation_id is required")
	}
	if config.Steps <= 0 {
		config.Steps = 5
	}
	if config.ForkPoint.Description == "" {
		return "", "", fmt.Errorf("fork_point description is required")
	}
	if _, err := e.store.GetConversation(ctx, config.ConversationID); err != nil {
		return "", "", fmt.Errorf("conversation lookup: %w", err)
	}

	sessionID := fmt.Sprintf("sim_%d", time.Now().UnixMilli())

	// 只对当前会话的图谱做快照，避免把其他会话的数据混入模拟上下文。
	entities, err := e.store.ListEntitiesByConversation(ctx, config.ConversationID)
	if err != nil {
		return "", "", fmt.Errorf("listing entities: %w", err)
	}
	rels, err := e.store.ListRelationshipsByConversation(ctx, config.ConversationID)
	if err != nil {
		return "", "", fmt.Errorf("listing relationships: %w", err)
	}
	overview := e.graphStore.Overview(entities, rels)
	snapshotBytes, err := json.Marshal(overview)
	if err != nil {
		return "", "", fmt.Errorf("serializing graph snapshot: %w", err)
	}

	sess := storage.SimulationSession{
		ID:                    sessionID,
		ConversationID:        config.ConversationID,
		ForkDescription:       config.ForkPoint.Description,
		Status:                "running",
		StepCount:             config.Steps,
		OriginalGraphSnapshot: string(snapshotBytes),
	}

	// Create session BEFORE submitting task so the session row exists
	// even if the task fails immediately.
	if err := e.store.CreateSimulationSession(ctx, sess); err != nil {
		return "", "", fmt.Errorf("creating session: %w", err)
	}

	taskID := e.tasks.Submit("simulation:"+sessionID, func(taskCtx context.Context) error {
		return e.executeSimulation(taskCtx, sessionID, config)
	})

	// Update session with task ID
	if err := e.store.UpdateSimulationSession(ctx, sessionID, "running", "", "", 0); err != nil {
		log.Printf("simulation: failed to update session %s with task_id: %v", sessionID, err)
	}

	return sessionID, taskID, nil
}

// GetSession returns a simulation session by ID.
func (e *Engine) GetSession(ctx context.Context, id string) (storage.SimulationSession, error) {
	return e.store.GetSimulationSession(ctx, id)
}

// ListSessions returns all simulation sessions.
func (e *Engine) ListSessions(ctx context.Context, conversationID string) ([]storage.SimulationSession, error) {
	return e.store.ListSimulationSessionsByConversation(ctx, conversationID)
}

// GetSessionSteps returns all steps for a simulation session.
func (e *Engine) GetSessionSteps(ctx context.Context, sessionID string) ([]storage.SimulationStep, error) {
	return e.store.GetSimulationSteps(ctx, sessionID)
}

// BuildAllProfilesAsync triggers async profile building for all person entities.
func (e *Engine) BuildAllProfilesAsync(ctx context.Context, conversationID string) (string, error) {
	if strings.TrimSpace(conversationID) == "" {
		return "", fmt.Errorf("conversation_id is required")
	}
	if _, err := e.store.GetConversation(ctx, conversationID); err != nil {
		return "", fmt.Errorf("conversation lookup: %w", err)
	}

	taskID := e.tasks.Submit("build_profiles:"+conversationID, func(taskCtx context.Context) error {
		_, err := e.profiler.BuildAllProfiles(taskCtx, conversationID)
		return err
	})
	return taskID, nil
}

func (e *Engine) executeSimulation(ctx context.Context, sessionID string, config SimulationConfig) error {
	// 1. 仅克隆当前会话的图谱，避免跨会话边和节点进入模拟。
	entities, err := e.store.ListEntitiesByConversation(ctx, config.ConversationID)
	if err != nil {
		e.failSession(ctx, sessionID, err)
		return err
	}
	rels, err := e.store.ListRelationshipsByConversation(ctx, config.ConversationID)
	if err != nil {
		e.failSession(ctx, sessionID, err)
		return err
	}
	clonedGraph := buildScopedGraph(entities, rels)

	// Build name→entityID mapping so we can resolve person names to graph node IDs.
	nameToID := make(map[string]string, len(entities))
	for _, ent := range entities {
		nameToID[ent.Name] = ent.ID
	}

	// 2. Load/build person profiles.
	profiles, err := e.profiler.BuildAllProfiles(ctx, config.ConversationID)
	if err != nil {
		e.failSession(ctx, sessionID, err)
		return err
	}

	// 3. Apply fork point: modify cloned graph edge weights.
	// AffectedNodes contains entity IDs (not names).
	for _, nodeID := range config.ForkPoint.AffectedNodes {
		neighbors := clonedGraph.Neighbors(nodeID)
		if len(neighbors) == 0 {
			log.Printf("simulation: fork affected node %q has no neighbors or not found, skipping", nodeID)
			continue
		}
		for _, neighbor := range neighbors {
			_ = clonedGraph.AddEdge(nodeID, neighbor, 0.5)
		}
	}

	// 4. Multi-step evolution loop.
	var allStepDescriptions []string
	for step := 1; step <= config.Steps; step++ {
		if ctx.Err() != nil {
			e.failSession(ctx, sessionID, ctx.Err())
			return ctx.Err()
		}

		stepResult, err := e.runStep(ctx, step, clonedGraph, profiles, config, allStepDescriptions)
		if err != nil {
			e.failSession(ctx, sessionID, err)
			return err
		}

		// Apply relationship changes to cloned graph using entity IDs.
		for _, reaction := range stepResult.reactions {
			personID, ok := nameToID[reaction.PersonName]
			if !ok {
				log.Printf("simulation: person %q not found in entity map, skipping reaction", reaction.PersonName)
				continue
			}
			for _, change := range reaction.RelationshipChanges {
				targetID, ok := nameToID[change.Target]
				if !ok {
					log.Printf("simulation: target %q not found in entity map, skipping edge", change.Target)
					continue
				}
				if err := clonedGraph.AddEdge(personID, targetID, change.WeightDelta); err != nil {
					log.Printf("simulation: AddEdge %s→%s failed: %v", reaction.PersonName, change.Target, err)
				}
			}
		}

		// Build step description.
		var descParts []string
		for _, r := range stepResult.reactions {
			descParts = append(descParts, fmt.Sprintf("%s: %s", r.PersonName, r.Reaction))
		}
		stepDesc := strings.Join(descParts, "; ")
		allStepDescriptions = append(allStepDescriptions, stepDesc)

		// Serialize and save step.
		changesJSON, _ := json.Marshal(stepResult.reactions)
		reactionsJSON, _ := json.Marshal(stepResult.reactions)

		simStep := storage.SimulationStep{
			ID:            fmt.Sprintf("%s_step_%d", sessionID, step),
			SessionID:     sessionID,
			StepNumber:    step,
			Description:   stepDesc,
			EntityChanges: string(changesJSON),
			Reactions:     string(reactionsJSON),
		}
		if err := e.store.SaveSimulationStep(ctx, simStep); err != nil {
			e.failSession(ctx, sessionID, err)
			return err
		}
	}

	// 5. Generate narrative.
	narrative, err := e.generateNarrative(ctx, config, allStepDescriptions, profiles)
	if err != nil {
		e.failSession(ctx, sessionID, err)
		return err
	}

	// 6. Serialize final graph snapshot.
	finalEdges := clonedGraph.AllEdges()
	finalSnapshot, _ := json.Marshal(finalEdges)

	// 7. Update session as completed.
	return e.store.UpdateSimulationSession(ctx, sessionID, "completed", narrative, string(finalSnapshot), config.Steps)
}

type stepResult struct {
	reactions []personReaction
}

type personReaction struct {
	PersonName          string              `json:"person_name"`
	Reaction            string              `json:"reaction"`
	Actions             []string            `json:"actions"`
	RelationshipChanges []relationshipDelta `json:"relationship_changes"`
}

type relationshipDelta struct {
	Target      string  `json:"target"`
	Change      string  `json:"change"`
	WeightDelta float64 `json:"weight_delta"`
}

const stepPrompt = `你是一位平行人生模拟器。现在正在模拟一个假设情景的演变。

假设变化: %s

当前被模拟的人物: %s
人物性格: %s
人物关系模式: %s

当前关系图状态:
%s

%s

这是第 %d 步演变。请模拟这个人在此情景下会如何反应。

请以 JSON 格式输出：
{"reaction": "这个人的反应描述", "actions": ["具体行动1", "具体行动2"], "relationship_changes": [{"target": "对方名字", "change": "关系变化描述", "weight_delta": 0.1}]}

请只输出 JSON。`

func (e *Engine) runStep(ctx context.Context, step int, clonedGraph *graph.GraphStore, profiles []storage.PersonProfile, config SimulationConfig, prevDescriptions []string) (stepResult, error) {
	edges := clonedGraph.AllEdges()
	edgesJSON, _ := json.Marshal(edges)

	var prevContext string
	if len(prevDescriptions) > 0 {
		prevContext = "之前的演变:\n"
		for i, desc := range prevDescriptions {
			prevContext += fmt.Sprintf("第%d步: %s\n", i+1, desc)
		}
	}

	var mu sync.Mutex
	var reactions []personReaction
	var wg sync.WaitGroup

	for _, profile := range profiles {
		wg.Add(1)
		go func(p storage.PersonProfile) {
			defer wg.Done()

			prompt := fmt.Sprintf(stepPrompt,
				config.ForkPoint.Description,
				p.Name,
				p.Personality,
				p.RelationshipToSelf,
				string(edgesJSON),
				prevContext,
				step,
			)

			response, err := e.llm.Generate(ctx, prompt)
			if err != nil {
				return
			}

			jsonStr := llm.ExtractJSON(response)
			var reaction personReaction
			if err := json.Unmarshal([]byte(jsonStr), &reaction); err != nil {
				return
			}
			reaction.PersonName = p.Name

			mu.Lock()
			reactions = append(reactions, reaction)
			mu.Unlock()
		}(profile)
	}

	wg.Wait()

	return stepResult{reactions: reactions}, nil
}

const narrativePrompt = `你是一位人生叙事作家。根据以下平行人生模拟的演变过程，写一个引人深思的故事。

假设变化: %s

涉及的人物:
%s

演变过程:
%s

请写一段500-800字的叙事，描述这个平行人生中各人物的变化和故事发展。
使用第三人称，语调温暖但带有哲思。直接输出故事文本，不要输出JSON。`

func (e *Engine) generateNarrative(ctx context.Context, config SimulationConfig, stepDescriptions []string, profiles []storage.PersonProfile) (string, error) {
	var peopleParts []string
	for _, p := range profiles {
		peopleParts = append(peopleParts, fmt.Sprintf("- %s: %s", p.Name, p.Personality))
	}

	var stepParts []string
	for i, desc := range stepDescriptions {
		stepParts = append(stepParts, fmt.Sprintf("第%d步: %s", i+1, desc))
	}

	prompt := fmt.Sprintf(narrativePrompt,
		config.ForkPoint.Description,
		strings.Join(peopleParts, "\n"),
		strings.Join(stepParts, "\n"),
	)

	return e.llm.Generate(ctx, prompt)
}

func (e *Engine) failSession(ctx context.Context, sessionID string, err error) {
	if dbErr := e.store.UpdateSimulationSession(ctx, sessionID, "failed", err.Error(), "", 0); dbErr != nil {
		log.Printf("simulation: failed to mark session %s as failed: %v", sessionID, dbErr)
	}
}

func buildScopedGraph(entities []storage.Entity, relationships []storage.Relationship) *graph.GraphStore {
	scoped := graph.NewGraphStore()
	for _, entity := range entities {
		scoped.AddNode(entity.ID)
	}
	for _, relationship := range relationships {
		_ = scoped.AddEdge(relationship.SourceEntityID, relationship.TargetEntityID, relationship.Weight)
	}
	return scoped
}
