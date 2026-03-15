package report

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/welife-os/welife-os/engine/internal/llm"
)

const maxReactIterations = 5

// ReactAgent implements the ReACT (Reasoning + Acting + Tool-use) loop
// for generating individual report sections.
type ReactAgent struct {
	llm   llm.LLMClient
	tools map[string]Tool
}

// NewReactAgent creates a new ReACT agent with the given tools.
func NewReactAgent(llmClient llm.LLMClient, tools []Tool) *ReactAgent {
	toolMap := make(map[string]Tool, len(tools))
	for _, t := range tools {
		toolMap[t.Name()] = t
	}
	return &ReactAgent{llm: llmClient, tools: toolMap}
}

const reactPrompt = `你是一个人生报告生成专家。你需要为报告的一个章节生成内容。

## 对话范围
对话ID: %s

## 当前章节
标题: %s
类型: %s
图表类型: %s

## 时间范围
起始: %s
结束: %s

## 章节要求
%s

## 可用工具
%s

## 对话历史
%s

## 指令
你需要通过调用工具获取数据，然后基于数据生成章节内容。
系统会自动把工具调用限定在上述对话范围内；消息工具还会自动限定在给定时间范围内。

如果你需要获取更多数据，输出 JSON：
{"thought": "你的思考", "action": "工具名称", "params": {"参数名": "参数值"}, "finished": false}

如果你已经有足够的数据来生成章节内容，输出 JSON：
{"thought": "总结思考", "action": "finish", "narrative": "章节叙述文本", "data": null, "items": null, "finished": true}

注意：
- data 字段用于 chart 类型章节，应该是 ECharts option 格式的 JSON 对象
- items 字段用于 list 类型章节，应该是对象数组
- narrative 字段是章节的文字说明，所有类型章节都需要
- 请只输出 JSON，不要输出其他内容`

// reactResponse is the expected JSON structure from the LLM in each ReACT iteration.
type reactResponse struct {
	Thought   string            `json:"thought"`
	Action    string            `json:"action"`
	Params    map[string]string `json:"params,omitempty"`
	Narrative string            `json:"narrative,omitempty"`
	Data      any               `json:"data,omitempty"`
	Items     []any             `json:"items,omitempty"`
	Finished  bool              `json:"finished"`
}

// GenerateSection runs the ReACT loop to generate a single report section.
func (a *ReactAgent) GenerateSection(ctx context.Context, plan SectionPlan, scope ToolScope) (Section, error) {
	toolDescs := a.toolDescriptions()
	var history strings.Builder

	for i := 0; i < maxReactIterations; i++ {
		chartType := plan.ChartType
		if chartType == "" {
			chartType = "无"
		}

		prompt := fmt.Sprintf(reactPrompt,
			scope.ConversationID,
			plan.Title, plan.Type, chartType,
			scope.Period.Start, scope.Period.End,
			plan.Hints,
			toolDescs,
			history.String(),
		)

		response, err := a.llm.Generate(ctx, prompt)
		if err != nil {
			return Section{}, fmt.Errorf("LLM generate (iteration %d): %w", i, err)
		}

		jsonStr := llm.ExtractJSON(response)
		var resp reactResponse
		if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
			return Section{}, fmt.Errorf("parsing ReACT response (iteration %d): %w", i, err)
		}

		// Record thought in history
		fmt.Fprintf(&history, "思考 %d: %s\n", i+1, resp.Thought)

		if resp.Finished || resp.Action == "finish" {
			return Section{
				Title:     plan.Title,
				Type:      plan.Type,
				ChartType: plan.ChartType,
				Data:      resp.Data,
				Items:     resp.Items,
				Narrative: resp.Narrative,
			}, nil
		}

		// Execute tool
		tool, ok := a.tools[resp.Action]
		if !ok {
			fmt.Fprintf(&history, "错误: 未知工具 %q，可用工具: %s\n", resp.Action, toolDescs)
			continue
		}

		observation, err := tool.Execute(ctx, scopedToolParams(resp.Action, resp.Params, scope))
		if err != nil {
			fmt.Fprintf(&history, "动作: %s(%v)\n观察: 工具执行失败: %s\n\n", resp.Action, resp.Params, err.Error())
			continue
		}

		// Truncate long observations to keep context manageable
		if len(observation) > 3000 {
			observation = observation[:3000] + "...(已截断)"
		}

		fmt.Fprintf(&history, "动作: %s(%v)\n观察: %s\n\n", resp.Action, resp.Params, observation)
	}

	// Max iterations reached - generate section with whatever data we have
	return a.fallbackSection(ctx, plan, history.String())
}

func scopedToolParams(action string, params map[string]string, scope ToolScope) map[string]string {
	scoped := make(map[string]string, len(params)+4)
	for k, v := range params {
		scoped[k] = v
	}

	if scope.ConversationID == "" {
		return scoped
	}

	switch action {
	case "graph_search":
		scoped["conversation_id"] = scope.ConversationID
	case "forum_search":
		scoped["conversation_id"] = scope.ConversationID
		if scope.Period.Start != "" {
			scoped["after"] = scope.Period.Start
		}
		if scope.Period.End != "" {
			scoped["before"] = scope.Period.End
		}
	case "message_search":
		scoped["conversation_id"] = scope.ConversationID
		if scope.Period.Start != "" {
			scoped["after"] = scope.Period.Start
		}
		if scope.Period.End != "" {
			scoped["before"] = scope.Period.End
		}
	}

	return scoped
}

// fallbackSection generates a text-only section when the ReACT loop exhausts iterations.
func (a *ReactAgent) fallbackSection(ctx context.Context, plan SectionPlan, history string) (Section, error) {
	prompt := fmt.Sprintf(`你是报告生成专家。ReACT 循环已达到最大迭代次数。
请基于以下已收集的信息，直接生成章节内容。

章节标题: %s
已收集信息:
%s

请输出 JSON：
{"narrative": "章节叙述文本"}

请只输出 JSON。`, plan.Title, history)

	response, err := a.llm.Generate(ctx, prompt)
	if err != nil {
		return Section{
			Title:     plan.Title,
			Type:      "text",
			Narrative: "数据收集不完整，无法生成此章节。",
		}, nil
	}

	jsonStr := llm.ExtractJSON(response)
	var result struct {
		Narrative string `json:"narrative"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return Section{
			Title:     plan.Title,
			Type:      "text",
			Narrative: "数据收集不完整，无法生成此章节。",
		}, nil
	}

	return Section{
		Title:     plan.Title,
		Type:      "text",
		Narrative: result.Narrative,
	}, nil
}

func (a *ReactAgent) toolDescriptions() string {
	var sb strings.Builder
	for _, t := range a.tools {
		fmt.Fprintf(&sb, "- %s: %s\n", t.Name(), t.Description())
	}
	return sb.String()
}
