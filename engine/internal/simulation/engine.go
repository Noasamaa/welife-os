package simulation

import (
	"context"
	"encoding/json"
	"fmt"
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
	llm        *llm.Client
	store      *storage.Store
	tasks      *task.Manager
	profiler   *ProfileBuilder
	graphStore *graph.GraphStore
}

// NewEngine creates a simulation engine.
func NewEngine(llmClient *llm.Client, store *storage.Store, tasks *task.Manager, profiler *ProfileBuilder, graphStore *graph.GraphStore) *Engine {
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
	if config.Steps <= 0 {
		config.Steps = 5
	}
	if config.ForkPoint.Description == "" {
		return "", "", fmt.Errorf("fork_point description is required")
	}

	sessionID := fmt.Sprintf("sim_%d", time.Now().UnixMilli())

	// Take original graph snapshot.
	entities, err := e.store.ListEntities(ctx)
	if err != nil {
		return "", "", fmt.Errorf("listing entities: %w", err)
	}
	rels, err := e.store.ListRelationships(ctx)
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
		ForkDescription:       config.ForkPoint.Description,
		Status:                "running",
		StepCount:             config.Steps,
		OriginalGraphSnapshot: string(snapshotBytes),
	}

	taskID := e.tasks.Submit("simulation:"+sessionID, func(taskCtx context.Context) error {
		return e.executeSimulation(taskCtx, sessionID, config)
	})

	sess.TaskID = taskID
	if err := e.store.CreateSimulationSession(ctx, sess); err != nil {
		return "", "", fmt.Errorf("creating session: %w", err)
	}

	return sessionID, taskID, nil
}

// GetSession returns a simulation session by ID.
func (e *Engine) GetSession(ctx context.Context, id string) (storage.SimulationSession, error) {
	return e.store.GetSimulationSession(ctx, id)
}

// ListSessions returns all simulation sessions.
func (e *Engine) ListSessions(ctx context.Context) ([]storage.SimulationSession, error) {
	return e.store.ListSimulationSessions(ctx)
}

// GetSessionSteps returns all steps for a simulation session.
func (e *Engine) GetSessionSteps(ctx context.Context, sessionID string) ([]storage.SimulationStep, error) {
	return e.store.GetSimulationSteps(ctx, sessionID)
}

// BuildAllProfilesAsync triggers async profile building for all person entities.
func (e *Engine) BuildAllProfilesAsync(_ context.Context) (string, error) {
	taskID := e.tasks.Submit("build_profiles", func(taskCtx context.Context) error {
		_, err := e.profiler.BuildAllProfiles(taskCtx)
		return err
	})
	return taskID, nil
}

func (e *Engine) executeSimulation(ctx context.Context, sessionID string, config SimulationConfig) error {
	// 1. Clone graph.
	clonedGraph := e.graphStore.Clone()

	// 2. Load/build person profiles.
	profiles, err := e.profiler.BuildAllProfiles(ctx)
	if err != nil {
		e.failSession(ctx, sessionID, err)
		return err
	}

	// 3. Apply fork point: modify cloned graph edge weights.
	for _, nodeName := range config.ForkPoint.AffectedNodes {
		neighbors := clonedGraph.Neighbors(nodeName)
		for _, neighbor := range neighbors {
			_ = clonedGraph.AddEdge(nodeName, neighbor, 0.5)
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

		// Apply relationship changes to cloned graph.
		for _, reaction := range stepResult.reactions {
			for _, change := range reaction.RelationshipChanges {
				_ = clonedGraph.AddEdge(reaction.PersonName, change.Target, change.WeightDelta)
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
	PersonName          string             `json:"person_name"`
	Reaction            string             `json:"reaction"`
	Actions             []string           `json:"actions"`
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
	_ = e.store.UpdateSimulationSession(ctx, sessionID, "failed", err.Error(), "", 0)
}
