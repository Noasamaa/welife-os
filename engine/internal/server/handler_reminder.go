package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/welife-os/welife-os/engine/internal/storage"
)

var ruleSeq uint64

func (s *Server) handlePendingReminders(w http.ResponseWriter, r *http.Request) {
	reminders, err := s.reminderService.ListPending(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if reminders == nil {
		writeJSON(w, http.StatusOK, []storage.Reminder{})
		return
	}
	writeJSON(w, http.StatusOK, reminders)
}

func (s *Server) handleMarkReminderRead(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.reminderService.MarkRead(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "read"})
}

func (s *Server) handleDismissReminder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.reminderService.Dismiss(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "dismissed"})
}

func (s *Server) handleListReminderRules(w http.ResponseWriter, r *http.Request) {
	rules, err := s.store.ListReminderRules(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if rules == nil {
		writeJSON(w, http.StatusOK, []storage.ReminderRule{})
		return
	}
	writeJSON(w, http.StatusOK, rules)
}

func (s *Server) handleCreateReminderRule(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req struct {
		RuleType        string `json:"rule_type"`
		EntityID        string `json:"entity_id"`
		ActionItemID    string `json:"action_item_id"`
		ThresholdDays   int    `json:"threshold_days"`
		CronExpr        string `json:"cron_expr"`
		MessageTemplate string `json:"message_template"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.RuleType == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "rule_type is required"})
		return
	}
	if req.MessageTemplate == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "message_template is required"})
		return
	}

	seq := atomic.AddUint64(&ruleSeq, 1)
	rule := storage.ReminderRule{
		ID:              fmt.Sprintf("rule_%d_%d", time.Now().UnixNano(), seq),
		RuleType:        req.RuleType,
		EntityID:        req.EntityID,
		ActionItemID:    req.ActionItemID,
		ThresholdDays:   req.ThresholdDays,
		CronExpr:        req.CronExpr,
		MessageTemplate: req.MessageTemplate,
		Enabled:         true,
	}

	if err := s.store.CreateReminderRule(r.Context(), rule); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, rule)
}

func (s *Server) handleUpdateReminderRule(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	id := chi.URLParam(r, "id")

	var req struct {
		Enabled *bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Enabled == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "enabled is required"})
		return
	}

	if err := s.store.UpdateReminderRule(r.Context(), id, *req.Enabled); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) handleDeleteReminderRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.store.DeleteReminderRule(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
