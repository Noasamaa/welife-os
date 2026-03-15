package forum_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/welife-os/welife-os/engine/internal/agent"
	"github.com/welife-os/welife-os/engine/internal/forum"
	"github.com/welife-os/welife-os/engine/internal/llm"
	"github.com/welife-os/welife-os/engine/internal/storage"
	"github.com/welife-os/welife-os/engine/internal/task"
)

type scopeProbeAgent struct{}

func (scopeProbeAgent) Name() string { return "scope_probe" }

func (scopeProbeAgent) Analyze(_ context.Context, input agent.AnalysisInput) (agent.AnalysisOutput, error) {
	return agent.AnalysisOutput{
		AgentName: "scope_probe",
		Summary:   "entities=" + strconv.Itoa(len(input.Entities)) + ",relationships=" + strconv.Itoa(len(input.Relationships)),
		Details: []agent.Finding{{
			Type:       "scope",
			Title:      "scope",
			Content:    "probe",
			Confidence: 1,
		}},
	}, nil
}

func (scopeProbeAgent) Debate(_ context.Context, state agent.DebateState) (agent.ForumMessage, error) {
	return agent.ForumMessage{
		AgentName:  "scope_probe",
		Round:      state.Round,
		Stance:     "scope",
		Content:    "scoped",
		Confidence: 1,
	}, nil
}

func newMockOllamaServer(t *testing.T) *httptest.Server {
	t.Helper()

	// This mock returns different responses based on the prompt content.
	// For simplicity, we return a generic valid response for all calls.
	analyzeResp := `{
		"emotion_timeline": [],
		"emotion_shifts": [],
		"relationship_temperature": 50,
		"summary": "测试分析",
		"opportunities": [],
		"missed_followups": [],
		"cross_links": [],
		"perspective": "realist",
		"risk_items": [],
		"overall_assessment": "测试评估",
		"consensus_risks": [],
		"divergences": [],
		"topics": ["议题1", "议题2"],
		"stance": "测试立场",
		"content": "测试论点",
		"evidence": [],
		"confidence": 0.8
	}`

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/generate" {
			w.Header().Set("Content-Type", "application/json")
			resp := map[string]any{
				"model":      "test",
				"response":   analyzeResp,
				"done":       true,
				"created_at": "2024-01-01T00:00:00Z",
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		if r.URL.Path == "/api/tags" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"models": []any{}})
			return
		}
		http.NotFound(w, r)
	}))
}

func setupTestEngine(t *testing.T) (*forum.Engine, *storage.Store, *task.Manager) {
	t.Helper()

	ollamaServer := newMockOllamaServer(t)
	t.Cleanup(ollamaServer.Close)

	llmClient, err := llm.New(llm.Config{
		BaseURL: ollamaServer.URL,
		Model:   "test",
	})
	if err != nil {
		t.Fatalf("create LLM client: %v", err)
	}

	store, err := storage.Open(context.Background(), storage.Config{
		Path: t.TempDir() + "/forum_engine_test.db",
		Key:  "test-key",
	})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	// Seed a conversation for the debate to reference
	if err := store.SaveConversation(context.Background(), storage.Conversation{
		ID:               "conv_test",
		Platform:         "test",
		ConversationType: "private",
		Title:            "测试对话",
		MessageCount:     2,
	}); err != nil {
		t.Fatalf("save conversation: %v", err)
	}

	if err := store.SaveMessages(context.Background(), []storage.StoredMessage{
		{ID: "m1", ConversationID: "conv_test", Platform: "test", SenderID: "u1", SenderName: "Alice", Content: "你好", MessageType: "text", Timestamp: "2024-01-01T10:00:00Z"},
		{ID: "m2", ConversationID: "conv_test", Platform: "test", SenderID: "u2", SenderName: "Bob", Content: "你好啊", MessageType: "text", Timestamp: "2024-01-01T10:01:00Z"},
	}); err != nil {
		t.Fatalf("save messages: %v", err)
	}

	taskMgr := task.NewManager(2)
	t.Cleanup(func() { _ = taskMgr.Close() })

	agents := []agent.Agent{
		agent.NewEmotionAgent(llmClient),
		agent.NewOpportunityAgent(llmClient),
		agent.NewRiskAgent(llmClient),
	}

	moderator := forum.NewModerator(llmClient)
	engine := forum.NewEngine(agents, moderator, store, taskMgr)

	return engine, store, taskMgr
}

func TestRunDebateCreatesSession(t *testing.T) {
	engine, store, _ := setupTestEngine(t)
	ctx := context.Background()

	sessionID, taskID, err := engine.RunDebate(ctx, "conv_test")
	if err != nil {
		t.Fatalf("run debate: %v", err)
	}

	if sessionID == "" {
		t.Fatal("expected non-empty session ID")
	}
	if taskID == "" {
		t.Fatal("expected non-empty task ID")
	}

	// Verify session was created in the database
	session, err := store.GetSession(ctx, sessionID)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if session.ConversationID != "conv_test" {
		t.Fatalf("expected conv_test, got %s", session.ConversationID)
	}
	if session.Status != "running" {
		t.Fatalf("expected running, got %s", session.Status)
	}
}

func TestRunDebateInvalidConversation(t *testing.T) {
	engine, _, _ := setupTestEngine(t)
	ctx := context.Background()

	_, _, err := engine.RunDebate(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent conversation")
	}
}

func TestRunDebateCompletesAsync(t *testing.T) {
	engine, store, taskMgr := setupTestEngine(t)
	ctx := context.Background()

	sessionID, taskID, err := engine.RunDebate(ctx, "conv_test")
	if err != nil {
		t.Fatalf("run debate: %v", err)
	}

	// Wait for the task to complete
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		info, ok := taskMgr.Status(taskID)
		if ok && (info.Status == task.StatusSucceeded || info.Status == task.StatusFailed) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	info, ok := taskMgr.Status(taskID)
	if !ok {
		t.Fatal("task not found")
	}
	if info.Status == task.StatusFailed {
		t.Fatalf("debate failed: %s", info.Error)
	}
	if info.Status != task.StatusSucceeded {
		t.Fatalf("expected succeeded, got %s", info.Status)
	}

	// Verify session is completed
	session, err := store.GetSession(ctx, sessionID)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if session.Status != "completed" {
		t.Fatalf("expected completed, got %s", session.Status)
	}
	if session.Summary == "" {
		t.Fatal("expected non-empty summary")
	}

	// Verify forum messages were saved
	msgs, err := store.GetForumMessages(ctx, sessionID)
	if err != nil {
		t.Fatalf("get forum messages: %v", err)
	}
	if len(msgs) == 0 {
		t.Fatal("expected forum messages to be saved")
	}

	// Should have round 1 messages (3 agents) + debate round messages
	hasRound1 := false
	hasRound2 := false
	for _, m := range msgs {
		if m.Round == 1 {
			hasRound1 = true
		}
		if m.Round == 2 {
			hasRound2 = true
		}
	}
	if !hasRound1 {
		t.Fatal("expected round 1 messages")
	}
	if !hasRound2 {
		t.Fatal("expected round 2 (debate) messages")
	}
}

func TestListSessions(t *testing.T) {
	engine, _, _ := setupTestEngine(t)
	ctx := context.Background()

	sessions, err := engine.ListSessions(ctx)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("expected empty sessions initially, got %d", len(sessions))
	}

	_, _, err = engine.RunDebate(ctx, "conv_test")
	if err != nil {
		t.Fatalf("run debate: %v", err)
	}

	sessions, err = engine.ListSessions(ctx)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
}

func TestRunDebateScopesEntitiesAndRelationshipsToConversation(t *testing.T) {
	ollamaServer := newMockOllamaServer(t)
	defer ollamaServer.Close()

	llmClient, err := llm.New(llm.Config{
		BaseURL: ollamaServer.URL,
		Model:   "test",
	})
	if err != nil {
		t.Fatalf("create LLM client: %v", err)
	}

	store, err := storage.Open(context.Background(), storage.Config{
		Path: t.TempDir() + "/forum_scope_test.db",
		Key:  "test-key",
	})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	for _, conv := range []storage.Conversation{
		{ID: "conv_a", Platform: "test", ConversationType: "private", Title: "A", MessageCount: 1},
		{ID: "conv_b", Platform: "test", ConversationType: "private", Title: "B", MessageCount: 1},
	} {
		if err := store.SaveConversation(ctx, conv); err != nil {
			t.Fatalf("save conversation: %v", err)
		}
	}
	if err := store.SaveMessages(ctx, []storage.StoredMessage{
		{ID: "msg_a", ConversationID: "conv_a", Platform: "test", SenderID: "u1", SenderName: "Alice", Content: "hello", MessageType: "text", Timestamp: "2024-01-01T10:00:00Z"},
	}); err != nil {
		t.Fatalf("save messages: %v", err)
	}
	if err := store.SaveEntities(ctx, []storage.Entity{
		{ID: "entity_a_1", Type: "person", Name: "Alice", SourceConversation: "conv_a"},
		{ID: "entity_a_2", Type: "person", Name: "Bob", SourceConversation: "conv_a"},
		{ID: "entity_b_1", Type: "person", Name: "Mallory", SourceConversation: "conv_b"},
	}); err != nil {
		t.Fatalf("save entities: %v", err)
	}
	if err := store.SaveRelationships(ctx, []storage.Relationship{
		{ID: "rel_a", SourceEntityID: "entity_a_1", TargetEntityID: "entity_a_2", Type: "friend", Weight: 1},
		{ID: "rel_b", SourceEntityID: "entity_b_1", TargetEntityID: "entity_a_1", Type: "watching", Weight: 1},
	}); err != nil {
		t.Fatalf("save relationships: %v", err)
	}

	taskMgr := task.NewManager(1)
	defer func() { _ = taskMgr.Close() }()

	engine := forum.NewEngine([]agent.Agent{scopeProbeAgent{}}, forum.NewModerator(llmClient), store, taskMgr)
	sessionID, taskID, err := engine.RunDebate(ctx, "conv_a")
	if err != nil {
		t.Fatalf("run debate: %v", err)
	}

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		info, ok := taskMgr.Status(taskID)
		if ok && (info.Status == task.StatusSucceeded || info.Status == task.StatusFailed) {
			if info.Status == task.StatusFailed {
				t.Fatalf("debate failed: %s", info.Error)
			}
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	msgs, err := store.GetForumMessages(ctx, sessionID)
	if err != nil {
		t.Fatalf("get forum messages: %v", err)
	}
	if len(msgs) == 0 {
		t.Fatal("expected saved forum messages")
	}
	if !strings.Contains(msgs[0].Content, "entities=2,relationships=1") {
		t.Fatalf("unexpected scope summary: %q", msgs[0].Content)
	}
}

func newEmptyTopicsOllamaServer(t *testing.T) *httptest.Server {
	t.Helper()

	genericResp := `{
		"emotion_timeline": [],
		"emotion_shifts": [],
		"relationship_temperature": 50,
		"summary": "测试分析",
		"opportunities": [],
		"missed_followups": [],
		"cross_links": [],
		"perspective": "realist",
		"risk_items": [],
		"overall_assessment": "测试评估",
		"consensus_risks": [],
		"divergences": [],
		"stance": "测试立场",
		"content": "测试论点",
		"evidence": [],
		"confidence": 0.8
	}`

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/generate" {
			var req struct {
				Prompt string `json:"prompt"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode generate request: %v", err)
			}

			response := genericResp
			switch {
			case strings.Contains(req.Prompt, "提取 2-3 个值得深入辩论的核心议题"):
				response = `{"topics": []}`
			case strings.Contains(req.Prompt, "辩论已经结束"):
				response = `{"summary": "空议题回退摘要"}`
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"model":      "test",
				"response":   response,
				"done":       true,
				"created_at": "2024-01-01T00:00:00Z",
			})
			return
		}
		if r.URL.Path == "/api/tags" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"models": []any{}})
			return
		}
		http.NotFound(w, r)
	}))
}

func TestRunDebateFallsBackWhenModeratorReturnsNoTopics(t *testing.T) {
	ollamaServer := newEmptyTopicsOllamaServer(t)
	defer ollamaServer.Close()

	llmClient, err := llm.New(llm.Config{
		BaseURL: ollamaServer.URL,
		Model:   "test",
	})
	if err != nil {
		t.Fatalf("create LLM client: %v", err)
	}

	store, err := storage.Open(context.Background(), storage.Config{
		Path: t.TempDir() + "/forum_engine_empty_topics.db",
		Key:  "test-key",
	})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer func() { _ = store.Close() }()

	if err := store.SaveConversation(context.Background(), storage.Conversation{
		ID:               "conv_empty_topics",
		Platform:         "test",
		ConversationType: "private",
		Title:            "测试对话",
		MessageCount:     2,
	}); err != nil {
		t.Fatalf("save conversation: %v", err)
	}
	if err := store.SaveMessages(context.Background(), []storage.StoredMessage{
		{ID: "m1", ConversationID: "conv_empty_topics", Platform: "test", SenderID: "u1", SenderName: "Alice", Content: "你好", MessageType: "text", Timestamp: "2024-01-01T10:00:00Z"},
		{ID: "m2", ConversationID: "conv_empty_topics", Platform: "test", SenderID: "u2", SenderName: "Bob", Content: "你好啊", MessageType: "text", Timestamp: "2024-01-01T10:01:00Z"},
	}); err != nil {
		t.Fatalf("save messages: %v", err)
	}

	taskMgr := task.NewManager(2)
	defer func() { _ = taskMgr.Close() }()

	engine := forum.NewEngine([]agent.Agent{
		agent.NewEmotionAgent(llmClient),
		agent.NewOpportunityAgent(llmClient),
		agent.NewRiskAgent(llmClient),
	}, forum.NewModerator(llmClient), store, taskMgr)

	sessionID, taskID, err := engine.RunDebate(context.Background(), "conv_empty_topics")
	if err != nil {
		t.Fatalf("run debate: %v", err)
	}

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		info, ok := taskMgr.Status(taskID)
		if ok && (info.Status == task.StatusSucceeded || info.Status == task.StatusFailed) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	info, ok := taskMgr.Status(taskID)
	if !ok {
		t.Fatal("task not found")
	}
	if info.Status != task.StatusSucceeded {
		t.Fatalf("expected succeeded, got %s (%s)", info.Status, info.Error)
	}

	session, err := store.GetSession(context.Background(), sessionID)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if session.Status != "completed" {
		t.Fatalf("expected completed, got %s", session.Status)
	}
	if session.Summary == "" {
		t.Fatal("expected summary to be populated")
	}
}
