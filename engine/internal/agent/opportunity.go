package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/welife-os/welife-os/engine/internal/llm"
	"github.com/welife-os/welife-os/engine/internal/storage"
)

const opportunityAgentName = "opportunity_miner"

// OpportunityAgent scans conversations for projects, jobs, collaborations,
// and identifies unfollowed items using graph relationships.
type OpportunityAgent struct {
	llm *llm.Client
}

// NewOpportunityAgent creates a new opportunity miner agent.
func NewOpportunityAgent(llmClient *llm.Client) *OpportunityAgent {
	return &OpportunityAgent{llm: llmClient}
}

func (a *OpportunityAgent) Name() string { return opportunityAgentName }

const opportunityAnalysisPrompt = `你是一位人脉与机会分析专家。请从以下对话和知识图谱数据中挖掘潜在机会。

分析要求：
1. 识别对话中提到的项目、职位、合作机会、活动邀请
2. 发现「说了但没跟进」的事项（承诺、约定、待办）
3. 基于实体关系发现跨对话的潜在关联

知识图谱实体：
%s

知识图谱关系：
%s

对话记录：
%s

请以 JSON 格式输出：
{
  "opportunities": [
    {"type": "project|job|collaboration|event", "title": "标题", "description": "描述", "related_people": ["人名"], "message_ids": ["消息ID"], "urgency": "high|medium|low"}
  ],
  "missed_followups": [
    {"what": "承诺/约定内容", "who": "相关人", "when_mentioned": "提及时间", "message_id": "消息ID"}
  ],
  "cross_links": [
    {"entity_a": "实体A", "entity_b": "实体B", "connection": "关联描述"}
  ],
  "summary": "机会分析摘要"
}

请只输出 JSON，不要输出其他内容。`

type opportunityLLMResponse struct {
	Opportunities  []opportunityItem `json:"opportunities"`
	MissedFollowup []missedFollowup  `json:"missed_followups"`
	CrossLinks     []crossLink       `json:"cross_links"`
	Summary        string            `json:"summary"`
}

type opportunityItem struct {
	Type          string   `json:"type"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	RelatedPeople []string `json:"related_people"`
	MessageIDs    []string `json:"message_ids"`
	Urgency       string   `json:"urgency"`
}

type missedFollowup struct {
	What          string `json:"what"`
	Who           string `json:"who"`
	WhenMentioned string `json:"when_mentioned"`
	MessageID     string `json:"message_id"`
}

type crossLink struct {
	EntityA    string `json:"entity_a"`
	EntityB    string `json:"entity_b"`
	Connection string `json:"connection"`
}

func (a *OpportunityAgent) Analyze(ctx context.Context, input AnalysisInput) (AnalysisOutput, error) {
	if len(input.Messages) == 0 {
		return AnalysisOutput{AgentName: opportunityAgentName, Summary: "没有消息可供分析"}, nil
	}

	entitiesStr := formatEntities(input.Entities)
	relsStr := formatRelationships(input.Relationships)
	msgsStr := formatMessages(input.Messages)

	prompt := fmt.Sprintf(opportunityAnalysisPrompt, entitiesStr, relsStr, msgsStr)

	response, err := a.llm.Generate(ctx, prompt)
	if err != nil {
		return AnalysisOutput{}, fmt.Errorf("LLM generate: %w", err)
	}

	jsonStr := llm.ExtractJSON(response)
	var result opportunityLLMResponse
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return AnalysisOutput{}, fmt.Errorf("parsing opportunity response: %w", err)
	}

	var findings []Finding

	for _, opp := range result.Opportunities {
		confidence := urgencyToConfidence(opp.Urgency)
		findings = append(findings, Finding{
			Type:       "opportunity",
			Title:      fmt.Sprintf("[%s] %s", opp.Type, opp.Title),
			Content:    opp.Description,
			Evidence:   opp.MessageIDs,
			Confidence: confidence,
		})
	}

	for _, mf := range result.MissedFollowup {
		findings = append(findings, Finding{
			Type:       "missed_followup",
			Title:      fmt.Sprintf("未跟进: %s", mf.What),
			Content:    fmt.Sprintf("%s 于 %s 提到，尚未跟进", mf.Who, mf.WhenMentioned),
			Evidence:   []string{mf.MessageID},
			Confidence: 0.6,
		})
	}

	return AnalysisOutput{
		AgentName: opportunityAgentName,
		Summary:   result.Summary,
		Details:   findings,
		Data: map[string]any{
			"opportunities":    result.Opportunities,
			"missed_followups": result.MissedFollowup,
			"cross_links":      result.CrossLinks,
		},
	}, nil
}

func (a *OpportunityAgent) Debate(ctx context.Context, state DebateState) (ForumMessage, error) {
	return debateHelper(ctx, a.llm, opportunityAgentName, "机会挖掘师", state)
}

func formatEntities(entities []storage.Entity) string {
	if len(entities) == 0 {
		return "（无实体数据）"
	}
	var sb strings.Builder
	for _, e := range entities {
		fmt.Fprintf(&sb, "- [%s] %s\n", e.Type, e.Name)
	}
	return sb.String()
}

func formatRelationships(rels []storage.Relationship) string {
	if len(rels) == 0 {
		return "（无关系数据）"
	}
	var sb strings.Builder
	for _, r := range rels {
		fmt.Fprintf(&sb, "- %s -[%s]-> %s (权重: %.1f)\n", r.SourceEntityID, r.Type, r.TargetEntityID, r.Weight)
	}
	return sb.String()
}

func formatMessages(msgs []storage.StoredMessage) string {
	var sb strings.Builder
	for _, m := range msgs {
		fmt.Fprintf(&sb, "[%s] %s (ID:%s): %s\n", m.Timestamp, m.SenderName, m.ID, m.Content)
	}
	return sb.String()
}

func urgencyToConfidence(urgency string) float64 {
	switch urgency {
	case "high":
		return 0.9
	case "medium":
		return 0.7
	case "low":
		return 0.5
	default:
		return 0.5
	}
}
