package forum

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/welife-os/welife-os/engine/internal/agent"
	"github.com/welife-os/welife-os/engine/internal/storage"
	"github.com/welife-os/welife-os/engine/internal/task"
)

// sessionSeq provides a monotonically increasing counter for unique session IDs.
var sessionSeq uint64

// Engine orchestrates multi-agent debates on conversation data.
// It runs a 3-round debate process:
//  1. Independent analysis by all agents
//  2. Cross-debate rounds on generated topics
//  3. Moderator consensus summary
type Engine struct {
	agents    []agent.Agent
	moderator *Moderator
	store     *storage.Store
	tasks     *task.Manager
	config    DebateConfig
}

// NewEngine creates a new forum debate engine.
func NewEngine(agents []agent.Agent, moderator *Moderator, store *storage.Store, tasks *task.Manager) *Engine {
	return &Engine{
		agents:    agents,
		moderator: moderator,
		store:     store,
		tasks:     tasks,
		config:    DefaultConfig(),
	}
}

// RunDebate starts an async debate for a conversation, returning the session ID and task ID.
func (e *Engine) RunDebate(ctx context.Context, conversationID string) (string, string, error) {
	// Verify conversation exists
	if _, err := e.store.GetConversation(ctx, conversationID); err != nil {
		return "", "", fmt.Errorf("conversation lookup: %w", err)
	}

	seq := atomic.AddUint64(&sessionSeq, 1)
	sessionID := fmt.Sprintf("forum_%d_%d", time.Now().UnixNano(), seq)

	// Create session with placeholder task ID; will be updated after Submit
	if err := e.store.CreateSession(ctx, storage.ForumSession{
		ID:             sessionID,
		ConversationID: conversationID,
		TaskID:         "pending",
		Status:         string(StatusRunning),
	}); err != nil {
		return "", "", fmt.Errorf("creating session: %w", err)
	}

	taskID := e.tasks.Submit("forum_debate", func(taskCtx context.Context) error {
		return e.executeDebate(taskCtx, sessionID, conversationID)
	})

	// Update the session with the real task ID
	if err := e.store.UpdateSession(ctx, sessionID, string(StatusRunning), taskID, ""); err != nil {
		return "", "", fmt.Errorf("updating session task_id: %w", err)
	}

	return sessionID, taskID, nil
}

// GetSession returns a forum session by ID.
func (e *Engine) GetSession(ctx context.Context, sessionID string) (storage.ForumSession, error) {
	return e.store.GetSession(ctx, sessionID)
}

// ListSessions returns all forum sessions.
func (e *Engine) ListSessions(ctx context.Context) ([]storage.ForumSession, error) {
	return e.store.ListSessions(ctx)
}

// GetSessionMessages returns all messages for a debate session.
func (e *Engine) GetSessionMessages(ctx context.Context, sessionID string) ([]storage.ForumMessageRecord, error) {
	return e.store.GetForumMessages(ctx, sessionID)
}

func (e *Engine) executeDebate(ctx context.Context, sessionID, conversationID string) error {
	// Load conversation data
	messages, err := e.store.GetMessages(ctx, conversationID, 1000, 0)
	if err != nil {
		e.failSession(ctx, sessionID, err)
		return err
	}

	entities, err := e.store.ListEntities(ctx)
	if err != nil {
		e.failSession(ctx, sessionID, err)
		return err
	}

	relationships, err := e.store.ListRelationships(ctx)
	if err != nil {
		e.failSession(ctx, sessionID, err)
		return err
	}

	input := agent.AnalysisInput{
		ConversationID: conversationID,
		Messages:       messages,
		Entities:       entities,
		Relationships:  relationships,
	}

	// === Round 1: Independent analysis ===
	analyses, round1Msgs, err := e.runAnalysis(ctx, sessionID, input)
	if err != nil {
		e.failSession(ctx, sessionID, err)
		return err
	}

	if err := e.store.SaveForumMessages(ctx, round1Msgs); err != nil {
		e.failSession(ctx, sessionID, err)
		return err
	}

	// === Round 2+: Cross-debate ===
	topics, err := e.moderator.GenerateTopics(ctx, analyses)
	if err != nil {
		e.failSession(ctx, sessionID, err)
		return err
	}

	if len(topics) == 0 {
		topics = []string{"请围绕当前最重要的分歧、证据和行动建议继续辩论"}
	}

	var allDebateMsgs []agent.ForumMessage
	for _, r1 := range round1Msgs {
		allDebateMsgs = append(allDebateMsgs, toAgentMessage(r1))
	}

	for round := 0; round < e.config.DebateRounds; round++ {
		topic := topics[0]
		if round < len(topics) {
			topic = topics[round]
		}

		roundMsgs, err := e.runDebateRound(ctx, sessionID, round+2, topic, analyses, allDebateMsgs)
		if err != nil {
			e.failSession(ctx, sessionID, err)
			return err
		}

		if err := e.store.SaveForumMessages(ctx, roundMsgs); err != nil {
			e.failSession(ctx, sessionID, err)
			return err
		}

		for _, rm := range roundMsgs {
			allDebateMsgs = append(allDebateMsgs, toAgentMessage(rm))
		}
	}

	// === Final: Moderator consensus ===
	summary, err := e.moderator.Summarize(ctx, allDebateMsgs)
	if err != nil {
		e.failSession(ctx, sessionID, err)
		return err
	}

	if err := e.store.UpdateSession(ctx, sessionID, string(StatusCompleted), "", summary); err != nil {
		return fmt.Errorf("completing session: %w", err)
	}

	return nil
}

// runAnalysis executes Round 1: all agents analyze independently in parallel.
func (e *Engine) runAnalysis(ctx context.Context, sessionID string, input agent.AnalysisInput) ([]agent.AnalysisOutput, []storage.ForumMessageRecord, error) {
	type result struct {
		idx    int
		output agent.AnalysisOutput
		err    error
	}

	results := make([]result, len(e.agents))
	var wg sync.WaitGroup

	for i, ag := range e.agents {
		wg.Add(1)
		go func(idx int, a agent.Agent) {
			defer wg.Done()
			out, err := a.Analyze(ctx, input)
			results[idx] = result{idx: idx, output: out, err: err}
		}(i, ag)
	}

	wg.Wait()

	analyses := make([]agent.AnalysisOutput, 0, len(e.agents))
	var msgs []storage.ForumMessageRecord

	for _, r := range results {
		if r.err != nil {
			return nil, nil, fmt.Errorf("agent %d analysis: %w", r.idx, r.err)
		}
		analyses = append(analyses, r.output)

		// Convert analysis output to a forum message record
		detailsJSON, err := json.Marshal(r.output.Details)
		if err != nil {
			return nil, nil, fmt.Errorf("marshalling details for %s: %w", r.output.AgentName, err)
		}
		msgs = append(msgs, storage.ForumMessageRecord{
			ID:         fmt.Sprintf("%s_r1_%s", sessionID, r.output.AgentName),
			SessionID:  sessionID,
			AgentName:  r.output.AgentName,
			Round:      1,
			Stance:     "analysis",
			Content:    r.output.Summary,
			Evidence:   string(detailsJSON),
			Confidence: averageConfidence(r.output.Details),
		})
	}

	return analyses, msgs, nil
}

// runDebateRound executes one cross-debate round where agents respond to each other.
func (e *Engine) runDebateRound(
	ctx context.Context,
	sessionID string,
	round int,
	topic string,
	analyses []agent.AnalysisOutput,
	history []agent.ForumMessage,
) ([]storage.ForumMessageRecord, error) {
	type result struct {
		idx int
		msg agent.ForumMessage
		err error
	}

	results := make([]result, len(e.agents))
	var wg sync.WaitGroup

	for i, ag := range e.agents {
		// Build debate state for this agent
		myPrior := findAnalysis(analyses, ag.Name())
		otherViews := filterAnalyses(analyses, ag.Name())

		state := agent.DebateState{
			SessionID:  sessionID,
			Round:      round,
			Topic:      topic,
			History:    history,
			MyPrior:    myPrior,
			OtherViews: otherViews,
		}

		wg.Add(1)
		go func(idx int, a agent.Agent, s agent.DebateState) {
			defer wg.Done()
			msg, err := a.Debate(ctx, s)
			results[idx] = result{idx: idx, msg: msg, err: err}
		}(i, ag, state)
	}

	wg.Wait()

	var msgs []storage.ForumMessageRecord
	for _, r := range results {
		if r.err != nil {
			return nil, fmt.Errorf("agent %d debate: %w", r.idx, r.err)
		}

		evidenceJSON, err := json.Marshal(r.msg.Evidence)
		if err != nil {
			return nil, fmt.Errorf("marshalling evidence for %s: %w", r.msg.AgentName, err)
		}
		msgs = append(msgs, storage.ForumMessageRecord{
			ID:         fmt.Sprintf("%s_r%d_%s", sessionID, round, r.msg.AgentName),
			SessionID:  sessionID,
			AgentName:  r.msg.AgentName,
			Round:      round,
			Stance:     r.msg.Stance,
			Content:    r.msg.Content,
			Evidence:   string(evidenceJSON),
			Confidence: r.msg.Confidence,
		})
	}

	return msgs, nil
}

func (e *Engine) failSession(ctx context.Context, sessionID string, err error) {
	if dbErr := e.store.UpdateSession(ctx, sessionID, string(StatusFailed), "", err.Error()); dbErr != nil {
		log.Printf("forum: failed to mark session %s as failed: %v", sessionID, dbErr)
	}
}

func findAnalysis(analyses []agent.AnalysisOutput, name string) *agent.AnalysisOutput {
	for _, a := range analyses {
		if a.AgentName == name {
			return &a
		}
	}
	return nil
}

func filterAnalyses(analyses []agent.AnalysisOutput, excludeName string) []agent.AnalysisOutput {
	var filtered []agent.AnalysisOutput
	for _, a := range analyses {
		if a.AgentName != excludeName {
			filtered = append(filtered, a)
		}
	}
	return filtered
}

func averageConfidence(findings []agent.Finding) float64 {
	if len(findings) == 0 {
		return 0
	}
	var sum float64
	for _, f := range findings {
		sum += f.Confidence
	}
	return sum / float64(len(findings))
}

func toAgentMessage(record storage.ForumMessageRecord) agent.ForumMessage {
	var evidence []string
	// Evidence may be a JSON array of strings or a JSON array of Finding objects.
	// Either way, a parse failure is non-fatal; we proceed with nil evidence.
	if record.Evidence != "" {
		_ = json.Unmarshal([]byte(record.Evidence), &evidence)
	}
	return agent.ForumMessage{
		AgentName:  record.AgentName,
		Round:      record.Round,
		Stance:     record.Stance,
		Content:    record.Content,
		Evidence:   evidence,
		Confidence: record.Confidence,
	}
}
