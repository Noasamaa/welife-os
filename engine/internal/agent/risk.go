package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/welife-os/welife-os/engine/internal/llm"
)

const riskAgentName = "risk_debate_team"

// RiskAgent runs 3 sub-perspectives (optimist, pessimist, realist) in parallel,
// then uses a moderator prompt to synthesize a final risk assessment.
type RiskAgent struct {
	llm llm.LLMClient
}

// NewRiskAgent creates a new risk debate team agent.
func NewRiskAgent(llmClient llm.LLMClient) *RiskAgent {
	return &RiskAgent{llm: llmClient}
}

func (a *RiskAgent) Name() string { return riskAgentName }

type subRole struct {
	name   string
	prompt string
}

var subRoles = []subRole{
	{
		name: "optimist",
		prompt: `你是一位乐观派风险分析师。请从积极的角度分析以下对话中的风险因素。
你倾向于：
- 看到风险中的机会
- 认为大多数问题是可控的
- 关注人际关系中的积极信号
- 强调恢复力和适应能力

对话记录：
%s

请以 JSON 格式输出：
{
  "perspective": "optimist",
  "risk_items": [
    {"title": "风险标题", "severity": "low|medium|high", "assessment": "评估说明", "mitigation": "建议应对措施"}
  ],
  "overall_assessment": "整体评估"
}

请只输出 JSON。`,
	},
	{
		name: "pessimist",
		prompt: `你是一位悲观派风险分析师。请从警惕的角度分析以下对话中的风险因素。
你倾向于：
- 识别潜在的隐患和危险信号
- 关注可能恶化的趋势
- 警惕过度承诺和空洞保证
- 强调最坏情况的准备

对话记录：
%s

请以 JSON 格式输出：
{
  "perspective": "pessimist",
  "risk_items": [
    {"title": "风险标题", "severity": "low|medium|high", "assessment": "评估说明", "mitigation": "建议应对措施"}
  ],
  "overall_assessment": "整体评估"
}

请只输出 JSON。`,
	},
	{
		name: "realist",
		prompt: `你是一位现实派风险分析师。请从客观平衡的角度分析以下对话中的风险因素。
你倾向于：
- 基于事实和数据做判断
- 区分实际风险和想象风险
- 权衡概率和影响
- 提供务实的应对建议

对话记录：
%s

请以 JSON 格式输出：
{
  "perspective": "realist",
  "risk_items": [
    {"title": "风险标题", "severity": "low|medium|high", "assessment": "评估说明", "mitigation": "建议应对措施"}
  ],
  "overall_assessment": "整体评估"
}

请只输出 JSON。`,
	},
}

type subRoleResponse struct {
	Perspective       string         `json:"perspective"`
	RiskItems         []riskItem     `json:"risk_items"`
	OverallAssessment string         `json:"overall_assessment"`
}

type riskItem struct {
	Title      string `json:"title"`
	Severity   string `json:"severity"`
	Assessment string `json:"assessment"`
	Mitigation string `json:"mitigation"`
}

const moderatorPrompt = `你是风险评估主持人。以下是三位风险分析师（乐观派、悲观派、现实派）对同一组对话的独立分析。
请综合三方观点，输出最终的风险评估。

乐观派评估：
%s

悲观派评估：
%s

现实派评估：
%s

请以 JSON 格式输出最终综合评估：
{
  "consensus_risks": [
    {"title": "风险标题", "severity": "low|medium|high", "consensus": "三方共识描述", "action": "建议行动"}
  ],
  "divergences": [
    {"topic": "分歧点", "optimist_view": "乐观派看法", "pessimist_view": "悲观派看法", "realist_view": "现实派看法"}
  ],
  "summary": "综合风险评估摘要"
}

请只输出 JSON。`

type moderatorResponse struct {
	ConsensusRisks []consensusRisk `json:"consensus_risks"`
	Divergences    []divergence    `json:"divergences"`
	Summary        string          `json:"summary"`
}

type consensusRisk struct {
	Title     string `json:"title"`
	Severity  string `json:"severity"`
	Consensus string `json:"consensus"`
	Action    string `json:"action"`
}

type divergence struct {
	Topic         string `json:"topic"`
	OptimistView  string `json:"optimist_view"`
	PessimistView string `json:"pessimist_view"`
	RealistView   string `json:"realist_view"`
}

func (a *RiskAgent) Analyze(ctx context.Context, input AnalysisInput) (AnalysisOutput, error) {
	if len(input.Messages) == 0 {
		return AnalysisOutput{AgentName: riskAgentName, Summary: "没有消息可供分析"}, nil
	}

	msgsStr := formatMessages(input.Messages)

	// Run 3 sub-roles in parallel
	subResults := make([]subRoleResponse, len(subRoles))
	var mu sync.Mutex
	var firstErr error
	var wg sync.WaitGroup

	for i, role := range subRoles {
		wg.Add(1)
		go func(idx int, r subRole) {
			defer wg.Done()

			prompt := fmt.Sprintf(r.prompt, msgsStr)
			response, err := a.llm.Generate(ctx, prompt)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("sub-role %s: %w", r.name, err)
				}
				mu.Unlock()
				return
			}

			jsonStr := llm.ExtractJSON(response)
			var result subRoleResponse
			if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("parsing %s response: %w", r.name, err)
				}
				mu.Unlock()
				return
			}

			mu.Lock()
			subResults[idx] = result
			mu.Unlock()
		}(i, role)
	}

	wg.Wait()
	if firstErr != nil {
		return AnalysisOutput{}, firstErr
	}

	// Moderator synthesizes the 3 perspectives
	modResult, err := a.synthesize(ctx, subResults)
	if err != nil {
		return AnalysisOutput{}, fmt.Errorf("moderator synthesis: %w", err)
	}

	var findings []Finding
	for _, risk := range modResult.ConsensusRisks {
		findings = append(findings, Finding{
			Type:       "risk",
			Title:      risk.Title,
			Content:    fmt.Sprintf("%s\n建议行动: %s", risk.Consensus, risk.Action),
			Confidence: severityToConfidence(risk.Severity),
		})
	}

	// Build sub_debates from the raw sub-role results
	subDebates := make([]map[string]any, 0, len(subResults))
	for _, sr := range subResults {
		subDebates = append(subDebates, map[string]any{
			"perspective":        sr.Perspective,
			"risk_items":         sr.RiskItems,
			"overall_assessment": sr.OverallAssessment,
		})
	}

	return AnalysisOutput{
		AgentName: riskAgentName,
		Summary:   modResult.Summary,
		Details:   findings,
		Data: map[string]any{
			"risk_items":   modResult.ConsensusRisks,
			"divergences":  modResult.Divergences,
			"sub_debates":  subDebates,
		},
	}, nil
}

func (a *RiskAgent) synthesize(ctx context.Context, results []subRoleResponse) (moderatorResponse, error) {
	if len(results) < 3 {
		return moderatorResponse{}, fmt.Errorf("expected 3 sub-role results, got %d", len(results))
	}

	parts := make([]string, len(results))
	for i, r := range results {
		data, err := json.Marshal(r)
		if err != nil {
			return moderatorResponse{}, fmt.Errorf("marshalling sub-role %d: %w", i, err)
		}
		parts[i] = string(data)
	}

	prompt := fmt.Sprintf(moderatorPrompt, parts[0], parts[1], parts[2])

	response, err := a.llm.Generate(ctx, prompt)
	if err != nil {
		return moderatorResponse{}, fmt.Errorf("LLM generate: %w", err)
	}

	jsonStr := llm.ExtractJSON(response)
	var result moderatorResponse
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return moderatorResponse{}, fmt.Errorf("parsing moderator response: %w", err)
	}

	return result, nil
}

func (a *RiskAgent) Debate(ctx context.Context, state DebateState) (ForumMessage, error) {
	return debateHelper(ctx, a.llm, riskAgentName, "风险辩论团（综合乐观/悲观/现实三个视角）", state)
}

func severityToConfidence(severity string) float64 {
	switch severity {
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
