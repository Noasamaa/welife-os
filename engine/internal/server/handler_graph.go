package server

import (
	"encoding/json"
	"log"
	"net/http"
)

func (s *Server) handleTriggerGraphBuild(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req struct {
		ConversationID string `json:"conversation_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.ConversationID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "conversation_id is required"})
		return
	}

	taskID, err := s.graphEngine.BuildGraph(r.Context(), req.ConversationID)
	if err != nil {
		log.Printf("graph-build: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to start graph build"})
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{
		"task_id": taskID,
		"status":  "building",
	})
}

func (s *Server) handleGraphOverview(w http.ResponseWriter, r *http.Request) {
	overview, err := s.graphEngine.GetOverview(r.Context())
	if err != nil {
		log.Printf("graph-overview: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load graph overview"})
		return
	}
	writeJSON(w, http.StatusOK, overview)
}
