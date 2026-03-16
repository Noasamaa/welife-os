package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleGetTaskStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	info, ok := s.taskManager.Status(id)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
		return
	}
	writeJSON(w, http.StatusOK, info)
}
