package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/welife-os/welife-os/engine/internal/server"
	"github.com/welife-os/welife-os/engine/internal/testutil"
)

func TestHealthEndpoint(t *testing.T) {
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models":[]}`))
	}))
	defer ollama.Close()

	app, err := server.New(testutil.NewServerConfig(t, ollama.URL))
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	defer func() {
		_ = app.Shutdown(t.Context())
	}()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	app.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
}

func TestSystemStatusEndpoint(t *testing.T) {
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models":[]}`))
	}))
	defer ollama.Close()

	app, err := server.New(testutil.NewServerConfig(t, ollama.URL))
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	defer func() {
		_ = app.Shutdown(t.Context())
	}()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/system/status", nil)
	app.Handler().ServeHTTP(recorder, request)

	var payload struct {
		Backend struct {
			Status string `json:"status"`
		} `json:"backend"`
		Storage struct {
			Driver string `json:"driver"`
			Ready  bool   `json:"ready"`
		} `json:"storage"`
		LLM struct {
			Provider  string `json:"provider"`
			Reachable bool   `json:"reachable"`
			Model     string `json:"model"`
		} `json:"llm"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if payload.Backend.Status != "ok" {
		t.Fatalf("unexpected backend status: %s", payload.Backend.Status)
	}
	if payload.Storage.Driver != "sqlcipher" || !payload.Storage.Ready {
		t.Fatalf("unexpected storage status: %+v", payload.Storage)
	}
	if payload.LLM.Provider != "ollama" || !payload.LLM.Reachable {
		t.Fatalf("unexpected llm status: %+v", payload.LLM)
	}
}

func TestSystemStatusToleratesUnavailableOllama(t *testing.T) {
	app, err := server.New(server.Config{
		Host:         "127.0.0.1",
		Port:         18080,
		DatabasePath: testutil.TempDatabasePath(t),
		DatabaseKey:  "welife-phase0-test-key",
		LLMProvider:  "ollama",
		LLMBaseURL:   "http://127.0.0.1:65530",
		LLMModel:     "qwen3.5:9b",
	})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	defer func() {
		_ = app.Shutdown(t.Context())
	}()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/system/status", nil)
	app.Handler().ServeHTTP(recorder, request)

	var payload struct {
		LLM struct {
			Reachable bool `json:"reachable"`
		} `json:"llm"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.LLM.Reachable {
		t.Fatal("llm should be reported as unreachable")
	}
}
