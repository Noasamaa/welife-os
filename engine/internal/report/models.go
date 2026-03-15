package report

// ReportContent is the structured content of a generated report, stored as JSON.
type ReportContent struct {
	Title    string       `json:"title"`
	Type     string       `json:"type"`
	Period   ReportPeriod `json:"period"`
	Sections []Section    `json:"sections"`
	Summary  string       `json:"summary"`
}

// ReportPeriod defines the time range of a report.
type ReportPeriod struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// ToolScope constrains ReACT tool calls to a single conversation and time range.
type ToolScope struct {
	ConversationID string
	Period         ReportPeriod
}

// Section represents one section of a report.
type Section struct {
	Title     string `json:"title"`
	Type      string `json:"type"`                 // "chart", "list", "text"
	ChartType string `json:"chart_type,omitempty"` // "line", "network", "heatmap"
	Data      any    `json:"data,omitempty"`       // ECharts option for chart sections
	Items     []any  `json:"items,omitempty"`      // for list sections
	Narrative string `json:"narrative"`
}

// SectionPlan describes what a section should contain, used as input to the ReACT loop.
type SectionPlan struct {
	Title     string `json:"title"`
	Type      string `json:"type"`
	ChartType string `json:"chart_type,omitempty"`
	Hints     string `json:"hints"`
}

// GenerateRequest is the input for triggering report generation.
type GenerateRequest struct {
	Type           string `json:"type"`
	ConversationID string `json:"conversation_id"`
	PeriodStart    string `json:"period_start,omitempty"`
	PeriodEnd      string `json:"period_end,omitempty"`
}
