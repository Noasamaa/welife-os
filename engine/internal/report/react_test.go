package report_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/welife-os/welife-os/engine/internal/llm"
	"github.com/welife-os/welife-os/engine/internal/report"
	"github.com/welife-os/welife-os/engine/internal/storage"
)

func newMockOllamaServer(t *testing.T, responseBody string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/generate" {
			w.Header().Set("Content-Type", "application/json")
			resp := map[string]any{
				"model":      "test",
				"response":   responseBody,
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

func newTestLLMClient(t *testing.T, server *httptest.Server) *llm.Client {
	t.Helper()
	client, err := llm.New(llm.Config{
		BaseURL: server.URL,
		Model:   "test",
	})
	if err != nil {
		t.Fatalf("create LLM client: %v", err)
	}
	return client
}

func newTestStore(t *testing.T) *storage.Store {
	t.Helper()
	store, err := storage.Open(context.Background(), storage.Config{
		Path: t.TempDir() + "/react_test.db",
		Key:  "test-key",
	})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

// mockTool is a simple test tool that returns a fixed response.
type mockTool struct {
	name     string
	response string
}

func (m *mockTool) Name() string        { return m.name }
func (m *mockTool) Description() string { return "test tool" }
func (m *mockTool) Execute(_ context.Context, _ map[string]string) (string, error) {
	return m.response, nil
}

type captureTool struct {
	name   string
	params map[string]string
}

func (c *captureTool) Name() string        { return c.name }
func (c *captureTool) Description() string { return "capture tool" }
func (c *captureTool) Execute(_ context.Context, params map[string]string) (string, error) {
	c.params = params
	return `{"ok":true}`, nil
}

func TestReactAgentFinishesImmediately(t *testing.T) {
	// LLM returns a finished response on first iteration
	finishedResp := `{"thought": "已有足够数据", "action": "finish", "narrative": "测试叙述", "data": null, "finished": true}`
	server := newMockOllamaServer(t, finishedResp)
	defer server.Close()

	client := newTestLLMClient(t, server)
	agent := report.NewReactAgent(client, []report.Tool{})

	plan := report.SectionPlan{
		Title: "测试章节",
		Type:  "text",
		Hints: "测试",
	}

	section, err := agent.GenerateSection(context.Background(), plan, report.ToolScope{
		ConversationID: "conv_001",
		Period: report.ReportPeriod{
			Start: "2026-03-09T00:00:00Z",
			End:   "2026-03-15T23:59:59Z",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if section.Title != "测试章节" {
		t.Fatalf("expected title '测试章节', got %s", section.Title)
	}
	if section.Narrative != "测试叙述" {
		t.Fatalf("expected narrative '测试叙述', got %s", section.Narrative)
	}
}

func TestReactAgentCallsTool(t *testing.T) {
	// LLM first calls a tool, then finishes
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/generate" {
			callCount++
			var resp string
			if callCount == 1 {
				resp = `{"thought": "需要搜索消息", "action": "test_tool", "params": {"keyword": "test"}, "finished": false}`
			} else {
				resp = `{"thought": "数据充分", "action": "finish", "narrative": "基于工具数据的叙述", "finished": true}`
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"model": "test", "response": resp, "done": true, "created_at": "2024-01-01T00:00:00Z",
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
	defer server.Close()

	client := newTestLLMClient(t, server)
	tools := []report.Tool{&mockTool{name: "test_tool", response: `{"data": "mock"}`}}
	agent := report.NewReactAgent(client, tools)

	section, err := agent.GenerateSection(context.Background(), report.SectionPlan{
		Title: "工具测试", Type: "text", Hints: "测试",
	}, report.ToolScope{
		ConversationID: "conv_001",
		Period: report.ReportPeriod{
			Start: "2026-03-09T00:00:00Z",
			End:   "2026-03-15T23:59:59Z",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if section.Narrative != "基于工具数据的叙述" {
		t.Fatalf("unexpected narrative: %s", section.Narrative)
	}
	if callCount < 2 {
		t.Fatalf("expected at least 2 LLM calls, got %d", callCount)
	}
}

func TestReactAgentContextCancel(t *testing.T) {
	server := newMockOllamaServer(t, `{}`)
	defer server.Close()

	client := newTestLLMClient(t, server)
	agent := report.NewReactAgent(client, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := agent.GenerateSection(ctx, report.SectionPlan{
		Title: "取消测试", Type: "text", Hints: "测试",
	}, report.ToolScope{
		ConversationID: "conv_001",
		Period: report.ReportPeriod{
			Start: "2026-01-01T00:00:00Z",
			End:   "2026-03-15T23:59:59Z",
		},
	})
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestReactAgentScopesToolCalls(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/generate" {
			callCount++
			var resp string
			if callCount == 1 {
				resp = `{"thought": "先查消息", "action": "message_search", "params": {"conversation_id": "wrong", "after": "2026-01-01T00:00:00Z"}, "finished": false}`
			} else {
				resp = `{"thought": "完成", "action": "finish", "narrative": "完成", "finished": true}`
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"model": "test", "response": resp, "done": true, "created_at": "2024-01-01T00:00:00Z",
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
	defer server.Close()

	client := newTestLLMClient(t, server)
	tool := &captureTool{name: "message_search"}
	agent := report.NewReactAgent(client, []report.Tool{tool})

	_, err := agent.GenerateSection(context.Background(), report.SectionPlan{
		Title: "作用域测试", Type: "text", Hints: "测试",
	}, report.ToolScope{
		ConversationID: "conv_scoped",
		Period: report.ReportPeriod{
			Start: "2026-03-09T00:00:00Z",
			End:   "2026-03-15T23:59:59Z",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tool.params["conversation_id"] != "conv_scoped" {
		t.Fatalf("expected scoped conversation_id, got %q", tool.params["conversation_id"])
	}
	if tool.params["after"] != "2026-03-09T00:00:00Z" {
		t.Fatalf("expected scoped after, got %q", tool.params["after"])
	}
	if tool.params["before"] != "2026-03-15T23:59:59Z" {
		t.Fatalf("expected scoped before, got %q", tool.params["before"])
	}
}
