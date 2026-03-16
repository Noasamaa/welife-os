package server

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/welife-os/welife-os/engine/internal/importer"
	"github.com/welife-os/welife-os/engine/internal/parser"
)

func (s *Server) handleImportUpload(w http.ResponseWriter, r *http.Request) {
	// Limit upload to 100MB
	r.Body = http.MaxBytesReader(w, r.Body, 100<<20)

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid multipart form or file too large"})
		return
	}
	if r.MultipartForm != nil {
		defer func() {
			if err := r.MultipartForm.RemoveAll(); err != nil {
				log.Printf("import-upload: cleanup multipart form: %v", err)
			}
		}()
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing 'file' field"})
		return
	}
	defer file.Close()

	formatStr := r.FormValue("format")
	selfName := r.FormValue("self_name")

	var format parser.Format
	if formatStr != "" && formatStr != "auto" {
		format = parser.Format(formatStr)
	}

	opts := parser.Options{
		SelfName: selfName,
	}

	req := importer.ImportRequest{
		FileName: header.Filename,
		Format:   format,
		Data:     file,
		Options:  opts,
	}

	result, err := s.importer.Import(r.Context(), req)
	if err != nil {
		log.Printf("import-upload: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "import failed"})
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

func (s *Server) handleListImportJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := s.store.ListImportJobs(r.Context())
	if err != nil {
		log.Printf("list-import-jobs: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list import jobs"})
		return
	}
	writeJSON(w, http.StatusOK, jobs)
}

func (s *Server) handleGetImportJob(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	job, err := s.store.GetImportJob(r.Context(), id)
	if err != nil {
		writeResourceError(w, "get-import-job", err, "failed to get import job")
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (s *Server) handleDeleteImportJob(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.store.DeleteImportJob(r.Context(), id); err != nil {
		writeResourceError(w, "delete-import-job", err, "failed to delete import job")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleListConversations(w http.ResponseWriter, r *http.Request) {
	convs, err := s.store.ListConversations(r.Context())
	if err != nil {
		log.Printf("list-conversations: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list conversations"})
		return
	}
	writeJSON(w, http.StatusOK, convs)
}

func (s *Server) handleGetConversation(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	conv, err := s.store.GetConversation(r.Context(), id)
	if err != nil {
		writeResourceError(w, "get-conversation", err, "failed to get conversation")
		return
	}
	writeJSON(w, http.StatusOK, conv)
}

func (s *Server) handleDeleteConversation(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.store.DeleteConversation(r.Context(), id); err != nil {
		writeResourceError(w, "delete-conversation", err, "failed to delete conversation")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleGetMessages(w http.ResponseWriter, r *http.Request) {
	convID := chi.URLParam(r, "id")
	limit := queryInt(r, "limit", 50, maxMessagePageSize)
	offset := queryInt(r, "offset", 0, 0)

	msgs, err := s.store.GetMessages(r.Context(), convID, limit, offset)
	if err != nil {
		log.Printf("get-messages: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get messages"})
		return
	}

	total, err := s.store.MessageCount(r.Context(), convID)
	if err != nil {
		log.Printf("[WARN] get-messages: message count: %v", err)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"messages": msgs,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

func queryInt(r *http.Request, key string, fallback int, max int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return fallback
	}
	var n int
	if _, err := fmt.Sscanf(v, "%d", &n); err != nil || n < 0 {
		return fallback
	}
	if max > 0 && n > max {
		return max
	}
	return n
}
