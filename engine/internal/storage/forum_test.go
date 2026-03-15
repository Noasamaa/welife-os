package storage_test

import (
	"context"
	"testing"

	"github.com/welife-os/welife-os/engine/internal/storage"
)

func openTestStore(t *testing.T) *storage.Store {
	t.Helper()
	store, err := storage.Open(context.Background(), storage.Config{
		Path: t.TempDir() + "/forum_test.db",
		Key:  "test-key",
	})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func TestCreateAndGetSession(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	session := storage.ForumSession{
		ID:             "sess_001",
		ConversationID: "conv_001",
		TaskID:         "task_001",
		Status:         "running",
	}

	if err := store.CreateSession(ctx, session); err != nil {
		t.Fatalf("create session: %v", err)
	}

	got, err := store.GetSession(ctx, "sess_001")
	if err != nil {
		t.Fatalf("get session: %v", err)
	}

	if got.ID != "sess_001" {
		t.Fatalf("expected sess_001, got %s", got.ID)
	}
	if got.ConversationID != "conv_001" {
		t.Fatalf("expected conv_001, got %s", got.ConversationID)
	}
	if got.Status != "running" {
		t.Fatalf("expected running, got %s", got.Status)
	}
}

func TestUpdateSession(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	session := storage.ForumSession{
		ID:             "sess_002",
		ConversationID: "conv_001",
		TaskID:         "task_002",
		Status:         "running",
	}
	if err := store.CreateSession(ctx, session); err != nil {
		t.Fatalf("create session: %v", err)
	}

	if err := store.UpdateSession(ctx, "sess_002", "completed", "", "test summary"); err != nil {
		t.Fatalf("update session: %v", err)
	}

	got, err := store.GetSession(ctx, "sess_002")
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if got.Status != "completed" {
		t.Fatalf("expected completed, got %s", got.Status)
	}
	if got.Summary != "test summary" {
		t.Fatalf("expected 'test summary', got %s", got.Summary)
	}
	if got.TaskID != "task_002" {
		t.Fatalf("expected task_002 to be preserved, got %s", got.TaskID)
	}
	if got.CompletedAt == "" {
		t.Fatal("expected completed_at to be set")
	}
}

func TestListSessions(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	for _, id := range []string{"sess_a", "sess_b", "sess_c"} {
		if err := store.CreateSession(ctx, storage.ForumSession{
			ID:             id,
			ConversationID: "conv_001",
			TaskID:         "task_" + id,
			Status:         "running",
		}); err != nil {
			t.Fatalf("create session %s: %v", id, err)
		}
	}

	sessions, err := store.ListSessions(ctx)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 3 {
		t.Fatalf("expected 3 sessions, got %d", len(sessions))
	}
}

func TestGetSessionNotFound(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	_, err := store.GetSession(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent session")
	}
}

func TestSaveAndGetForumMessages(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	// Create a session first (FK constraint)
	if err := store.CreateSession(ctx, storage.ForumSession{
		ID:             "sess_msg",
		ConversationID: "conv_001",
		TaskID:         "task_msg",
		Status:         "running",
	}); err != nil {
		t.Fatalf("create session: %v", err)
	}

	msg := storage.ForumMessageRecord{
		ID:         "msg_001",
		SessionID:  "sess_msg",
		AgentName:  "emotion_analyst",
		Round:      1,
		Stance:     "analysis",
		Content:    "情感分析结果",
		Evidence:   `["msg1","msg2"]`,
		Confidence: 0.85,
	}

	if err := store.SaveForumMessage(ctx, msg); err != nil {
		t.Fatalf("save message: %v", err)
	}

	msgs, err := store.GetForumMessages(ctx, "sess_msg")
	if err != nil {
		t.Fatalf("get messages: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].ID != "msg_001" {
		t.Fatalf("expected msg_001, got %s", msgs[0].ID)
	}
	if msgs[0].Confidence != 0.85 {
		t.Fatalf("expected confidence 0.85, got %f", msgs[0].Confidence)
	}
}

func TestSaveForumMessagesBatch(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	if err := store.CreateSession(ctx, storage.ForumSession{
		ID:             "sess_batch",
		ConversationID: "conv_001",
		TaskID:         "task_batch",
		Status:         "running",
	}); err != nil {
		t.Fatalf("create session: %v", err)
	}

	msgs := []storage.ForumMessageRecord{
		{ID: "bm1", SessionID: "sess_batch", AgentName: "emotion_analyst", Round: 1, Stance: "analysis", Content: "分析1", Confidence: 0.8},
		{ID: "bm2", SessionID: "sess_batch", AgentName: "opportunity_miner", Round: 1, Stance: "analysis", Content: "分析2", Confidence: 0.7},
		{ID: "bm3", SessionID: "sess_batch", AgentName: "risk_debate_team", Round: 1, Stance: "analysis", Content: "分析3", Confidence: 0.6},
	}

	if err := store.SaveForumMessages(ctx, msgs); err != nil {
		t.Fatalf("save messages: %v", err)
	}

	got, err := store.GetForumMessages(ctx, "sess_batch")
	if err != nil {
		t.Fatalf("get messages: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(got))
	}
}

func TestForumMessagesOrdering(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	if err := store.CreateSession(ctx, storage.ForumSession{
		ID:             "sess_order",
		ConversationID: "conv_001",
		TaskID:         "task_order",
		Status:         "running",
	}); err != nil {
		t.Fatalf("create session: %v", err)
	}

	msgs := []storage.ForumMessageRecord{
		{ID: "om3", SessionID: "sess_order", AgentName: "risk_debate_team", Round: 2, Stance: "debate", Content: "round2", Confidence: 0.5},
		{ID: "om1", SessionID: "sess_order", AgentName: "emotion_analyst", Round: 1, Stance: "analysis", Content: "round1", Confidence: 0.5},
		{ID: "om2", SessionID: "sess_order", AgentName: "opportunity_miner", Round: 1, Stance: "analysis", Content: "round1", Confidence: 0.5},
	}

	if err := store.SaveForumMessages(ctx, msgs); err != nil {
		t.Fatalf("save messages: %v", err)
	}

	got, err := store.GetForumMessages(ctx, "sess_order")
	if err != nil {
		t.Fatalf("get messages: %v", err)
	}

	// Should be ordered by round ASC
	if got[0].Round != 1 {
		t.Fatalf("first message should be round 1, got %d", got[0].Round)
	}
	if got[len(got)-1].Round != 2 {
		t.Fatalf("last message should be round 2, got %d", got[len(got)-1].Round)
	}
}
