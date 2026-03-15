package forum

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/welife-os/welife-os/engine/internal/agent"
	"github.com/welife-os/welife-os/engine/internal/llm"
)

// Moderator controls the debate flow: generates topics, manages rounds,
// and produces consensus summaries.
type Moderator struct {
	llm llm.LLMClient
}

// NewModerator creates a new debate moderator.
func NewModerator(llmClient llm.LLMClient) *Moderator {
	return &Moderator{llm: llmClient}
}

const topicGenerationPrompt = `你是一位辩论主持人。以下是三位分析师对同一组对话的独立分析结果。
请从中提取 2-3 个值得深入辩论的核心议题。

%s

请以 JSON 格式输出：
{"topics": ["议题1", "议题2", "议题3"]}

请只输出 JSON。`

// GenerateTopics takes the first-round analysis outputs and generates debate topics.
func (m *Moderator) GenerateTopics(ctx context.Context, analyses []agent.AnalysisOutput) ([]string, error) {
	var sb strings.Builder
	for _, a := range analyses {
		sb.WriteString(fmt.Sprintf("## %s 的分析\n", a.AgentName))
		sb.WriteString(fmt.Sprintf("摘要: %s\n", a.Summary))
		for _, f := range a.Details {
			sb.WriteString(fmt.Sprintf("- [%s] %s: %s\n", f.Type, f.Title, f.Content))
		}
		sb.WriteString("\n")
	}

	prompt := fmt.Sprintf(topicGenerationPrompt, sb.String())

	response, err := m.llm.Generate(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM generate topics: %w", err)
	}

	jsonStr := llm.ExtractJSON(response)
	var result struct {
		Topics []string `json:"topics"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("parsing topics: %w", err)
	}

	return result.Topics, nil
}

const consensusPrompt = `你是辩论主持人。辩论已经结束，请根据所有辩论记录总结共识和关键洞见。

辩论记录：
%s

请以 JSON 格式输出：
{"summary": "共识摘要，包含关键结论和行动建议"}

请只输出 JSON。`

// Summarize produces a consensus summary from all debate messages.
func (m *Moderator) Summarize(ctx context.Context, messages []agent.ForumMessage) (string, error) {
	var sb strings.Builder
	for _, msg := range messages {
		sb.WriteString(fmt.Sprintf("[第%d轮] %s (%s): %s\n", msg.Round, msg.AgentName, msg.Stance, msg.Content))
	}

	prompt := fmt.Sprintf(consensusPrompt, sb.String())

	response, err := m.llm.Generate(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("LLM summarize: %w", err)
	}

	jsonStr := llm.ExtractJSON(response)
	var result struct {
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return "", fmt.Errorf("parsing summary: %w", err)
	}

	return result.Summary, nil
}
