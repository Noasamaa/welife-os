package server_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/welife-os/welife-os/engine/internal/server"
	"github.com/welife-os/welife-os/engine/internal/storage"
	"github.com/welife-os/welife-os/engine/internal/testutil"
)

func TestUpdateLLMConfigRejectsInvalidConfigWithoutPersisting(t *testing.T) {
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models":[]}`))
	}))
	defer ollama.Close()

	cfg := testutil.NewServerConfig(t, ollama.URL)
	app, err := server.New(cfg)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	defer func() { _ = app.Shutdown(t.Context()) }()

	rec := doJSON(t, app, http.MethodPatch, "/api/v1/system/llm-config", map[string]string{
		"provider": "openai-compatible",
		"base_url": "https://example.com",
		"model":    "gpt-test",
	})
	assertStatus(t, rec, http.StatusBadRequest)
	assertBodyContains(t, rec, "invalid llm config")

	getRec := doJSON(t, app, http.MethodGet, "/api/v1/system/llm-config", nil)
	assertStatus(t, getRec, http.StatusOK)

	var payload struct {
		Provider string `json:"provider"`
		Model    string `json:"model"`
	}
	payload = decodeJSON[struct {
		Provider string `json:"provider"`
		Model    string `json:"model"`
	}](t, getRec)
	if payload.Provider != "ollama" {
		t.Fatalf("provider = %q, want ollama", payload.Provider)
	}
	if payload.Model != "qwen3.5:9b" {
		t.Fatalf("model = %q, want qwen3.5:9b", payload.Model)
	}
}

func TestServerFallsBackWhenPersistedLLMConfigIsInvalid(t *testing.T) {
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models":[]}`))
	}))
	defer ollama.Close()

	cfg := testutil.NewServerConfig(t, ollama.URL)

	store, err := storage.Open(t.Context(), storage.Config{
		Path: cfg.DatabasePath,
		Key:  cfg.DatabaseKey,
	})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	if err := store.SaveSettings(t.Context(), map[string]string{
		"llm_provider": "openai-compatible",
		"llm_base_url": "https://example.com",
		"llm_model":    "gpt-test",
	}); err != nil {
		t.Fatalf("seed settings: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}

	app, err := server.New(cfg)
	if err != nil {
		t.Fatalf("server should fall back to base config, got: %v", err)
	}
	defer func() { _ = app.Shutdown(t.Context()) }()

	rec := doJSON(t, app, http.MethodGet, "/api/v1/system/status", nil)
	assertStatus(t, rec, http.StatusOK)
}
