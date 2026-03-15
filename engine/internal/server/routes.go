package server

import (
	"crypto/subtle"
	"encoding/json"
	"log"
	"net/http"
	"os"
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
	router.Route("/api/v1", func(api chi.Router) {
		api.Use(apiAuthMiddleware)
		api.Get("/system/status", s.handleSystemStatus)
		api.Get("/system/llm-config", s.handleGetLLMConfig)
		api.Patch("/system/llm-config", s.handleUpdateLLMConfig)

		// Import endpoints
		api.Post("/import", s.handleImportUpload)
		api.Get("/import/jobs", s.handleListImportJobs)
		api.Get("/import/jobs/{id}", s.handleGetImportJob)
		api.Delete("/import/jobs/{id}", s.handleDeleteImportJob)

		// Conversation endpoints
		api.Get("/conversations", s.handleListConversations)
		api.Get("/conversations/{id}", s.handleGetConversation)
		api.Delete("/conversations/{id}", s.handleDeleteConversation)
		api.Get("/conversations/{id}/messages", s.handleGetMessages)

		// Graph endpoints
		api.Post("/graph/build", s.handleTriggerGraphBuild)
		api.Get("/graph/overview", s.handleGraphOverview)
		api.Get("/graph/nodes/{id}", s.handleGetGraphNode)
		api.Get("/graph/nodes/{id}/neighborhood", s.handleGetGraphNeighborhood)
		api.Get("/graph/search", s.handleGraphSearch)

		// Forum debate endpoints
		api.Post("/forum/debate", s.handleTriggerDebate)
		api.Get("/forum/sessions", s.handleListForumSessions)
		api.Get("/forum/sessions/{id}", s.handleGetForumSession)

		// Report endpoints
		api.Post("/reports/generate", s.handleGenerateReport)
		api.Get("/reports", s.handleListReports)
		api.Get("/reports/{id}", s.handleGetReport)
		api.Delete("/reports/{id}", s.handleDeleteReport)
		api.Get("/reports/{id}/html", s.handleExportReportHTML)
		api.Get("/reports/{id}/pdf", s.handleExportReportPDF)

		// Coach / Action Items endpoints
		api.Post("/coach/generate-plan", s.handleGenerateActionPlan)
		api.Get("/action-items", s.handleListActionItems)
		api.Get("/action-items/{id}", s.handleGetActionItem)
		api.Patch("/action-items/{id}", s.handleUpdateActionItem)
		api.Delete("/action-items/{id}", s.handleDeleteActionItem)

		// Reminder endpoints
		api.Get("/reminders/pending", s.handlePendingReminders)
		api.Patch("/reminders/{id}/read", s.handleMarkReminderRead)
		api.Patch("/reminders/{id}/dismiss", s.handleDismissReminder)
		api.Get("/reminder-rules", s.handleListReminderRules)
		api.Post("/reminder-rules", s.handleCreateReminderRule)
		api.Patch("/reminder-rules/{id}", s.handleUpdateReminderRule)
		api.Delete("/reminder-rules/{id}", s.handleDeleteReminderRule)

		// Simulation endpoints
		api.Post("/simulation/profiles/build", s.handleBuildProfiles)
		api.Get("/simulation/profiles", s.handleListProfiles)
		api.Post("/simulation/run", s.handleRunSimulation)
		api.Get("/simulation/sessions", s.handleListSimulations)
		api.Get("/simulation/sessions/{id}", s.handleGetSimulation)

		// Embedding / semantic search endpoints
		api.Post("/embeddings/build", s.handleBuildEmbeddings)
		api.Post("/search/semantic", s.handleSemanticSearch)
	})

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
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-WeLife-API-Token")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

const apiTokenHeader = "X-WeLife-API-Token"

func apiAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedToken, ok := os.LookupEnv("WELIFE_API_TOKEN")
		if !ok || strings.TrimSpace(expectedToken) == "" {
			next.ServeHTTP(w, r)
			return
		}

		providedToken := r.Header.Get(apiTokenHeader)
		if subtle.ConstantTimeCompare([]byte(providedToken), []byte(expectedToken)) != 1 {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
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
