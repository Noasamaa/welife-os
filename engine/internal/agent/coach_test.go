package agent_test

import (
	"context"
	"testing"

	"github.com/welife-os/welife-os/engine/internal/agent"
	"github.com/welife-os/welife-os/engine/internal/storage"
)

func TestCoachAgentName(t *testing.T) {
	server := newMockOllamaServer(t, "{}")
	defer server.Close()

	client := newTestLLMClient(t, server)
	a := agent.NewCoachAgent(client, nil)

	if a.Name() != "execution_coach" {
		t.Fatalf("expected name execution_coach, got %s", a.Name())
	}
}

func TestCoachAgentAnalyzeEmpty(t *testing.T) {
	server := newMockOllamaServer(t, "{}")
	defer server.Close()

	client := newTestLLMClient(t, server)
	a := agent.NewCoachAgent(client, nil)

	out, err := a.Analyze(context.Background(), agent.AnalysisInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.AgentName != "execution_coach" {
		t.Fatalf("expected agent name execution_coach, got %s", out.AgentName)
	}
	if out.Summary != "没有消息可供分析" {
		t.Fatalf("expected empty analysis summary, got %s", out.Summary)
	}
}

func TestCoachAgentAnalyze(t *testing.T) {
	mockResponse := `{
		"action_items": [
			{"title": "联系张三讨论项目", "description": "张三提到了一个合作机会需要跟进", "priority": "high", "category": "contact", "due_date": "2024-02-01"},
			{"title": "准备提案文档", "description": "为下周的会议准备项目提案", "priority": "medium", "category": "project", "due_date": ""}
		],
		"summary": "发现2个需要跟进的行动项"
	}`
	server := newMockOllamaServer(t, mockResponse)
	defer server.Close()

	client := newTestLLMClient(t, server)
	a := agent.NewCoachAgent(client, nil)

	input := agent.AnalysisInput{
		ConversationID: "conv1",
		Messages: []storage.StoredMessage{
			{ID: "msg1", SenderName: "Alice", Content: "张三说可以合作那个项目", Timestamp: "2024-01-01T10:00:00Z"},
			{ID: "msg2", SenderName: "Bob", Content: "好的，下周开会讨论", Timestamp: "2024-01-01T11:00:00Z"},
		},
	}

	out, err := a.Analyze(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.AgentName != "execution_coach" {
		t.Fatalf("expected execution_coach, got %s", out.AgentName)
	}
	if len(out.Details) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(out.Details))
	}
	if out.Details[0].Type != "action_item" {
		t.Fatalf("expected action_item type, got %s", out.Details[0].Type)
	}
	if out.Data == nil {
		t.Fatal("expected data to be non-nil")
	}
	if _, ok := out.Data["action_items"]; !ok {
		t.Fatal("expected action_items in data")
	}
}

func TestCoachAgentDebate(t *testing.T) {
	mockResponse := `{"stance": "关注执行落地", "content": "从执行教练角度看，需要制定具体的跟进计划...", "evidence": ["msg1"], "confidence": 0.82}`
	server := newMockOllamaServer(t, mockResponse)
	defer server.Close()

	client := newTestLLMClient(t, server)
	a := agent.NewCoachAgent(client, nil)

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
	if msg.AgentName != "execution_coach" {
		t.Fatalf("expected execution_coach, got %s", msg.AgentName)
	}
	if msg.Round != 1 {
		t.Fatalf("expected round 1, got %d", msg.Round)
	}
	if msg.Confidence < 0 || msg.Confidence > 1 {
		t.Fatalf("confidence out of range: %f", msg.Confidence)
	}
}
