package server_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/welife-os/welife-os/engine/internal/server"
	"github.com/welife-os/welife-os/engine/internal/storage"
	"github.com/welife-os/welife-os/engine/internal/testutil"
)

func TestTriggerGraphBuildMissingConversationID(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodPost, "/api/v1/graph/build",
		map[string]string{})
	assertStatus(t, rec, http.StatusBadRequest)
	assertBodyContains(t, rec, "conversation_id")
}

func TestTriggerGraphBuildInvalidJSON(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodPost, "/api/v1/graph/build", "{bad")
	assertStatus(t, rec, http.StatusBadRequest)
	assertBodyContains(t, rec, "invalid")
}

func TestGraphOverview(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodGet, "/api/v1/graph/overview", nil)
	assertStatus(t, rec, http.StatusOK)
}

// --- Node detail endpoint ---

func TestGetGraphNodeSuccess(t *testing.T) {
	app, cleanup := newSeededTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodGet, "/api/v1/graph/nodes/e_alice", nil)
	assertStatus(t, rec, http.StatusOK)

	body := decodeJSON[map[string]any](t, rec)
	node, ok := body["node"].(map[string]any)
	if !ok {
		t.Fatal("expected 'node' object in response")
	}
	if node["id"] != "e_alice" {
		t.Fatalf("node.id = %v, want e_alice", node["id"])
	}
	if node["name"] != "Alice" {
		t.Fatalf("node.name = %v, want Alice", node["name"])
	}

	// Alice has neighbors (bob is connected via a relationship)
	neighbors, ok := body["neighbors"].([]any)
	if !ok {
		t.Fatal("expected 'neighbors' array in response")
	}
	if len(neighbors) == 0 {
		t.Fatal("expected at least one neighbor for Alice")
	}

	edges, ok := body["edges"].([]any)
	if !ok {
		t.Fatal("expected 'edges' array in response")
	}
	if len(edges) == 0 {
		t.Fatal("expected at least one edge for Alice")
	}
}

func TestGetGraphNodeNotFound(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodGet, "/api/v1/graph/nodes/nonexistent", nil)
	assertStatus(t, rec, http.StatusNotFound)
	assertBodyContains(t, rec, "not found")
}

// --- Neighborhood endpoint ---

func TestGetGraphNeighborhoodDepth1(t *testing.T) {
	app, cleanup := newSeededTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodGet, "/api/v1/graph/nodes/e_alice/neighborhood?depth=1", nil)
	assertStatus(t, rec, http.StatusOK)

	body := decodeJSON[map[string]any](t, rec)
	if body["center_id"] != "e_alice" {
		t.Fatalf("center_id = %v, want e_alice", body["center_id"])
	}

	nodes, ok := body["nodes"].([]any)
	if !ok {
		t.Fatal("expected 'nodes' array in response")
	}
	// depth=1: alice + bob (direct neighbor)
	if len(nodes) < 2 {
		t.Fatalf("expected at least 2 nodes at depth 1, got %d", len(nodes))
	}
}

func TestGetGraphNeighborhoodDepth2(t *testing.T) {
	app, cleanup := newSeededTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodGet, "/api/v1/graph/nodes/e_alice/neighborhood?depth=2", nil)
	assertStatus(t, rec, http.StatusOK)

	body := decodeJSON[map[string]any](t, rec)
	nodes, ok := body["nodes"].([]any)
	if !ok {
		t.Fatal("expected 'nodes' array in response")
	}
	// depth=2: alice -> bob -> carol
	if len(nodes) < 3 {
		t.Fatalf("expected at least 3 nodes at depth 2, got %d", len(nodes))
	}
}

func TestGetGraphNeighborhoodInvalidDepth(t *testing.T) {
	app, cleanup := newSeededTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodGet, "/api/v1/graph/nodes/e_alice/neighborhood?depth=5", nil)
	assertStatus(t, rec, http.StatusBadRequest)
	assertBodyContains(t, rec, "depth must be 1 or 2")
}

func TestGetGraphNeighborhoodNotFound(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodGet, "/api/v1/graph/nodes/nonexistent/neighborhood", nil)
	assertStatus(t, rec, http.StatusNotFound)
	assertBodyContains(t, rec, "not found")
}

// --- Search endpoint ---

func TestGraphSearchSuccess(t *testing.T) {
	app, cleanup := newSeededTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodGet, "/api/v1/graph/search?q=alice", nil)
	assertStatus(t, rec, http.StatusOK)

	body := decodeJSON[map[string]any](t, rec)
	results, ok := body["results"].([]any)
	if !ok {
		t.Fatal("expected 'results' array in response")
	}
	if len(results) == 0 {
		t.Fatal("expected at least one search result for 'alice'")
	}

	first := results[0].(map[string]any)
	if first["name"] != "Alice" {
		t.Fatalf("first result name = %v, want Alice", first["name"])
	}
	if _, ok := first["degree"]; !ok {
		t.Fatal("expected 'degree' field in search result")
	}
}

func TestGraphSearchCaseInsensitive(t *testing.T) {
	app, cleanup := newSeededTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodGet, "/api/v1/graph/search?q=ALICE", nil)
	assertStatus(t, rec, http.StatusOK)

	body := decodeJSON[map[string]any](t, rec)
	results := body["results"].([]any)
	if len(results) == 0 {
		t.Fatal("expected case-insensitive search to find 'Alice'")
	}
}

func TestGraphSearchNoResults(t *testing.T) {
	app, cleanup := newSeededTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodGet, "/api/v1/graph/search?q=zzzznonexistent", nil)
	assertStatus(t, rec, http.StatusOK)

	body := decodeJSON[map[string]any](t, rec)
	results := body["results"].([]any)
	if len(results) != 0 {
		t.Fatalf("expected empty results, got %d", len(results))
	}
}

func TestGraphSearchMissingQuery(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodGet, "/api/v1/graph/search", nil)
	assertStatus(t, rec, http.StatusBadRequest)
	assertBodyContains(t, rec, "q query parameter is required")
}

func TestGraphSearchEmptyQuery(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodGet, "/api/v1/graph/search?q=", nil)
	assertStatus(t, rec, http.StatusBadRequest)
	assertBodyContains(t, rec, "q query parameter is required")
}

// --- test helper ---

// newSeededTestApp creates a test server with pre-seeded entities and
// relationships so that graph endpoints return meaningful data.
// Graph: alice -> bob -> carol
func newSeededTestApp(t *testing.T) (*server.Server, func()) {
	t.Helper()

	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models":[]}`))
	}))

	cfg := testutil.NewServerConfig(t, ollama.URL)

	// Open store to seed test data before the server starts.
	store, err := storage.Open(t.Context(), storage.Config{
		Path: cfg.DatabasePath,
		Key:  cfg.DatabaseKey,
	})
	if err != nil {
		ollama.Close()
		t.Fatalf("open store for seeding: %v", err)
	}

	entities := []storage.Entity{
		{ID: "e_alice", Type: "person", Name: "Alice", SourceConversation: "conv_test"},
		{ID: "e_bob", Type: "person", Name: "Bob", SourceConversation: "conv_test"},
		{ID: "e_carol", Type: "person", Name: "Carol", SourceConversation: "conv_test"},
	}
	if err := store.SaveEntities(t.Context(), entities); err != nil {
		_ = store.Close()
		ollama.Close()
		t.Fatalf("seed entities: %v", err)
	}

	rels := []storage.Relationship{
		{ID: "r_ab", SourceEntityID: "e_alice", TargetEntityID: "e_bob", Type: "friend", Weight: 1.0},
		{ID: "r_bc", SourceEntityID: "e_bob", TargetEntityID: "e_carol", Type: "colleague", Weight: 1.0},
	}
	if err := store.SaveRelationships(t.Context(), rels); err != nil {
		_ = store.Close()
		ollama.Close()
		t.Fatalf("seed relationships: %v", err)
	}

	if err := store.Close(); err != nil {
		ollama.Close()
		t.Fatalf("close seed store: %v", err)
	}

	// Now create server — graphEngine.Load() will pick up seeded data.
	app, err := server.New(cfg)
	if err != nil {
		ollama.Close()
		t.Fatalf("newSeededTestApp: %v", err)
	}

	cleanup := func() {
		_ = app.Shutdown(t.Context())
		ollama.Close()
	}
	return app, cleanup
}
