package report

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/welife-os/welife-os/engine/internal/llm"
	"github.com/welife-os/welife-os/engine/internal/storage"
	"github.com/welife-os/welife-os/engine/internal/task"
)

var reportSeq uint64

// Generator orchestrates the report generation pipeline.
type Generator struct {
	agent *ReactAgent
	store *storage.Store
	tasks *task.Manager
	llm   llm.LLMClient
}

// NewGenerator creates a new report generator.
func NewGenerator(llmClient llm.LLMClient, store *storage.Store, tasks *task.Manager, tools []Tool) *Generator {
	return &Generator{
		agent: NewReactAgent(llmClient, tools),
		store: store,
		tasks: tasks,
		llm:   llmClient,
	}
}

// Generate starts async report generation, returning the report ID and task ID.
func (g *Generator) Generate(ctx context.Context, req GenerateRequest) (string, string, error) {
	// Validate report type
	if req.Type != "weekly" && req.Type != "monthly" && req.Type != "annual" {
		return "", "", fmt.Errorf("invalid report type %q: must be weekly, monthly, or annual", req.Type)
	}

	// Verify conversation exists
	if _, err := g.store.GetConversation(ctx, req.ConversationID); err != nil {
		return "", "", fmt.Errorf("conversation lookup: %w", err)
	}

	// Compute period
	period, err := ResolvePeriod(req.Type, req.PeriodStart, req.PeriodEnd)
	if err != nil {
		return "", "", err
	}

	seq := atomic.AddUint64(&reportSeq, 1)
	reportID := fmt.Sprintf("report_%d_%d", time.Now().UnixNano(), seq)
	title := TitleForType(req.Type, period)

	if err := g.store.CreateReport(ctx, storage.Report{
		ID:             reportID,
		Type:           req.Type,
		ConversationID: req.ConversationID,
		TaskID:         "pending",
		Status:         "running",
		Title:          title,
		Content:        "{}",
		PeriodStart:    period.Start,
		PeriodEnd:      period.End,
	}); err != nil {
		return "", "", fmt.Errorf("creating report: %w", err)
	}

	taskID := g.tasks.Submit("report_generate", func(taskCtx context.Context) error {
		return g.executeGeneration(taskCtx, reportID, req.ConversationID, req.Type, period)
	})

	// Update with real task ID
	if err := g.store.BindReportTask(ctx, reportID, taskID); err != nil {
		return "", "", fmt.Errorf("updating report task_id: %w", err)
	}

	return reportID, taskID, nil
}

// GetReport returns a report by ID.
func (g *Generator) GetReport(ctx context.Context, id string) (storage.Report, error) {
	return g.store.GetReport(ctx, id)
}

// ListReports returns all reports.
func (g *Generator) ListReports(ctx context.Context) ([]storage.Report, error) {
	return g.store.ListReports(ctx)
}

// DeleteReport removes a report by ID.
func (g *Generator) DeleteReport(ctx context.Context, id string) error {
	return g.store.DeleteReport(ctx, id)
}

func (g *Generator) executeGeneration(ctx context.Context, reportID, conversationID, reportType string, period ReportPeriod) error {
	sectionPlans, err := SectionsForType(reportType)
	if err != nil {
		g.failReport(ctx, reportID, err)
		return err
	}

	title := TitleForType(reportType, period)
	var sections []Section
	scope := ToolScope{
		ConversationID: conversationID,
		Period:         period,
	}

	for _, plan := range sectionPlans {
		section, err := g.agent.GenerateSection(ctx, plan, scope)
		if err != nil {
			g.failReport(ctx, reportID, err)
			return err
		}
		sections = append(sections, section)
	}

	// Generate summary
	summary, err := g.generateSummary(ctx, reportType, sections)
	if err != nil {
		g.failReport(ctx, reportID, err)
		return err
	}

	content := ReportContent{
		Title:    title,
		Type:     reportType,
		Period:   period,
		Sections: sections,
		Summary:  summary,
	}
	content = sanitizeReportContent(content)

	contentJSON, err := json.Marshal(content)
	if err != nil {
		g.failReport(ctx, reportID, err)
		return err
	}

	if err := g.store.UpdateReport(ctx, reportID, "completed", title, string(contentJSON)); err != nil {
		return fmt.Errorf("completing report: %w", err)
	}

	return nil
}

const summaryPrompt = `你是人生报告撰写专家。请基于以下报告章节内容，撰写一份简洁的报告总结。

报告类型: %s
章节内容:
%s

请以 JSON 格式输出：
{"summary": "报告总结文本（200字以内）"}

请只输出 JSON。`

func (g *Generator) generateSummary(ctx context.Context, reportType string, sections []Section) (string, error) {
	var sb fmt.Stringer = buildSectionsSummary(sections)

	prompt := fmt.Sprintf(summaryPrompt, reportType, sb)

	response, err := g.llm.Generate(ctx, prompt)
	if err != nil {
		return "报告已生成，但总结生成失败。", nil
	}

	jsonStr := llm.ExtractJSON(response)
	var result struct {
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return "报告已生成。", nil
	}

	return result.Summary, nil
}

type stringerFunc struct {
	s string
}

func (sf stringerFunc) String() string { return sf.s }

func buildSectionsSummary(sections []Section) fmt.Stringer {
	var sb string
	for _, s := range sections {
		sb += fmt.Sprintf("## %s\n%s\n\n", s.Title, s.Narrative)
	}
	return stringerFunc{s: sb}
}

func (g *Generator) failReport(ctx context.Context, reportID string, err error) {
	if dbErr := g.store.UpdateReport(ctx, reportID, "failed", "", err.Error()); dbErr != nil {
		log.Printf("report: failed to mark report %s as failed: %v", reportID, dbErr)
	}
}
