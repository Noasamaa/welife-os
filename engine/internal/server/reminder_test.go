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

func TestPendingRemindersEmpty(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodGet, "/api/v1/reminders/pending", nil)
	assertStatus(t, rec, http.StatusOK)
}

func TestMarkReminderReadNotFound(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodPatch, "/api/v1/reminders/nonexistent/read", nil)
	assertStatus(t, rec, http.StatusNotFound)
}

func TestDismissReminderNotFound(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodPatch, "/api/v1/reminders/nonexistent/dismiss", nil)
	assertStatus(t, rec, http.StatusNotFound)
}

func TestListReminderRulesEmpty(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodGet, "/api/v1/reminder-rules", nil)
	assertStatus(t, rec, http.StatusOK)
}

func TestCreateReminderRuleSuccess(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodPost, "/api/v1/reminder-rules", map[string]any{
		"rule_type":        "contact_gap",
		"entity_id":        "ent_1",
		"threshold_days":   7,
		"message_template": "No contact for 7 days",
	})
	assertStatus(t, rec, http.StatusCreated)
	assertBodyContains(t, rec, "contact_gap")
}

func TestCreateReminderRuleMissingRuleType(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodPost, "/api/v1/reminder-rules",
		map[string]string{"message_template": "hello"})
	assertStatus(t, rec, http.StatusBadRequest)
	assertBodyContains(t, rec, "rule_type")
}

func TestCreateReminderRuleMissingMessageTemplate(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodPost, "/api/v1/reminder-rules",
		map[string]string{"rule_type": "contact_gap"})
	assertStatus(t, rec, http.StatusBadRequest)
	assertBodyContains(t, rec, "message_template")
}

func TestUpdateReminderRuleMissingEnabled(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodPatch, "/api/v1/reminder-rules/some-id",
		map[string]string{})
	assertStatus(t, rec, http.StatusBadRequest)
	assertBodyContains(t, rec, "enabled")
}

func TestUpdateReminderRuleNotFound(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodPatch, "/api/v1/reminder-rules/nonexistent",
		map[string]any{"enabled": false})
	assertStatus(t, rec, http.StatusNotFound)
}

func TestDeleteReminderRuleNotFound(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodDelete, "/api/v1/reminder-rules/nonexistent", nil)
	assertStatus(t, rec, http.StatusNotFound)
}

func TestReminderRuleCRUD(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	// Create
	createRec := doJSON(t, app, http.MethodPost, "/api/v1/reminder-rules", map[string]any{
		"rule_type":        "contact_gap",
		"entity_id":        "ent_crud",
		"threshold_days":   14,
		"message_template": "Overdue contact",
	})
	assertStatus(t, createRec, http.StatusCreated)

	created := decodeJSON[map[string]any](t, createRec)
	ruleID, ok := created["id"].(string)
	if !ok || ruleID == "" {
		t.Fatal("expected non-empty rule id")
	}

	// List — should contain the created rule
	listRec := doJSON(t, app, http.MethodGet, "/api/v1/reminder-rules", nil)
	assertStatus(t, listRec, http.StatusOK)
	assertBodyContains(t, listRec, ruleID)

	// Update — disable the rule
	updateRec := doJSON(t, app, http.MethodPatch, "/api/v1/reminder-rules/"+ruleID,
		map[string]any{"enabled": false})
	assertStatus(t, updateRec, http.StatusOK)
	assertBodyContains(t, updateRec, "updated")

	// Delete
	deleteRec := doJSON(t, app, http.MethodDelete, "/api/v1/reminder-rules/"+ruleID, nil)
	assertStatus(t, deleteRec, http.StatusOK)
	assertBodyContains(t, deleteRec, "deleted")
}
