package server

import (
	"encoding/json"
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

	taskID, err := s.simEngine.BuildAllProfilesAsync(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{
		"task_id": taskID,
		"status":  "building",
	})
}

func (s *Server) handleListProfiles(w http.ResponseWriter, r *http.Request) {
	profiles, err := s.store.ListPersonProfiles(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
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
	if config.ForkPoint.Description == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "fork_point.description is required"})
		return
	}

	sessionID, taskID, err := s.simEngine.RunSimulation(r.Context(), config)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
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

	sessions, err := s.simEngine.ListSessions(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
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

	session, err := s.simEngine.GetSession(r.Context(), id)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}

	steps, err := s.simEngine.GetSessionSteps(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"session": session,
		"steps":   steps,
	})
}
