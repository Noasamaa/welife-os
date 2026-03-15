package server_test

import (
	"net/http"
	"testing"
)

func TestTriggerDebateMissingConversationID(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodPost, "/api/v1/forum/debate",
		map[string]string{})
	assertStatus(t, rec, http.StatusBadRequest)
	assertBodyContains(t, rec, "conversation_id")
}

func TestTriggerDebateInvalidJSON(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodPost, "/api/v1/forum/debate", "{bad")
	assertStatus(t, rec, http.StatusBadRequest)
	assertBodyContains(t, rec, "invalid")
}

func TestListForumSessionsEmpty(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodGet, "/api/v1/forum/sessions", nil)
	assertStatus(t, rec, http.StatusOK)
}

func TestGetForumSessionNotFound(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodGet, "/api/v1/forum/sessions/nonexistent", nil)
	assertStatus(t, rec, http.StatusNotFound)
	assertBodyContains(t, rec, "not found")
}
