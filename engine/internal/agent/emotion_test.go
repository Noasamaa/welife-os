package agent_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/welife-os/welife-os/engine/internal/agent"
	"github.com/welife-os/welife-os/engine/internal/llm"
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
		// /api/tags for health check
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
		Timeout: 0, // no timeout for tests
	})
	if err != nil {
		t.Fatalf("failed to create LLM client: %v", err)
	}
	return client
}

func TestEmotionAgentName(t *testing.T) {
	server := newMockOllamaServer(t, "{}")
	defer server.Close()

	client := newTestLLMClient(t, server)
	a := agent.NewEmotionAgent(client)

	if a.Name() != "emotion_analyst" {
		t.Fatalf("expected name emotion_analyst, got %s", a.Name())
	}
}

func TestEmotionAgentAnalyzeEmpty(t *testing.T) {
	server := newMockOllamaServer(t, "{}")
	defer server.Close()

	client := newTestLLMClient(t, server)
	a := agent.NewEmotionAgent(client)

	out, err := a.Analyze(context.Background(), agent.AnalysisInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.AgentName != "emotion_analyst" {
		t.Fatalf("expected agent name emotion_analyst, got %s", out.AgentName)
	}
}

func TestEmotionAgentAnalyze(t *testing.T) {
	mockResponse := `{
		"emotion_timeline": [
			{"message_id": "msg1", "emotion": "积极", "intensity": 0.8, "note": "开心"}
		],
		"emotion_shifts": [
			{"from_message_id": "msg1", "to_message_id": "msg2", "from_emotion": "积极", "to_emotion": "消极", "reason": "话题转变"}
		],
		"relationship_temperature": 72.5,
		"summary": "整体情感偏积极"
	}`
	server := newMockOllamaServer(t, mockResponse)
	defer server.Close()

	client := newTestLLMClient(t, server)
	a := agent.NewEmotionAgent(client)

	input := agent.AnalysisInput{
		ConversationID: "conv1",
		Messages: []storage.StoredMessage{
			{ID: "msg1", SenderName: "Alice", Content: "太好了!", Timestamp: "2024-01-01T10:00:00Z"},
			{ID: "msg2", SenderName: "Alice", Content: "算了吧", Timestamp: "2024-01-01T11:00:00Z"},
		},
	}

	out, err := a.Analyze(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.AgentName != "emotion_analyst" {
		t.Fatalf("expected emotion_analyst, got %s", out.AgentName)
	}
	if len(out.Details) == 0 {
		t.Fatal("expected at least one finding")
	}
	if out.Details[0].Type != "emotion_shift" {
		t.Fatalf("expected emotion_shift, got %s", out.Details[0].Type)
	}
	if out.Data == nil {
		t.Fatal("expected data to be non-nil")
	}
	if _, ok := out.Data["emotion_timeline"]; !ok {
		t.Fatal("expected emotion_timeline in data")
	}
	if _, ok := out.Data["relationship_temperatures"]; !ok {
		t.Fatal("expected relationship_temperatures in data")
	}
}

func TestEmotionAgentDebate(t *testing.T) {
	mockResponse := `{"stance": "关注情绪变化", "content": "从情感分析角度看...", "evidence": ["msg1"], "confidence": 0.85}`
	server := newMockOllamaServer(t, mockResponse)
	defer server.Close()

	client := newTestLLMClient(t, server)
	a := agent.NewEmotionAgent(client)

	state := agent.DebateState{
		SessionID: "sess1",
		Round:     2,
		Topic:     "测试议题",
		History:   []agent.ForumMessage{},
	}

	msg, err := a.Debate(context.Background(), state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.AgentName != "emotion_analyst" {
		t.Fatalf("expected emotion_analyst, got %s", msg.AgentName)
	}
	if msg.Round != 2 {
		t.Fatalf("expected round 2, got %d", msg.Round)
	}
	if msg.Confidence < 0 || msg.Confidence > 1 {
		t.Fatalf("confidence out of range: %f", msg.Confidence)
	}
}

func TestEmotionAgentAnalyzeContextCancel(t *testing.T) {
	server := newMockOllamaServer(t, "{}")
	defer server.Close()

	client := newTestLLMClient(t, server)
	a := agent.NewEmotionAgent(client)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	input := agent.AnalysisInput{
		Messages: []storage.StoredMessage{
			{ID: "msg1", SenderName: "Alice", Content: "hello", Timestamp: "2024-01-01"},
		},
	}
	_, err := a.Analyze(ctx, input)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}
