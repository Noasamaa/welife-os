package storage_test

import (
	"context"
	"testing"

	"github.com/welife-os/welife-os/engine/internal/storage"
)

func TestCreateAndGetReport(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	r := storage.Report{
		ID:             "rpt_001",
		Type:           "weekly",
		ConversationID: "conv_001",
		TaskID:         "task_001",
		Status:         "running",
		Title:          "测试周报",
		Content:        "{}",
		PeriodStart:    "2026-03-09",
		PeriodEnd:      "2026-03-15",
	}

	if err := store.CreateReport(ctx, r); err != nil {
		t.Fatalf("create report: %v", err)
	}

	got, err := store.GetReport(ctx, "rpt_001")
	if err != nil {
		t.Fatalf("get report: %v", err)
	}
	if got.ID != "rpt_001" {
		t.Fatalf("expected rpt_001, got %s", got.ID)
	}
	if got.Type != "weekly" {
		t.Fatalf("expected weekly, got %s", got.Type)
	}
	if got.Status != "running" {
		t.Fatalf("expected running, got %s", got.Status)
	}
}

func TestUpdateReport(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	if err := store.CreateReport(ctx, storage.Report{
		ID: "rpt_002", Type: "monthly", ConversationID: "conv_001",
		TaskID: "task_002", Status: "running", Title: "月报",
		Content: "{}", PeriodStart: "2026-02-15", PeriodEnd: "2026-03-15",
	}); err != nil {
		t.Fatalf("create: %v", err)
	}

	content := `{"title":"月报","sections":[]}`
	if err := store.UpdateReport(ctx, "rpt_002", "completed", "完成的月报", content); err != nil {
		t.Fatalf("update: %v", err)
	}

	got, err := store.GetReport(ctx, "rpt_002")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != "completed" {
		t.Fatalf("expected completed, got %s", got.Status)
	}
	if got.Title != "完成的月报" {
		t.Fatalf("expected title update, got %s", got.Title)
	}
	if got.CompletedAt == "" {
		t.Fatal("expected completed_at to be set")
	}
}

func TestListReports(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	for i, id := range []string{"rpt_a", "rpt_b"} {
		types := []string{"weekly", "monthly"}
		if err := store.CreateReport(ctx, storage.Report{
			ID: id, Type: types[i], ConversationID: "conv_001",
			TaskID: "task_" + id, Status: "completed", Title: "报告",
			Content: "{}", PeriodStart: "2026-01-01", PeriodEnd: "2026-03-15",
		}); err != nil {
			t.Fatalf("create %s: %v", id, err)
		}
	}

	reports, err := store.ListReports(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(reports) != 2 {
		t.Fatalf("expected 2, got %d", len(reports))
	}
}

func TestDeleteReport(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	if err := store.CreateReport(ctx, storage.Report{
		ID: "rpt_del", Type: "weekly", ConversationID: "conv_001",
		TaskID: "task_del", Status: "completed", Title: "删除测试",
		Content: "{}", PeriodStart: "2026-03-01", PeriodEnd: "2026-03-15",
	}); err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := store.DeleteReport(ctx, "rpt_del"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	_, err := store.GetReport(ctx, "rpt_del")
	if err == nil {
		t.Fatal("expected not found error after delete")
	}
}

func TestDeleteReportNotFound(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	err := store.DeleteReport(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent report")
	}
}

func TestGetReportNotFound(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	_, err := store.GetReport(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSearchMessages(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	// Seed a conversation and messages
	if err := store.SaveConversation(ctx, storage.Conversation{
		ID: "conv_search", Platform: "test", ConversationType: "private",
		Title: "搜索测试", MessageCount: 3,
	}); err != nil {
		t.Fatalf("save conversation: %v", err)
	}

	msgs := []storage.StoredMessage{
		{ID: "s1", ConversationID: "conv_search", Platform: "test", SenderID: "u1", SenderName: "Alice", Content: "今天讨论AI项目", MessageType: "text", Timestamp: "2026-03-10T10:00:00Z"},
		{ID: "s2", ConversationID: "conv_search", Platform: "test", SenderID: "u2", SenderName: "Bob", Content: "好的，我准备方案", MessageType: "text", Timestamp: "2026-03-10T11:00:00Z"},
		{ID: "s3", ConversationID: "conv_search", Platform: "test", SenderID: "u1", SenderName: "Alice", Content: "下周开会确认", MessageType: "text", Timestamp: "2026-03-11T09:00:00Z"},
	}
	if err := store.SaveMessages(ctx, msgs); err != nil {
		t.Fatalf("save messages: %v", err)
	}

	// Search by keyword
	results, err := store.SearchMessages(ctx, storage.MessageSearchParams{Keyword: "AI"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result for 'AI', got %d", len(results))
	}

	// Search by sender
	results, err = store.SearchMessages(ctx, storage.MessageSearchParams{SenderName: "Bob"})
	if err != nil {
		t.Fatalf("search by sender: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result for Bob, got %d", len(results))
	}

	// Search by time range
	results, err = store.SearchMessages(ctx, storage.MessageSearchParams{
		After: "2026-03-11T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("search by time: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result after 3/11, got %d", len(results))
	}

	// Search all
	results, err = store.SearchMessages(ctx, storage.MessageSearchParams{ConversationID: "conv_search"})
	if err != nil {
		t.Fatalf("search all: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
}
