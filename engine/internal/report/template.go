package report

import (
	"fmt"
	"time"
)

// SectionsForType returns the section plans for a given report type.
func SectionsForType(reportType string) ([]SectionPlan, error) {
	switch reportType {
	case "weekly":
		return weeklyBriefSections(), nil
	case "monthly":
		return monthlyReportSections(), nil
	case "annual":
		return annualReviewSections(), nil
	default:
		return nil, fmt.Errorf("unknown report type: %s", reportType)
	}
}

// DefaultPeriod computes a default time period for the given report type ending at now.
func DefaultPeriod(reportType string) ReportPeriod {
	now := time.Now()
	end := now.Format(time.RFC3339)

	switch reportType {
	case "weekly":
		return ReportPeriod{
			Start: now.AddDate(0, 0, -7).Format(time.RFC3339),
			End:   end,
		}
	case "monthly":
		return ReportPeriod{
			Start: now.AddDate(0, -1, 0).Format(time.RFC3339),
			End:   end,
		}
	case "annual":
		return ReportPeriod{
			Start: now.AddDate(-1, 0, 0).Format(time.RFC3339),
			End:   end,
		}
	default:
		return ReportPeriod{Start: end, End: end}
	}
}

// TitleForType generates a default report title.
func TitleForType(reportType string, period ReportPeriod) string {
	switch reportType {
	case "weekly":
		return fmt.Sprintf("每周人生简报 (%s)", period.Start[:10])
	case "monthly":
		return fmt.Sprintf("每月人生报告 (%s)", period.Start[:7])
	case "annual":
		return fmt.Sprintf("年度人生复盘 (%s)", period.Start[:4])
	default:
		return "人生报告"
	}
}

func weeklyBriefSections() []SectionPlan {
	return []SectionPlan{
		{
			Title:     "本周情绪曲线",
			Type:      "chart",
			ChartType: "line",
			Hints:     "分析本周每天的整体情绪走势，使用 message_search 搜索本周消息，识别情绪变化。生成 ECharts line 图配置，x 轴为日期，y 轴为情绪分数(0-100)。",
		},
		{
			Title:     "人际关系变化",
			Type:      "chart",
			ChartType: "network",
			Hints:     "使用 graph_search 查询本周活跃的人际关系。生成 ECharts graph 图配置，节点为人物，边为互动关系，边的粗细代表互动频率。",
		},
		{
			Title:     "未跟进机会",
			Type:      "list",
			Hints:     "使用 forum_search 查看辩论记录中的机会挖掘结论，结合 message_search 搜索提到的项目、合作、承诺。列出本周提到但未跟进的事项。",
		},
		{
			Title:     "行动建议",
			Type:      "text",
			Hints:     "基于前面章节的分析，提出 3-5 条具体可执行的下周行动建议。包括关系维护、机会跟进、风险防范。",
		},
		{
			Title:     "辩论精华",
			Type:      "text",
			Hints:     "使用 forum_search 获取本周的辩论记录，提取最有价值的洞见和共识摘要。",
		},
	}
}

func monthlyReportSections() []SectionPlan {
	return []SectionPlan{
		{
			Title:     "关系网络变化",
			Type:      "chart",
			ChartType: "network",
			Hints:     "使用 graph_search 查询本月的人际关系图谱。对比月初和月末的关系变化，生成 ECharts graph 图配置。突出新增和减弱的连接。",
		},
		{
			Title:     "情绪趋势分析",
			Type:      "chart",
			ChartType: "line",
			Hints:     "分析本月每周的情绪趋势，使用 message_search 按周分段搜索。生成 ECharts line 图，标注情绪拐点事件。计算月度情感健康度评分(0-100)。",
		},
		{
			Title:     "机会回顾与遗漏",
			Type:      "list",
			Hints:     "使用 forum_search 和 message_search 回顾本月出现的所有机会。按「已跟进」和「遗漏」分类列出，评估每个机会的时效性。",
		},
		{
			Title:     "风险评估摘要",
			Type:      "text",
			Hints:     "使用 forum_search 获取本月的风险辩论记录。综合乐观派/悲观派/现实派的观点，提炼关键风险项和应对建议。",
		},
		{
			Title:     "关键数据统计",
			Type:      "text",
			Hints:     "使用 message_search 统计本月关键数据：消息总量、活跃联系人数、最频繁话题、平均回复速度变化等。",
		},
		{
			Title:     "辩论完整摘要",
			Type:      "text",
			Hints:     "使用 forum_search 获取本月所有辩论会话，生成完整的辩论要点摘要。包括各 Agent 的核心观点和最终共识。",
		},
	}
}

func annualReviewSections() []SectionPlan {
	return []SectionPlan{
		{
			Title:     "年度关系全景图",
			Type:      "chart",
			ChartType: "network",
			Hints:     "使用 graph_search 查询全年的人际关系图谱。生成包含所有重要人物的 ECharts graph 图。节点大小代表互动频率，颜色代表关系类型。",
		},
		{
			Title:     "关键决策节点回顾",
			Type:      "list",
			Hints:     "使用 message_search 搜索全年中涉及重大决策的对话。识别人生转折点、重要选择，列出每个决策节点及其后续影响。",
		},
		{
			Title:     "情绪年轮",
			Type:      "chart",
			ChartType: "heatmap",
			Hints:     "分析全年 12 个月的情绪变化，使用 message_search 按月分段。生成 ECharts heatmap，x 轴为月份，y 轴为周数，颜色代表情绪强度。",
		},
		{
			Title:     "年度人物排行榜",
			Type:      "list",
			Hints:     "使用 graph_search 和 message_search 统计全年互动最多的联系人。按互动频率排序，列出 top 10，附带关系描述和关键互动事件。",
		},
		{
			Title:     "年度关键词云",
			Type:      "text",
			Hints:     "使用 message_search 搜索全年高频话题和关键词。提取年度核心话题，描述每个话题的演变趋势。",
		},
		{
			Title:     "年度总结",
			Type:      "text",
			Hints:     "综合以上所有章节的分析，撰写一份全面的年度人生总结。包括成长与收获、挑战与教训、展望与建议。",
		},
	}
}
