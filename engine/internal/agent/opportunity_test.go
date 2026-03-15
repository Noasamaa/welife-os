package agent_test

import (
	"context"
	"testing"

	"github.com/welife-os/welife-os/engine/internal/agent"
	"github.com/welife-os/welife-os/engine/internal/storage"
)

func TestOpportunityAgentName(t *testing.T) {
	server := newMockOllamaServer(t, "{}")
	defer server.Close()

	client := newTestLLMClient(t, server)
	a := agent.NewOpportunityAgent(client)

	if a.Name() != "opportunity_miner" {
		t.Fatalf("expected name opportunity_miner, got %s", a.Name())
	}
}

func TestOpportunityAgentAnalyzeEmpty(t *testing.T) {
	server := newMockOllamaServer(t, "{}")
	defer server.Close()

	client := newTestLLMClient(t, server)
	a := agent.NewOpportunityAgent(client)

	out, err := a.Analyze(context.Background(), agent.AnalysisInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.AgentName != "opportunity_miner" {
		t.Fatalf("expected agent name opportunity_miner, got %s", out.AgentName)
	}
}

func TestOpportunityAgentAnalyze(t *testing.T) {
	mockResponse := `{
		"opportunities": [
			{"type": "collaboration", "title": "AI 合作", "description": "可以合作开发AI", "related_people": ["张三"], "message_ids": ["msg1"], "urgency": "high"}
		],
		"missed_followups": [
			{"what": "发送方案", "who": "李四", "when_mentioned": "上周", "message_id": "msg2"}
		],
		"cross_links": [],
		"summary": "发现1个合作机会和1个未跟进事项"
	}`
	server := newMockOllamaServer(t, mockResponse)
	defer server.Close()

	client := newTestLLMClient(t, server)
	a := agent.NewOpportunityAgent(client)

	input := agent.AnalysisInput{
		ConversationID: "conv1",
		Messages: []storage.StoredMessage{
			{ID: "msg1", SenderName: "张三", Content: "我们可以合作开发AI", Timestamp: "2024-01-01"},
			{ID: "msg2", SenderName: "李四", Content: "好的，我下周发方案给你", Timestamp: "2024-01-02"},
		},
		Entities: []storage.Entity{
			{ID: "e1", Type: "person", Name: "张三"},
		},
		Relationships: []storage.Relationship{
			{ID: "r1", SourceEntityID: "e1", TargetEntityID: "e2", Type: "合作", Weight: 1.0},
		},
	}

	out, err := a.Analyze(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.AgentName != "opportunity_miner" {
		t.Fatalf("expected opportunity_miner, got %s", out.AgentName)
	}
	if len(out.Details) < 2 {
		t.Fatalf("expected at least 2 findings (opportunity + missed_followup), got %d", len(out.Details))
	}

	// Check finding types
	typeMap := make(map[string]int)
	for _, f := range out.Details {
		typeMap[f.Type]++
	}
	if typeMap["opportunity"] != 1 {
		t.Fatalf("expected 1 opportunity finding, got %d", typeMap["opportunity"])
	}
	if typeMap["missed_followup"] != 1 {
		t.Fatalf("expected 1 missed_followup finding, got %d", typeMap["missed_followup"])
	}

	if out.Data == nil {
		t.Fatal("expected data to be non-nil")
	}
}

func TestOpportunityAgentDebate(t *testing.T) {
	mockResponse := `{"stance": "发现机会", "content": "从机会挖掘角度看...", "evidence": ["msg1"], "confidence": 0.75}`
	server := newMockOllamaServer(t, mockResponse)
	defer server.Close()

	client := newTestLLMClient(t, server)
	a := agent.NewOpportunityAgent(client)

	state := agent.DebateState{
		SessionID: "sess1",
		Round:     2,
		Topic:     "测试议题",
	}

	msg, err := a.Debate(context.Background(), state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.AgentName != "opportunity_miner" {
		t.Fatalf("expected opportunity_miner, got %s", msg.AgentName)
	}
	if msg.Round != 2 {
		t.Fatalf("expected round 2, got %d", msg.Round)
	}
}
