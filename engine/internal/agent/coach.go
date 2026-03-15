package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/welife-os/welife-os/engine/internal/llm"
	"github.com/welife-os/welife-os/engine/internal/storage"
)

// actionItemSeq provides a monotonically increasing counter for unique IDs.
var actionItemSeq uint64

const coachAgentName = "execution_coach"

// CoachAgent extracts actionable items from conversations and debate sessions.
// It identifies projects to follow up, people to contact, and decisions to make.
type CoachAgent struct {
	llm   llm.LLMClient
	store *storage.Store
}

// NewCoachAgent creates a new execution coach agent.
func NewCoachAgent(llmClient llm.LLMClient, store *storage.Store) *CoachAgent {
	return &CoachAgent{llm: llmClient, store: store}
}

func (a *CoachAgent) Name() string { return coachAgentName }

const coachAnalysisPrompt = `你是一位执行力教练。请从以下对话中提取可执行的行动项。

分析要求：
1. 识别需要跟进的项目或任务
2. 找出需要联系的人和具体原因
3. 提取需要做出的决策及其截止时间
4. 标注优先级（high/medium/low）和分类（project/contact/decision/followup）

已存在的行动项（避免重复）：
%s

对话记录：
%s

请以 JSON 格式输出：
{
  "action_items": [
    {"title": "标题", "description": "描述", "priority": "high|medium|low", "category": "project|contact|decision|followup", "due_date": "YYYY-MM-DD或空"}
  ],
  "summary": "行动项分析摘要"
}

请只输出 JSON，不要输出其他内容。`

const coachActionPlanPrompt = `你是一位执行力教练。请根据以下辩论总结和讨论内容，提取具体的行动计划。

辩论总结：
%s

辩论消息记录：
%s

请提取所有可行动的事项，包括：
1. 需要立即执行的任务
2. 需要跟进的事项
3. 需要联系的人
4. 需要做出的决策

请以 JSON 格式输出：
{
  "action_items": [
    {"title": "标题", "description": "详细描述", "priority": "high|medium|low", "category": "project|contact|decision|followup", "due_date": "YYYY-MM-DD或空"}
  ]
}

请只输出 JSON，不要输出其他内容。`

// coachLLMResponse is the expected JSON structure from the LLM.
type coachLLMResponse struct {
	ActionItems []coachActionItem `json:"action_items"`
	Summary     string            `json:"summary"`
}

type coachActionItem struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
	Category    string `json:"category"`
	DueDate     string `json:"due_date"`
}

func (a *CoachAgent) Analyze(ctx context.Context, input AnalysisInput) (AnalysisOutput, error) {
	if len(input.Messages) == 0 {
		return AnalysisOutput{AgentName: coachAgentName, Summary: "没有消息可供分析"}, nil
	}

	// Query existing action items to avoid duplicates
	existingStr := "（无已有行动项）"
	if a.store != nil {
		existing, err := a.store.ListActionItems(ctx, "", "")
		if err == nil && len(existing) > 0 {
			var sb strings.Builder
			for _, item := range existing {
				fmt.Fprintf(&sb, "- [%s] %s: %s\n", item.Priority, item.Title, item.Description)
			}
			existingStr = sb.String()
		}
	}

	msgsStr := formatMessages(input.Messages)
	prompt := fmt.Sprintf(coachAnalysisPrompt, existingStr, msgsStr)

	response, err := a.llm.Generate(ctx, prompt)
	if err != nil {
		return AnalysisOutput{}, fmt.Errorf("LLM generate: %w", err)
	}

	jsonStr := llm.ExtractJSON(response)
	var result coachLLMResponse
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return AnalysisOutput{}, fmt.Errorf("parsing coach response: %w", err)
	}

	var findings []Finding
	for _, item := range result.ActionItems {
		findings = append(findings, Finding{
			Type:       "action_item",
			Title:      fmt.Sprintf("[%s][%s] %s", item.Priority, item.Category, item.Title),
			Content:    item.Description,
			Evidence:   []string{},
			Confidence: priorityToConfidence(item.Priority),
		})
	}

	return AnalysisOutput{
		AgentName: coachAgentName,
		Summary:   result.Summary,
		Details:   findings,
		Data: map[string]any{
			"action_items": result.ActionItems,
		},
	}, nil
}

func (a *CoachAgent) Debate(ctx context.Context, state DebateState) (ForumMessage, error) {
	return debateHelper(ctx, a.llm, coachAgentName, "执行教练", state)
}

// GenerateActionPlan loads a forum session's summary and messages, sends them
// to the LLM to extract action items, saves them to storage, and returns them.
func (a *CoachAgent) GenerateActionPlan(ctx context.Context, sessionID string) ([]storage.ActionItem, error) {
	session, err := a.store.GetSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("loading session: %w", err)
	}

	forumMsgs, err := a.store.GetForumMessages(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("loading forum messages: %w", err)
	}

	var msgSB strings.Builder
	for _, m := range forumMsgs {
		fmt.Fprintf(&msgSB, "[第%d轮][%s] %s: %s\n", m.Round, m.Stance, m.AgentName, m.Content)
	}

	prompt := fmt.Sprintf(coachActionPlanPrompt, session.Summary, msgSB.String())

	response, err := a.llm.Generate(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM generate: %w", err)
	}

	jsonStr := llm.ExtractJSON(response)
	var result coachLLMResponse
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("parsing action plan response: %w", err)
	}

	var items []storage.ActionItem
	for _, ai := range result.ActionItems {
		item := storage.ActionItem{
			ID:              generateID(),
			SourceAgent:     coachAgentName,
			SourceSessionID: sessionID,
			Title:           ai.Title,
			Description:     ai.Description,
			Priority:        normalizePriority(ai.Priority),
			Status:          "pending",
			Category:        normalizeCategory(ai.Category),
			DueDate:         ai.DueDate,
		}
		if err := a.store.CreateActionItem(ctx, item); err != nil {
			return nil, fmt.Errorf("saving action item %q: %w", item.Title, err)
		}
		items = append(items, item)
	}

	return items, nil
}

func priorityToConfidence(priority string) float64 {
	switch priority {
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

func normalizePriority(p string) string {
	switch p {
	case "high", "medium", "low":
		return p
	default:
		return "medium"
	}
}

func normalizeCategory(c string) string {
	switch c {
	case "project", "contact", "decision", "followup":
		return c
	default:
		return "general"
	}
}

func generateID() string {
	seq := atomic.AddUint64(&actionItemSeq, 1)
	return fmt.Sprintf("action_%d_%d", time.Now().UnixNano(), seq)
}
