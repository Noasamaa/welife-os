package agent_test

import (
	"context"
	"testing"

	"github.com/welife-os/welife-os/engine/internal/agent"
	"github.com/welife-os/welife-os/engine/internal/storage"
)

func TestSimulatorAgentName(t *testing.T) {
	server := newMockOllamaServer(t, "{}")
	defer server.Close()

	client := newTestLLMClient(t, server)
	a := agent.NewSimulatorAgent(client)

	if a.Name() != "future_simulator" {
		t.Fatalf("expected name future_simulator, got %s", a.Name())
	}
}

func TestSimulatorAgentAnalyzeEmpty(t *testing.T) {
	server := newMockOllamaServer(t, "{}")
	defer server.Close()

	client := newTestLLMClient(t, server)
	a := agent.NewSimulatorAgent(client)

	out, err := a.Analyze(context.Background(), agent.AnalysisInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.AgentName != "future_simulator" {
		t.Fatalf("expected agent name future_simulator, got %s", out.AgentName)
	}
}

func TestSimulatorAgentAnalyze(t *testing.T) {
	mockResponse := `{
		"fork_points": [
			{
				"title": "职业选择",
				"description": "是否接受新工作",
				"affected_people": ["Alice"],
				"original_choice": "留在原公司",
				"alternative": "跳槽到新公司",
				"potential_impact": "可能获得更高薪资和发展机会",
				"confidence": 0.85
			}
		],
		"summary": "发现了一个关键的职业决策点"
	}`
	server := newMockOllamaServer(t, mockResponse)
	defer server.Close()

	client := newTestLLMClient(t, server)
	a := agent.NewSimulatorAgent(client)

	input := agent.AnalysisInput{
		ConversationID: "conv1",
		Messages: []storage.StoredMessage{
			{ID: "msg1", SenderName: "Alice", Content: "在考虑要不要跳槽", Timestamp: "2024-01-01T10:00:00Z"},
			{ID: "msg2", SenderName: "Bob", Content: "我觉得你可以试试", Timestamp: "2024-01-01T11:00:00Z"},
		},
		Entities: []storage.Entity{
			{ID: "e1", Name: "Alice", Type: "person"},
			{ID: "e2", Name: "Bob", Type: "person"},
		},
	}

	out, err := a.Analyze(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.AgentName != "future_simulator" {
		t.Fatalf("expected future_simulator, got %s", out.AgentName)
	}
	if len(out.Details) == 0 {
		t.Fatal("expected at least one finding")
	}
	if out.Details[0].Type != "fork_point" {
		t.Fatalf("expected fork_point finding type, got %s", out.Details[0].Type)
	}
	if out.Summary == "" {
		t.Fatal("expected non-empty summary")
	}
}

func TestSimulatorAgentDebate(t *testing.T) {
	mockResponse := `{"stance": "从未来模拟角度分析", "content": "如果做出不同选择...", "evidence": ["msg1"], "confidence": 0.8}`
	server := newMockOllamaServer(t, mockResponse)
	defer server.Close()

	client := newTestLLMClient(t, server)
	a := agent.NewSimulatorAgent(client)

	state := agent.DebateState{
		SessionID: "sess1",
		Round:     1,
		Topic:     "测试议题",
		History:   []agent.ForumMessage{},
	}

	msg, err := a.Debate(context.Background(), state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.AgentName != "future_simulator" {
		t.Fatalf("expected future_simulator, got %s", msg.AgentName)
	}
	if msg.Round != 1 {
		t.Fatalf("expected round 1, got %d", msg.Round)
	}
	if msg.Confidence < 0 || msg.Confidence > 1 {
		t.Fatalf("confidence out of range: %f", msg.Confidence)
	}
}

func TestSimulatorAgentAnalyzeContextCancel(t *testing.T) {
	server := newMockOllamaServer(t, "{}")
	defer server.Close()

	client := newTestLLMClient(t, server)
	a := agent.NewSimulatorAgent(client)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

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
