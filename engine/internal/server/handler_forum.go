package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/welife-os/welife-os/engine/internal/storage"
)

func (s *Server) handleTriggerDebate(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit

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

	sessionID, taskID, err := s.forumEngine.RunDebate(r.Context(), req.ConversationID)
	if err != nil {
		writeResourceError(w, "trigger-debate", err, "failed to start debate")
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{
		"session_id": sessionID,
		"task_id":    taskID,
	})
}

func (s *Server) handleListForumSessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := s.forumEngine.ListSessions(r.Context())
	if err != nil {
		log.Printf("list-forum-sessions: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list sessions"})
		return
	}
	if sessions == nil {
		writeJSON(w, http.StatusOK, []storage.ForumSession{})
		return
	}
	writeJSON(w, http.StatusOK, sessions)
}

func (s *Server) handleGetForumSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	session, err := s.forumEngine.GetSession(r.Context(), id)
	if err != nil {
		writeResourceError(w, "get-forum-session", err, "failed to get session")
		return
	}

	messages, err := s.forumEngine.GetSessionMessages(r.Context(), id)
	if err != nil {
		log.Printf("get-forum-messages: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get session messages"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"session":  session,
		"messages": messages,
	})
}
