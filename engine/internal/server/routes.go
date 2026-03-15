package server

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

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
	router.Use(devCORSMiddleware)
	router.Get("/health", s.handleHealth)
	router.Get("/api/v1/system/status", s.handleSystemStatus)
	router.Get("/api/v1/system/llm-config", s.handleGetLLMConfig)
	router.Patch("/api/v1/system/llm-config", s.handleUpdateLLMConfig)

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
	router.Get("/api/v1/reports/{id}/html", s.handleExportReportHTML)
	router.Get("/api/v1/reports/{id}/pdf", s.handleExportReportPDF)

	// Coach / Action Items endpoints
	router.Post("/api/v1/coach/generate-plan", s.handleGenerateActionPlan)
	router.Get("/api/v1/action-items", s.handleListActionItems)
	router.Get("/api/v1/action-items/{id}", s.handleGetActionItem)
	router.Patch("/api/v1/action-items/{id}", s.handleUpdateActionItem)
	router.Delete("/api/v1/action-items/{id}", s.handleDeleteActionItem)

	// Reminder endpoints
	router.Get("/api/v1/reminders/pending", s.handlePendingReminders)
	router.Patch("/api/v1/reminders/{id}/read", s.handleMarkReminderRead)
	router.Patch("/api/v1/reminders/{id}/dismiss", s.handleDismissReminder)
	router.Get("/api/v1/reminder-rules", s.handleListReminderRules)
	router.Post("/api/v1/reminder-rules", s.handleCreateReminderRule)
	router.Patch("/api/v1/reminder-rules/{id}", s.handleUpdateReminderRule)
	router.Delete("/api/v1/reminder-rules/{id}", s.handleDeleteReminderRule)

	// Simulation endpoints
	router.Post("/api/v1/simulation/profiles/build", s.handleBuildProfiles)
	router.Get("/api/v1/simulation/profiles", s.handleListProfiles)
	router.Post("/api/v1/simulation/run", s.handleRunSimulation)
	router.Get("/api/v1/simulation/sessions", s.handleListSimulations)
	router.Get("/api/v1/simulation/sessions/{id}", s.handleGetSimulation)

	// Embedding / semantic search endpoints
	router.Post("/api/v1/embeddings/build", s.handleBuildEmbeddings)
	router.Post("/api/v1/search/semantic", s.handleSemanticSearch)

	return router
}

var allowedOrigins = map[string]struct{}{
	"http://localhost:1420":   {},
	"http://127.0.0.1:1420":   {},
	"tauri://localhost":       {},
	"http://tauri.localhost":  {},
	"https://tauri.localhost": {},
}

func devCORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if _, ok := allowedOrigins[origin]; ok {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
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

// writeResourceError logs the real error server-side and returns a safe
// HTTP response. If the error message contains "not found" it maps to 404;
// otherwise it maps to 500.
func writeResourceError(w http.ResponseWriter, handler string, err error, fallbackMsg string) {
	status := http.StatusInternalServerError
	msg := fallbackMsg
	if strings.Contains(err.Error(), "not found") {
		status = http.StatusNotFound
		msg = "resource not found"
	}
	log.Printf("%s: %v", handler, err)
	writeJSON(w, status, map[string]string{"error": msg})
}
