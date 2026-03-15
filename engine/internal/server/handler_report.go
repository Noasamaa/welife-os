package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/welife-os/welife-os/engine/internal/report"
	"github.com/welife-os/welife-os/engine/internal/storage"
)

func (s *Server) handleGenerateReport(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req report.GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.ConversationID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "conversation_id is required"})
		return
	}
	if req.Type != "weekly" && req.Type != "monthly" && req.Type != "annual" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "type must be weekly, monthly, or annual"})
		return
	}

	reportID, taskID, err := s.reportGenerator.Generate(r.Context(), req)
	if err != nil {
		writeResourceError(w, "generate-report", err, "failed to generate report")
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{
		"report_id": reportID,
		"task_id":   taskID,
	})
}

func (s *Server) handleListReports(w http.ResponseWriter, r *http.Request) {
	reports, err := s.reportGenerator.ListReports(r.Context())
	if err != nil {
		log.Printf("list-reports: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list reports"})
		return
	}
	if reports == nil {
		writeJSON(w, http.StatusOK, []storage.Report{})
		return
	}
	writeJSON(w, http.StatusOK, reports)
}

func (s *Server) handleGetReport(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rpt, err := s.reportGenerator.GetReport(r.Context(), id)
	if err != nil {
		writeResourceError(w, "get-report", err, "failed to get report")
		return
	}
	writeJSON(w, http.StatusOK, rpt)
}

func (s *Server) handleDeleteReport(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.reportGenerator.DeleteReport(r.Context(), id); err != nil {
		writeResourceError(w, "delete-report", err, "failed to delete report")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleExportReportHTML(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rpt, err := s.reportGenerator.GetReport(r.Context(), id)
	if err != nil {
		writeResourceError(w, "export-html", err, "failed to get report")
		return
	}

	content, err := parseReportContent(rpt)
	if err != nil {
		log.Printf("export-html: parse content: %v", err)
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "report content is not ready"})
		return
	}

	html, err := s.renderer.RenderHTML(content)
	if err != nil {
		log.Printf("export-html: render: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to render HTML"})
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, html)
}

func (s *Server) handleExportReportPDF(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rpt, err := s.reportGenerator.GetReport(r.Context(), id)
	if err != nil {
		writeResourceError(w, "export-pdf", err, "failed to get report")
		return
	}

	content, err := parseReportContent(rpt)
	if err != nil {
		log.Printf("export-pdf: parse content: %v", err)
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "report content is not ready"})
		return
	}

	html, err := s.renderer.RenderHTML(content)
	if err != nil {
		log.Printf("export-pdf: render HTML: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to render HTML"})
		return
	}

	pdf, err := s.renderer.RenderPDF(r.Context(), html)
	if err != nil {
		log.Printf("export-pdf: render PDF: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to render PDF"})
		return
	}

	filename := content.Title + ".pdf"
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	if _, err := w.Write(pdf); err != nil {
		log.Printf("handler: writing PDF response: %v", err)
	}
}

func parseReportContent(rpt storage.Report) (report.ReportContent, error) {
	if rpt.Status != "completed" {
		return report.ReportContent{}, fmt.Errorf("report status is %q, not completed", rpt.Status)
	}
	var content report.ReportContent
	if err := json.Unmarshal([]byte(rpt.Content), &content); err != nil {
		return report.ReportContent{}, fmt.Errorf("parsing report content: %w", err)
	}
	return content, nil
}
