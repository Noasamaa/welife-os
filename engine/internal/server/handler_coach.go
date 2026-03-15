package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/welife-os/welife-os/engine/internal/storage"
)

func (s *Server) handleGenerateActionPlan(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req struct {
		SessionID string `json:"session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.SessionID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "session_id is required"})
		return
	}

	items, err := s.coachAgent.GenerateActionPlan(r.Context(), req.SessionID)
	if err != nil {
		writeResourceError(w, "generate-action-plan", err, "failed to generate action plan")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
		"count": len(items),
	})
}

func (s *Server) handleListActionItems(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	category := r.URL.Query().Get("category")

	items, err := s.store.ListActionItems(r.Context(), status, category)
	if err != nil {
		log.Printf("list-action-items: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list action items"})
		return
	}
	if items == nil {
		writeJSON(w, http.StatusOK, []storage.ActionItem{})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleGetActionItem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	item, err := s.store.GetActionItem(r.Context(), id)
	if err != nil {
		writeResourceError(w, "get-action-item", err, "failed to get action item")
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleUpdateActionItem(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	id := chi.URLParam(r, "id")

	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Status == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "status is required"})
		return
	}

	if err := s.store.UpdateActionItemStatus(r.Context(), id, req.Status); err != nil {
		writeResourceError(w, "update-action-item", err, "failed to update action item")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) handleDeleteActionItem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := s.store.DeleteActionItem(r.Context(), id); err != nil {
		writeResourceError(w, "delete-action-item", err, "failed to delete action item")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
