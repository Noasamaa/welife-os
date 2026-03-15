package server

import (
	"encoding/json"
	"net/http"
	"strings"

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
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
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
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
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
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, rpt)
}

func (s *Server) handleDeleteReport(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.reportGenerator.DeleteReport(r.Context(), id); err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
