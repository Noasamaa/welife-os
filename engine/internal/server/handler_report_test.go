package server_test

import (
	"net/http"
	"testing"
)

func TestGenerateReportMissingConversationID(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodPost, "/api/v1/reports/generate",
		map[string]string{"type": "weekly"})
	assertStatus(t, rec, http.StatusBadRequest)
	assertBodyContains(t, rec, "conversation_id")
}

func TestGenerateReportInvalidType(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodPost, "/api/v1/reports/generate",
		map[string]string{"conversation_id": "conv_1", "type": "invalid"})
	assertStatus(t, rec, http.StatusBadRequest)
	assertBodyContains(t, rec, "type must be")
}

func TestGenerateReportInvalidJSON(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodPost, "/api/v1/reports/generate", "{bad")
	assertStatus(t, rec, http.StatusBadRequest)
	assertBodyContains(t, rec, "invalid")
}

func TestListReportsEmpty(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodGet, "/api/v1/reports", nil)
	assertStatus(t, rec, http.StatusOK)
}

func TestGetReportNotFound(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodGet, "/api/v1/reports/nonexistent", nil)
	assertStatus(t, rec, http.StatusNotFound)
	assertBodyContains(t, rec, "not found")
}

func TestDeleteReportNotFound(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodDelete, "/api/v1/reports/nonexistent", nil)
	assertStatus(t, rec, http.StatusNotFound)
	assertBodyContains(t, rec, "not found")
}

func TestExportReportHTMLNotFound(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodGet, "/api/v1/reports/nonexistent/html", nil)
	assertStatus(t, rec, http.StatusNotFound)
	assertBodyContains(t, rec, "not found")
}

func TestExportReportPDFNotFound(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodGet, "/api/v1/reports/nonexistent/pdf", nil)
	assertStatus(t, rec, http.StatusNotFound)
	assertBodyContains(t, rec, "not found")
}
