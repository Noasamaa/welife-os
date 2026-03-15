package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/welife-os/welife-os/engine/internal/llm"
	"github.com/welife-os/welife-os/engine/internal/storage"
)

// Agent defines the interface for all analysis agents in the WeLife OS system.
// Each agent can independently analyze conversation data and participate in debates.
type Agent interface {
	// Name returns the unique identifier for this agent.
	Name() string

	// Analyze performs independent analysis on conversation data.
	Analyze(ctx context.Context, input AnalysisInput) (AnalysisOutput, error)

	// Debate generates a response given the current debate state.
	Debate(ctx context.Context, state DebateState) (ForumMessage, error)
}

// AnalysisInput contains the data provided to an agent for analysis.
type AnalysisInput struct {
	ConversationID string
	Messages       []storage.StoredMessage
	Entities       []storage.Entity
	Relationships  []storage.Relationship
}

// AnalysisOutput holds the results of an agent's analysis.
type AnalysisOutput struct {
	AgentName string         `json:"agent_name"`
	Summary   string         `json:"summary"`
	Details   []Finding      `json:"details"`
	Data      map[string]any `json:"data,omitempty"`
}

// Finding represents a single insight discovered by an agent.
type Finding struct {
	Type       string   `json:"type"`
	Title      string   `json:"title"`
	Content    string   `json:"content"`
	Evidence   []string `json:"evidence"`
	Confidence float64  `json:"confidence"`
}

// DebateState captures the current state of an ongoing debate session.
type DebateState struct {
	SessionID  string
	Round      int
	Topic      string
	History    []ForumMessage
	MyPrior    *AnalysisOutput
	OtherViews []AnalysisOutput
}

// ForumMessage represents a single message in a debate session.
type ForumMessage struct {
	AgentName  string   `json:"agent_name"`
	Round      int      `json:"round"`
	Stance     string   `json:"stance"`
	Content    string   `json:"content"`
	Evidence   []string `json:"evidence,omitempty"`
	Confidence float64  `json:"confidence"`
}

// debateHelper builds a debate prompt and sends it to the LLM, returning a ForumMessage.
// This is shared across all agents to avoid code duplication in Debate methods.
func debateHelper(ctx context.Context, client llm.LLMClient, agentName, roleDescription string, state DebateState) (ForumMessage, error) {
	var sb strings.Builder
	fmt.Fprintf(&sb, "你是%s，正在参与一场多视角辩论。\n\n", roleDescription)
	fmt.Fprintf(&sb, "当前辩论议题：%s\n\n", state.Topic)

	if state.MyPrior != nil {
		fmt.Fprintf(&sb, "你之前的分析结论：%s\n\n", state.MyPrior.Summary)
	}

	if len(state.OtherViews) > 0 {
		sb.WriteString("其他 Agent 的观点：\n")
		for _, view := range state.OtherViews {
			fmt.Fprintf(&sb, "- %s: %s\n", view.AgentName, view.Summary)
		}
		sb.WriteString("\n")
	}

	if len(state.History) > 0 {
		sb.WriteString("之前的辩论记录：\n")
		for _, msg := range state.History {
			fmt.Fprintf(&sb, "- [%s] %s\n", msg.AgentName, msg.Content)
		}
		sb.WriteString("\n")
	}

	fmt.Fprintf(&sb, `请从%s的角度回应，输出 JSON：
{"stance": "你的立场", "content": "你的论点", "evidence": ["支撑证据"], "confidence": 0.8}

请只输出 JSON。`, roleDescription)

	response, err := client.Generate(ctx, sb.String())
	if err != nil {
		return ForumMessage{}, fmt.Errorf("LLM debate: %w", err)
	}

	jsonStr := llm.ExtractJSON(response)
	var msg ForumMessage
	if err := json.Unmarshal([]byte(jsonStr), &msg); err != nil {
		return ForumMessage{}, fmt.Errorf("parsing debate response: %w", err)
	}

	msg.AgentName = agentName
	msg.Round = state.Round
	return msg, nil
}
