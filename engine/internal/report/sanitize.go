package report

import (
	"encoding/json"
	"strings"
	"unicode/utf8"
)

const (
	maxReportSections       = 12
	maxReportTitleRunes     = 120
	maxReportPeriodRunes    = 64
	maxReportNarrativeRunes = 4000
	maxReportSummaryRunes   = 400
	maxReportItemRunes      = 240
	maxReportChartBytes     = 32 << 10
	maxReportJSONDepth      = 8
	maxReportJSONNodes      = 800
	maxReportMapEntries     = 40
	maxReportArrayItems     = 100
	maxReportKeyRunes       = 64
	maxReportStringRunes    = 500
)

type jsonBudget struct {
	nodes int
}

func sanitizeReportContent(content ReportContent) ReportContent {
	content.Title = truncateText(content.Title, maxReportTitleRunes)
	content.Type = normalizeReportType(content.Type)
	content.Period = ReportPeriod{
		Start: truncateText(content.Period.Start, maxReportPeriodRunes),
		End:   truncateText(content.Period.End, maxReportPeriodRunes),
	}
	content.Summary = truncateText(content.Summary, maxReportSummaryRunes)

	if len(content.Sections) > maxReportSections {
		content.Sections = content.Sections[:maxReportSections]
	}

	sanitized := make([]Section, 0, len(content.Sections))
	for _, section := range content.Sections {
		sanitized = append(sanitized, sanitizeSection(section))
	}
	content.Sections = sanitized
	return content
}

func sanitizeSection(section Section) Section {
	section.Title = truncateText(section.Title, maxReportTitleRunes)
	section.Type = normalizeSectionType(section.Type)
	section.ChartType = normalizeChartType(section.ChartType)
	section.Narrative = truncateText(section.Narrative, maxReportNarrativeRunes)

	switch section.Type {
	case "chart":
		section.Data = sanitizeChartData(section.Data)
		section.Items = nil
	case "list":
		section.Data = nil
		section.ChartType = ""
		section.Items = sanitizeListItems(section.Items)
	default:
		section.Type = "text"
		section.ChartType = ""
		section.Data = nil
		section.Items = nil
	}

	return section
}

func sanitizeChartData(data any) any {
	if data == nil {
		return nil
	}

	normalized, ok := normalizeJSONLike(data)
	if !ok {
		return nil
	}

	root, ok := normalized.(map[string]any)
	if !ok {
		return nil
	}

	budget := &jsonBudget{}
	sanitized, ok := sanitizeJSONValue(root, 0, budget)
	if !ok {
		return nil
	}

	obj, ok := sanitized.(map[string]any)
	if !ok {
		return nil
	}

	raw, err := json.Marshal(obj)
	if err != nil || len(raw) > maxReportChartBytes {
		return nil
	}
	return obj
}

func sanitizeListItems(items []any) []any {
	if len(items) == 0 {
		return nil
	}

	limit := len(items)
	if limit > maxReportArrayItems {
		limit = maxReportArrayItems
	}

	sanitized := make([]any, 0, limit)
	for _, item := range items[:limit] {
		switch value := item.(type) {
		case string:
			text := truncateText(value, maxReportItemRunes)
			if text != "" {
				sanitized = append(sanitized, text)
			}
		default:
			normalized, ok := normalizeJSONLike(item)
			if !ok {
				continue
			}
			obj, ok := normalized.(map[string]any)
			if !ok {
				continue
			}

			entry := map[string]string{}
			for _, key := range []string{"title", "description", "content"} {
				raw, ok := obj[key].(string)
				if !ok {
					continue
				}
				text := truncateText(raw, maxReportItemRunes)
				if text != "" {
					entry[key] = text
				}
			}
			if len(entry) > 0 {
				sanitized = append(sanitized, entry)
			}
		}
	}

	if len(sanitized) == 0 {
		return nil
	}
	return sanitized
}

func sanitizeJSONValue(value any, depth int, budget *jsonBudget) (any, bool) {
	if depth > maxReportJSONDepth || budget.nodes >= maxReportJSONNodes {
		return nil, false
	}
	budget.nodes++

	switch typed := value.(type) {
	case nil, bool, float64:
		return typed, true
	case string:
		return truncateText(typed, maxReportStringRunes), true
	case []any:
		limit := len(typed)
		if limit > maxReportArrayItems {
			limit = maxReportArrayItems
		}

		items := make([]any, 0, limit)
		for _, item := range typed[:limit] {
			sanitized, ok := sanitizeJSONValue(item, depth+1, budget)
			if !ok {
				continue
			}
			items = append(items, sanitized)
		}
		return items, true
	case map[string]any:
		sanitized := make(map[string]any)
		count := 0
		for key, item := range typed {
			if count >= maxReportMapEntries {
				break
			}
			trimmedKey := truncateText(key, maxReportKeyRunes)
			if trimmedKey == "" {
				continue
			}
			sanitizedValue, ok := sanitizeJSONValue(item, depth+1, budget)
			if !ok {
				continue
			}
			sanitized[trimmedKey] = sanitizedValue
			count++
		}
		return sanitized, true
	default:
		return nil, false
	}
}

func normalizeJSONLike(value any) (any, bool) {
	raw, err := json.Marshal(value)
	if err != nil || len(raw) > maxReportChartBytes {
		return nil, false
	}

	var normalized any
	if err := json.Unmarshal(raw, &normalized); err != nil {
		return nil, false
	}
	return normalized, true
}

func truncateText(value string, limit int) string {
	value = strings.TrimSpace(value)
	if value == "" || limit <= 0 {
		return ""
	}
	if utf8.RuneCountInString(value) <= limit {
		return value
	}

	var builder strings.Builder
	builder.Grow(limit * 3)
	count := 0
	for _, r := range value {
		if count >= limit {
			break
		}
		builder.WriteRune(r)
		count++
	}
	return strings.TrimSpace(builder.String())
}

func normalizeReportType(value string) string {
	switch value {
	case "weekly", "monthly", "annual":
		return value
	default:
		return "weekly"
	}
}

func normalizeSectionType(value string) string {
	switch value {
	case "chart", "list", "text":
		return value
	default:
		return "text"
	}
}

func normalizeChartType(value string) string {
	switch value {
	case "line", "network", "heatmap":
		return value
	default:
		return ""
	}
}
