package forum_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/welife-os/welife-os/engine/internal/agent"
	"github.com/welife-os/welife-os/engine/internal/forum"
	"github.com/welife-os/welife-os/engine/internal/llm"
	"github.com/welife-os/welife-os/engine/internal/storage"
	"github.com/welife-os/welife-os/engine/internal/task"
)

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
	if sessions != nil && len(sessions) != 0 {
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
