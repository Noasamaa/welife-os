package server

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type healthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

type systemStatusResponse struct {
	Backend backendStatus `json:"backend"`
	Storage storageStatus `json:"storage"`
	LLM     llmStatus     `json:"llm"`
}

type backendStatus struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

type storageStatus struct {
	Driver string `json:"driver"`
	Ready  bool   `json:"ready"`
	Path   string `json:"path"`
}

type llmStatus struct {
	Provider  string `json:"provider"`
	Reachable bool   `json:"reachable"`
	BaseURL   string `json:"base_url"`
	Model     string `json:"model"`
}

func (s *Server) routes() http.Handler {
	router := chi.NewRouter()
	router.Get("/health", s.handleHealth)
	router.Get("/api/v1/system/status", s.handleSystemStatus)

	// Import endpoints
	router.Post("/api/v1/import", s.handleImportUpload)
	router.Get("/api/v1/import/jobs", s.handleListImportJobs)
	router.Get("/api/v1/import/jobs/{id}", s.handleGetImportJob)

	// Conversation endpoints
	router.Get("/api/v1/conversations", s.handleListConversations)
	router.Get("/api/v1/conversations/{id}", s.handleGetConversation)
	router.Get("/api/v1/conversations/{id}/messages", s.handleGetMessages)

	// Graph endpoints
	router.Post("/api/v1/graph/build", s.handleTriggerGraphBuild)
	router.Get("/api/v1/graph/overview", s.handleGraphOverview)

	// Forum debate endpoints
	router.Post("/api/v1/forum/debate", s.handleTriggerDebate)
	router.Get("/api/v1/forum/sessions", s.handleListForumSessions)
	router.Get("/api/v1/forum/sessions/{id}", s.handleGetForumSession)

	// Report endpoints
	router.Post("/api/v1/reports/generate", s.handleGenerateReport)
	router.Get("/api/v1/reports", s.handleListReports)
	router.Get("/api/v1/reports/{id}", s.handleGetReport)
	router.Delete("/api/v1/reports/{id}", s.handleDeleteReport)

	return router
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{
		Status:  "ok",
		Service: "welife-engine",
	})
}

func (s *Server) handleSystemStatus(w http.ResponseWriter, r *http.Request) {
	storageReady := s.store.Probe(r.Context()) == nil
	llmState := s.llmClient.Status(r.Context())

	writeJSON(w, http.StatusOK, systemStatusResponse{
		Backend: backendStatus{
			Status:  "ok",
			Version: Version,
		},
		Storage: storageStatus{
			Driver: "sqlcipher",
			Ready:  storageReady,
			Path:   s.store.Path(),
		},
		LLM: llmStatus{
			Provider:  llmState.Provider,
			Reachable: llmState.Reachable,
			BaseURL:   llmState.BaseURL,
			Model:     llmState.Model,
		},
	})
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}
