package server_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestImportUploadMissingFile(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	// Send a multipart form without the "file" field.
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	_ = writer.WriteField("format", "auto")
	writer.Close()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/import", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	app.Handler().ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusBadRequest)
	assertBodyContains(t, rec, "file")
}

func TestListImportJobsEmpty(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodGet, "/api/v1/import/jobs", nil)
	assertStatus(t, rec, http.StatusOK)
}

func TestGetImportJobNotFound(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodGet, "/api/v1/import/jobs/nonexistent", nil)
	assertStatus(t, rec, http.StatusNotFound)
	assertBodyContains(t, rec, "not found")
}

func TestListConversationsEmpty(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodGet, "/api/v1/conversations", nil)
	assertStatus(t, rec, http.StatusOK)
}

func TestGetConversationNotFound(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodGet, "/api/v1/conversations/nonexistent", nil)
	assertStatus(t, rec, http.StatusNotFound)
	assertBodyContains(t, rec, "not found")
}

func TestGetMessagesEmptyConversation(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodGet, "/api/v1/conversations/nonexistent/messages", nil)
	assertStatus(t, rec, http.StatusOK)

	payload := decodeJSON[map[string]any](t, rec)
	if _, ok := payload["messages"]; !ok {
		t.Fatal("expected 'messages' key in response")
	}
	if _, ok := payload["total"]; !ok {
		t.Fatal("expected 'total' key in response")
	}
}

func TestGetMessagesRespectsLimitOffset(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodGet, "/api/v1/conversations/xxx/messages?limit=10&offset=5", nil)
	assertStatus(t, rec, http.StatusOK)

	payload := decodeJSON[map[string]any](t, rec)
	if int(payload["limit"].(float64)) != 10 {
		t.Fatalf("expected limit=10, got %v", payload["limit"])
	}
	if int(payload["offset"].(float64)) != 5 {
		t.Fatalf("expected offset=5, got %v", payload["offset"])
	}
}

func TestGetMessagesCapsLimit(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodGet, "/api/v1/conversations/xxx/messages?limit=999999&offset=0", nil)
	assertStatus(t, rec, http.StatusOK)

	payload := decodeJSON[map[string]any](t, rec)
	if int(payload["limit"].(float64)) != 500 {
		t.Fatalf("expected capped limit=500, got %v", payload["limit"])
	}
}
