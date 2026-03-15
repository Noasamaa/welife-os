package server

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/welife-os/welife-os/engine/internal/simulation"
	"github.com/welife-os/welife-os/engine/internal/storage"
)

func (s *Server) handleBuildProfiles(w http.ResponseWriter, r *http.Request) {
	if s.simEngine == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "simulation engine not initialized"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req struct {
		ConversationID string `json:"conversation_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if strings.TrimSpace(req.ConversationID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "conversation_id is required"})
		return
	}

	taskID, err := s.simEngine.BuildAllProfilesAsync(r.Context(), req.ConversationID)
	if err != nil {
		writeResourceError(w, "build-profiles", err, "failed to start profile build")
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{
		"task_id": taskID,
		"status":  "building",
	})
}

func (s *Server) handleListProfiles(w http.ResponseWriter, r *http.Request) {
	conversationID := strings.TrimSpace(r.URL.Query().Get("conversation_id"))
	if conversationID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "conversation_id is required"})
		return
	}

	profiles, err := s.store.ListPersonProfilesByConversation(r.Context(), conversationID)
	if err != nil {
		log.Printf("list-profiles: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list profiles"})
		return
	}
	if profiles == nil {
		writeJSON(w, http.StatusOK, []storage.PersonProfile{})
		return
	}
	writeJSON(w, http.StatusOK, profiles)
}

func (s *Server) handleRunSimulation(w http.ResponseWriter, r *http.Request) {
	if s.simEngine == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "simulation engine not initialized"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var config simulation.SimulationConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if strings.TrimSpace(config.ConversationID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "conversation_id is required"})
		return
	}
	if config.ForkPoint.Description == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "fork_point.description is required"})
		return
	}

	sessionID, taskID, err := s.simEngine.RunSimulation(r.Context(), config)
	if err != nil {
		writeResourceError(w, "run-simulation", err, "failed to start simulation")
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{
		"session_id": sessionID,
		"task_id":    taskID,
	})
}

func (s *Server) handleListSimulations(w http.ResponseWriter, r *http.Request) {
	if s.simEngine == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "simulation engine not initialized"})
		return
	}

	conversationID := strings.TrimSpace(r.URL.Query().Get("conversation_id"))
	if conversationID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "conversation_id is required"})
		return
	}

	sessions, err := s.simEngine.ListSessions(r.Context(), conversationID)
	if err != nil {
		log.Printf("list-simulations: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list simulations"})
		return
	}
	if sessions == nil {
		writeJSON(w, http.StatusOK, []storage.SimulationSession{})
		return
	}
	writeJSON(w, http.StatusOK, sessions)
}

func (s *Server) handleGetSimulation(w http.ResponseWriter, r *http.Request) {
	if s.simEngine == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "simulation engine not initialized"})
		return
	}

	id := chi.URLParam(r, "id")
	conversationID := strings.TrimSpace(r.URL.Query().Get("conversation_id"))
	if conversationID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "conversation_id is required"})
		return
	}

	session, err := s.simEngine.GetSession(r.Context(), id)
	if err != nil {
		writeResourceError(w, "get-simulation", err, "failed to get simulation")
		return
	}
	if session.ConversationID != conversationID {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "resource not found"})
		return
	}

	steps, err := s.simEngine.GetSessionSteps(r.Context(), id)
	if err != nil {
		log.Printf("get-simulation-steps: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get simulation steps"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"session": session,
		"steps":   steps,
	})
}
