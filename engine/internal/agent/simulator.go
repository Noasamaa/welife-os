package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/welife-os/welife-os/engine/internal/llm"
)

const simulatorAgentName = "future_simulator"

// SimulatorAgent identifies potential fork points in conversation data
// for parallel-life simulations.
type SimulatorAgent struct {
	llm *llm.Client
}

// NewSimulatorAgent creates a new future simulator agent.
func NewSimulatorAgent(llmClient *llm.Client) *SimulatorAgent {
	return &SimulatorAgent{llm: llmClient}
}

func (a *SimulatorAgent) Name() string { return simulatorAgentName }

const simulatorAnalysisPrompt = `你是一位未来模拟分析专家。请分析以下对话，找出其中的关键决策点和人生转折机会。

分析要求：
1. 识别对话中提到的重要决定、选择或机会
2. 评估每个决策点的影响范围和潜在后果
3. 提出可能的"如果当时做了不同选择"的假设

对话中涉及的人物：
%s

对话记录：
%s

请以 JSON 格式输出：
{
  "fork_points": [
    {
      "title": "决策点标题",
      "description": "描述这个决策/选择",
      "affected_people": ["受影响的人"],
      "original_choice": "实际做出的选择",
      "alternative": "另一种可能的选择",
      "potential_impact": "如果选择不同可能的影响",
      "confidence": 0.8
    }
  ],
  "summary": "整体分析摘要"
}

请只输出 JSON。`

type simulatorLLMResponse struct {
	ForkPoints []simulatorForkPoint `json:"fork_points"`
	Summary    string               `json:"summary"`
}

type simulatorForkPoint struct {
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	AffectedPeople  []string `json:"affected_people"`
	OriginalChoice  string   `json:"original_choice"`
	Alternative     string   `json:"alternative"`
	PotentialImpact string   `json:"potential_impact"`
	Confidence      float64  `json:"confidence"`
}

func (a *SimulatorAgent) Analyze(ctx context.Context, input AnalysisInput) (AnalysisOutput, error) {
	if len(input.Messages) == 0 {
		return AnalysisOutput{AgentName: simulatorAgentName, Summary: "没有消息可供分析"}, nil
	}

	// Build entity names list.
	var entityNames []string
	for _, e := range input.Entities {
		entityNames = append(entityNames, e.Name)
	}
	entitiesStr := strings.Join(entityNames, "、")
	if entitiesStr == "" {
		entitiesStr = "未知"
	}

	// Build message transcript.
	var sb strings.Builder
	for _, m := range input.Messages {
		fmt.Fprintf(&sb, "[%s] %s: %s\n", m.Timestamp, m.SenderName, m.Content)
	}

	prompt := fmt.Sprintf(simulatorAnalysisPrompt, entitiesStr, sb.String())

	response, err := a.llm.Generate(ctx, prompt)
	if err != nil {
		return AnalysisOutput{}, fmt.Errorf("LLM generate: %w", err)
	}

	jsonStr := llm.ExtractJSON(response)
	var result simulatorLLMResponse
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return AnalysisOutput{}, fmt.Errorf("parsing simulator response: %w", err)
	}

	var findings []Finding
	for _, fp := range result.ForkPoints {
		findings = append(findings, Finding{
			Type:       "fork_point",
			Title:      fp.Title,
			Content:    fmt.Sprintf("原始选择: %s\n替代选择: %s\n潜在影响: %s", fp.OriginalChoice, fp.Alternative, fp.PotentialImpact),
			Evidence:   fp.AffectedPeople,
			Confidence: fp.Confidence,
		})
	}

	return AnalysisOutput{
		AgentName: simulatorAgentName,
		Summary:   result.Summary,
		Details:   findings,
	}, nil
}

func (a *SimulatorAgent) Debate(ctx context.Context, state DebateState) (ForumMessage, error) {
	return debateHelper(ctx, a.llm, simulatorAgentName, "未来模拟师", state)
}
