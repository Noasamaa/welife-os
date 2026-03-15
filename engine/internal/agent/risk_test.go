package agent_test

import (
	"context"
	"testing"

	"github.com/welife-os/welife-os/engine/internal/agent"
	"github.com/welife-os/welife-os/engine/internal/storage"
)

func TestRiskAgentName(t *testing.T) {
	server := newMockOllamaServer(t, "{}")
	defer server.Close()

	client := newTestLLMClient(t, server)
	a := agent.NewRiskAgent(client)

	if a.Name() != "risk_debate_team" {
		t.Fatalf("expected name risk_debate_team, got %s", a.Name())
	}
}

func TestRiskAgentAnalyzeEmpty(t *testing.T) {
	server := newMockOllamaServer(t, "{}")
	defer server.Close()

	client := newTestLLMClient(t, server)
	a := agent.NewRiskAgent(client)

	out, err := a.Analyze(context.Background(), agent.AnalysisInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.AgentName != "risk_debate_team" {
		t.Fatalf("expected risk_debate_team, got %s", out.AgentName)
	}
}

func TestRiskAgentAnalyze(t *testing.T) {
	// For the risk agent, 3 sub-roles + 1 moderator = 4 calls.
	// We serve the same response for all since the mock server is stateless.
	// Use a combined response that works for both sub-role and moderator parse paths:
	combinedResp := `{
		"perspective": "realist",
		"risk_items": [{"title": "沟通不足", "severity": "medium", "assessment": "双方缺乏定期沟通", "mitigation": "设定定期沟通机制"}],
		"overall_assessment": "存在中等风险",
		"consensus_risks": [{"title": "沟通风险", "severity": "medium", "consensus": "三方一致认为沟通不足", "action": "设定周会"}],
		"divergences": [],
		"summary": "综合评估：中等风险"
	}`

	server := newMockOllamaServer(t, combinedResp)
	defer server.Close()

	client := newTestLLMClient(t, server)
	a := agent.NewRiskAgent(client)

	input := agent.AnalysisInput{
		ConversationID: "conv1",
		Messages: []storage.StoredMessage{
			{ID: "msg1", SenderName: "Alice", Content: "这个项目进展太慢了", Timestamp: "2024-01-01"},
			{ID: "msg2", SenderName: "Bob", Content: "我们需要加快进度", Timestamp: "2024-01-02"},
		},
	}

	out, err := a.Analyze(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.AgentName != "risk_debate_team" {
		t.Fatalf("expected risk_debate_team, got %s", out.AgentName)
	}

	if len(out.Details) == 0 {
		t.Fatal("expected at least one risk finding")
	}

	if out.Data == nil {
		t.Fatal("expected data to be non-nil")
	}
	if _, ok := out.Data["risk_items"]; !ok {
		t.Fatal("expected risk_items in data")
	}
	if _, ok := out.Data["sub_debates"]; !ok {
		t.Fatal("expected sub_debates in data")
	}
}

func TestRiskAgentDebate(t *testing.T) {
	mockResponse := `{"stance": "综合风险评估", "content": "从风险角度看...", "evidence": ["msg1"], "confidence": 0.7}`
	server := newMockOllamaServer(t, mockResponse)
	defer server.Close()

	client := newTestLLMClient(t, server)
	a := agent.NewRiskAgent(client)

	state := agent.DebateState{
		SessionID: "sess1",
		Round:     3,
		Topic:     "风险评估",
		MyPrior: &agent.AnalysisOutput{
			AgentName: "risk_debate_team",
			Summary:   "存在中等风险",
		},
		OtherViews: []agent.AnalysisOutput{
			{AgentName: "emotion_analyst", Summary: "情感偏负面"},
		},
	}

	msg, err := a.Debate(context.Background(), state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.AgentName != "risk_debate_team" {
		t.Fatalf("expected risk_debate_team, got %s", msg.AgentName)
	}
	if msg.Round != 3 {
		t.Fatalf("expected round 3, got %d", msg.Round)
	}
}

func TestRiskAgentAnalyzeContextCancel(t *testing.T) {
	server := newMockOllamaServer(t, "{}")
	defer server.Close()

	client := newTestLLMClient(t, server)
	a := agent.NewRiskAgent(client)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	input := agent.AnalysisInput{
		Messages: []storage.StoredMessage{
			{ID: "msg1", SenderName: "Alice", Content: "test", Timestamp: "2024-01-01"},
		},
	}
	_, err := a.Analyze(ctx, input)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}
