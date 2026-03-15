package report

import (
	"strings"
	"testing"
)

func TestSanitizeSectionDropsInvalidChartData(t *testing.T) {
	section := sanitizeSection(Section{
		Title:     "图表章节",
		Type:      "chart",
		ChartType: "line",
		Data:      []any{"invalid-root"},
		Narrative: "保留叙述",
	})

	if section.Data != nil {
		t.Fatalf("expected invalid chart root to be dropped, got %#v", section.Data)
	}
	if section.Narrative != "保留叙述" {
		t.Fatalf("expected narrative to survive sanitization, got %q", section.Narrative)
	}
}

func TestSanitizeSectionTrimsListItems(t *testing.T) {
	section := sanitizeSection(Section{
		Title: "列表章节",
		Type:  "list",
		Items: []any{
			"  保留文本  ",
			map[string]any{
				"title":       "联系人",
				"description": strings.Repeat("描", maxReportItemRunes+20),
				"ignored":     "不会保留",
			},
			42,
		},
	})

	if len(section.Items) != 2 {
		t.Fatalf("expected 2 sanitized items, got %d", len(section.Items))
	}

	entry, ok := section.Items[1].(map[string]string)
	if !ok {
		t.Fatalf("expected structured list entry, got %#v", section.Items[1])
	}
	if _, exists := entry["ignored"]; exists {
		t.Fatal("expected unknown keys to be dropped")
	}
	if len([]rune(entry["description"])) != maxReportItemRunes {
		t.Fatalf("expected description to be trimmed to %d runes, got %d", maxReportItemRunes, len([]rune(entry["description"])))
	}
}

func TestSanitizeReportContentLimitsSummaryAndSections(t *testing.T) {
	sections := make([]Section, 0, maxReportSections+2)
	for i := 0; i < maxReportSections+2; i++ {
		sections = append(sections, Section{
			Title:     "章节",
			Type:      "text",
			Narrative: "内容",
		})
	}

	content := sanitizeReportContent(ReportContent{
		Title:    strings.Repeat("题", maxReportTitleRunes+10),
		Type:     "weird",
		Period:   ReportPeriod{Start: strings.Repeat("s", maxReportPeriodRunes+10), End: strings.Repeat("e", maxReportPeriodRunes+10)},
		Summary:  strings.Repeat("总", maxReportSummaryRunes+50),
		Sections: sections,
	})

	if content.Type != "weekly" {
		t.Fatalf("expected invalid report type to normalize to weekly, got %q", content.Type)
	}
	if len([]rune(content.Title)) != maxReportTitleRunes {
		t.Fatalf("expected title to be trimmed to %d runes, got %d", maxReportTitleRunes, len([]rune(content.Title)))
	}
	if len([]rune(content.Summary)) != maxReportSummaryRunes {
		t.Fatalf("expected summary to be trimmed to %d runes, got %d", maxReportSummaryRunes, len([]rune(content.Summary)))
	}
	if len(content.Sections) != maxReportSections {
		t.Fatalf("expected sections to be limited to %d, got %d", maxReportSections, len(content.Sections))
	}
}

func TestSanitizeChartDataDropsOversizedPayload(t *testing.T) {
	oversized := map[string]any{
		"series": []any{
			map[string]any{
				"name": strings.Repeat("x", maxReportChartBytes*2),
			},
		},
	}

	if sanitized := sanitizeChartData(oversized); sanitized != nil {
		t.Fatalf("expected oversized chart payload to be dropped, got %#v", sanitized)
	}
}
