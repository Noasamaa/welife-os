package server_test

import (
	"net/http"
	"testing"
)

func TestBuildEmbeddingsMissingConversationID(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodPost, "/api/v1/embeddings/build",
		map[string]string{})
	assertStatus(t, rec, http.StatusBadRequest)
	assertBodyContains(t, rec, "conversation_id")
}

func TestBuildEmbeddingsInvalidJSON(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodPost, "/api/v1/embeddings/build", "{bad")
	assertStatus(t, rec, http.StatusBadRequest)
	assertBodyContains(t, rec, "invalid")
}

func TestSemanticSearchMissingQuery(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodPost, "/api/v1/search/semantic",
		map[string]string{})
	assertStatus(t, rec, http.StatusBadRequest)
	assertBodyContains(t, rec, "query")
}

func TestSemanticSearchInvalidJSON(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodPost, "/api/v1/search/semantic", "{bad")
	assertStatus(t, rec, http.StatusBadRequest)
	assertBodyContains(t, rec, "invalid")
}
