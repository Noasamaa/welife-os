package report_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/welife-os/welife-os/engine/internal/report"
	"github.com/welife-os/welife-os/engine/internal/storage"
	"github.com/welife-os/welife-os/engine/internal/task"
)

func setupTestGenerator(t *testing.T) (*report.Generator, *storage.Store, *task.Manager) {
	t.Helper()

	// Mock LLM that always returns a finished section
	finishedResp := `{
		"thought": "数据充分",
		"action": "finish",
		"narrative": "测试叙述内容",
		"data": null,
		"finished": true,
		"summary": "测试总结"
	}`
	server := newMockOllamaServer(t, finishedResp)
	t.Cleanup(server.Close)

	client := newTestLLMClient(t, server)
	store := newTestStore(t)

	// Seed conversation
	if err := store.SaveConversation(context.Background(), storage.Conversation{
		ID: "conv_gen", Platform: "test", ConversationType: "private",
		Title: "生成测试", MessageCount: 1,
	}); err != nil {
		t.Fatalf("save conversation: %v", err)
	}

	if err := store.SaveMessages(context.Background(), []storage.StoredMessage{
		{ID: "gm1", ConversationID: "conv_gen", Platform: "test", SenderID: "u1",
			SenderName: "Alice", Content: "测试消息", MessageType: "text", Timestamp: "2026-03-10T10:00:00Z"},
	}); err != nil {
		t.Fatalf("save messages: %v", err)
	}

	taskMgr := task.NewManager(2)
	t.Cleanup(func() { _ = taskMgr.Close() })

	tools := []report.Tool{
		&mockTool{name: "graph_search", response: `{"entities":[],"entity_count":0}`},
		&mockTool{name: "forum_search", response: `{"sessions":[],"session_count":0}`},
		&mockTool{name: "message_search", response: `{"messages":[],"message_count":0}`},
	}

	gen := report.NewGenerator(client, store, taskMgr, tools)
	return gen, store, taskMgr
}

func TestGenerateCreatesReport(t *testing.T) {
	gen, store, _ := setupTestGenerator(t)
	ctx := context.Background()

	reportID, taskID, err := gen.Generate(ctx, report.GenerateRequest{
		Type:           "weekly",
		ConversationID: "conv_gen",
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if reportID == "" {
		t.Fatal("expected non-empty report ID")
	}
	if taskID == "" {
		t.Fatal("expected non-empty task ID")
	}

	// Verify report exists in DB
	rpt, err := store.GetReport(ctx, reportID)
	if err != nil {
		t.Fatalf("get report: %v", err)
	}
	if rpt.Type != "weekly" {
		t.Fatalf("expected weekly, got %s", rpt.Type)
	}
}

func TestGenerateInvalidType(t *testing.T) {
	gen, _, _ := setupTestGenerator(t)
	ctx := context.Background()

	_, _, err := gen.Generate(ctx, report.GenerateRequest{
		Type:           "daily",
		ConversationID: "conv_gen",
	})
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
}

func TestGenerateInvalidConversation(t *testing.T) {
	gen, _, _ := setupTestGenerator(t)
	ctx := context.Background()

	_, _, err := gen.Generate(ctx, report.GenerateRequest{
		Type:           "weekly",
		ConversationID: "nonexistent",
	})
	if err == nil {
		t.Fatal("expected error for nonexistent conversation")
	}
}

func TestGenerateCompletesAsync(t *testing.T) {
	gen, store, taskMgr := setupTestGenerator(t)
	ctx := context.Background()

	reportID, taskID, err := gen.Generate(ctx, report.GenerateRequest{
		Type:           "weekly",
		ConversationID: "conv_gen",
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	// Wait for task to complete
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
		t.Fatalf("generation failed: %s", info.Error)
	}
	if info.Status != task.StatusSucceeded {
		t.Fatalf("expected succeeded, got %s", info.Status)
	}

	// Verify report is completed with content
	rpt, err := store.GetReport(ctx, reportID)
	if err != nil {
		t.Fatalf("get report: %v", err)
	}
	if rpt.Status != "completed" {
		t.Fatalf("expected completed, got %s", rpt.Status)
	}

	// Verify content is valid JSON
	var content report.ReportContent
	if err := json.Unmarshal([]byte(rpt.Content), &content); err != nil {
		t.Fatalf("invalid content JSON: %v", err)
	}
	if len(content.Sections) == 0 {
		t.Fatal("expected sections in report content")
	}
	if content.Type != "weekly" {
		t.Fatalf("expected type weekly in content, got %s", content.Type)
	}
}

func TestListAndDeleteReports(t *testing.T) {
	gen, _, _ := setupTestGenerator(t)
	ctx := context.Background()

	// Initially empty
	reports, err := gen.ListReports(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if reports != nil && len(reports) != 0 {
		t.Fatalf("expected empty, got %d", len(reports))
	}

	// Generate one
	reportID, _, err := gen.Generate(ctx, report.GenerateRequest{
		Type: "monthly", ConversationID: "conv_gen",
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	reports, err = gen.ListReports(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(reports) != 1 {
		t.Fatalf("expected 1, got %d", len(reports))
	}

	// Delete it
	if err := gen.DeleteReport(ctx, reportID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	reports, err = gen.ListReports(ctx)
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if reports != nil && len(reports) != 0 {
		t.Fatalf("expected empty after delete, got %d", len(reports))
	}
}

func TestTemplateSectionsForType(t *testing.T) {
	for _, tt := range []struct {
		name string
		min  int
	}{
		{"weekly", 5},
		{"monthly", 6},
		{"annual", 6},
	} {
		sections, err := report.SectionsForType(tt.name)
		if err != nil {
			t.Fatalf("SectionsForType(%s): %v", tt.name, err)
		}
		if len(sections) < tt.min {
			t.Fatalf("expected at least %d sections for %s, got %d", tt.min, tt.name, len(sections))
		}
	}

	_, err := report.SectionsForType("invalid")
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
}
