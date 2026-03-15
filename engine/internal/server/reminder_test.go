package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/welife-os/welife-os/engine/internal/server"
	"github.com/welife-os/welife-os/engine/internal/testutil"
)

func TestCORSPreflightAllowsKnownOrigin(t *testing.T) {
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models":[]}`))
	}))
	defer ollama.Close()

	app, err := server.New(testutil.NewServerConfig(t, ollama.URL))
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	defer func() { _ = app.Shutdown(t.Context()) }()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodOptions, "/api/v1/system/status", nil)
	request.Header.Set("Origin", "http://localhost:1420")
	request.Header.Set("Access-Control-Request-Method", "GET")

	app.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:1420" {
		t.Fatalf("unexpected allow origin: %q", got)
	}
}

func TestCreateReminderRuleRejectsMissingRequiredFields(t *testing.T) {
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models":[]}`))
	}))
	defer ollama.Close()

	app, err := server.New(testutil.NewServerConfig(t, ollama.URL))
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	defer func() { _ = app.Shutdown(t.Context()) }()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/reminder-rules",
		strings.NewReader(`{"rule_type":"deadline","message_template":"soon"}`))
	request.Header.Set("Content-Type", "application/json")

	app.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "action_item_id") {
		t.Fatalf("expected action_item_id validation error, got %s", recorder.Body.String())
	}
}
