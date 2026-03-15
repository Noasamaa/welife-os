package simulation_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/welife-os/welife-os/engine/internal/graph"
	"github.com/welife-os/welife-os/engine/internal/llm"
	"github.com/welife-os/welife-os/engine/internal/simulation"
	"github.com/welife-os/welife-os/engine/internal/storage"
	"github.com/welife-os/welife-os/engine/internal/task"
)

func newMockOllamaServer(t *testing.T, responseBody string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/generate" {
			w.Header().Set("Content-Type", "application/json")
			resp := map[string]any{
				"model":      "test",
				"response":   responseBody,
				"done":       true,
				"created_at": "2024-01-01T00:00:00Z",
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		if r.URL.Path == "/api/tags" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"models": []any{}})
			return
		}
		http.NotFound(w, r)
	}))
}

func newTestLLMClient(t *testing.T, server *httptest.Server) *llm.Client {
	t.Helper()
	client, err := llm.New(llm.Config{
		BaseURL: server.URL,
		Model:   "test",
		Timeout: 0,
	})
	if err != nil {
		t.Fatalf("failed to create LLM client: %v", err)
	}
	return client
}

func newTestStore(t *testing.T) *storage.Store {
	t.Helper()
	store, err := storage.Open(context.Background(), storage.Config{
		Path: t.TempDir() + "/test.db",
		Key:  "test-key",
	})
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func TestProfileBuilderBuildAllEmpty(t *testing.T) {
	profileResp := `{"personality": "友善开朗", "relationship_to_self": "朋友关系", "behavioral_patterns": "积极主动"}`
	server := newMockOllamaServer(t, profileResp)
	defer server.Close()

	client := newTestLLMClient(t, server)
	store := newTestStore(t)

	builder := simulation.NewProfileBuilder(client, store)

	profiles, err := builder.BuildAllProfiles(context.Background(), "conv_test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(profiles) != 0 {
		t.Fatalf("expected 0 profiles for empty store, got %d", len(profiles))
	}
}

func TestProfileBuilderBuildProfile(t *testing.T) {
	profileResp := `{"personality": "友善开朗", "relationship_to_self": "好朋友", "behavioral_patterns": "积极主动"}`
	server := newMockOllamaServer(t, profileResp)
	defer server.Close()

	client := newTestLLMClient(t, server)
	store := newTestStore(t)
	ctx := context.Background()

	err := store.SaveConversation(ctx, storage.Conversation{
		ID:               "conv_test",
		Platform:         "test",
		ConversationType: "private",
		Title:            "测试对话",
	})
	if err != nil {
		t.Fatalf("failed to save conversation: %v", err)
	}

	// Seed entity.
	err = store.SaveEntity(ctx, storage.Entity{
		ID:                 "e_test_1",
		Type:               "person",
		Name:               "Alice",
		SourceConversation: "conv_test",
	})
	if err != nil {
		t.Fatalf("failed to save entity: %v", err)
	}

	builder := simulation.NewProfileBuilder(client, store)

	profile, err := builder.BuildProfile(ctx, "conv_test", "e_test_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if profile.Name != "Alice" {
		t.Fatalf("expected name Alice, got %s", profile.Name)
	}
	if profile.Personality != "友善开朗" {
		t.Fatalf("unexpected personality: %s", profile.Personality)
	}
	if profile.EntityID != "e_test_1" {
		t.Fatalf("expected entity ID e_test_1, got %s", profile.EntityID)
	}
}

func TestSimulationEngineRunAndComplete(t *testing.T) {
	// LLM returns different responses based on context. We use a single
	// mock that returns valid JSON for all prompts.
	callCount := int64(0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/generate" {
			n := atomic.AddInt64(&callCount, 1)
			var responseBody string
			// Profile building calls return profile JSON.
			// Simulation step calls return reaction JSON.
			// Narrative call returns plain text.
			if n <= 2 {
				responseBody = `{"personality": "友善", "relationship_to_self": "朋友", "behavioral_patterns": "积极"}`
			} else if n <= 6 {
				responseBody = `{"reaction": "积极面对", "actions": ["调整计划"], "relationship_changes": [{"target": "Bob", "change": "更加亲近", "weight_delta": 0.2}]}`
			} else {
				responseBody = "在这个平行世界中，一切都发生了变化..."
			}

			w.Header().Set("Content-Type", "application/json")
			resp := map[string]any{
				"model":      "test",
				"response":   responseBody,
				"done":       true,
				"created_at": "2024-01-01T00:00:00Z",
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		if r.URL.Path == "/api/tags" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"models": []any{}})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := newTestLLMClient(t, server)
	store := newTestStore(t)
	tasks := task.NewManager(1)
	defer tasks.Close()

	ctx := context.Background()

	_ = store.SaveConversation(ctx, storage.Conversation{
		ID:               "conv_test",
		Platform:         "test",
		ConversationType: "private",
		Title:            "测试对话",
	})

	// Seed entities.
	_ = store.SaveEntity(ctx, storage.Entity{ID: "e1", Type: "person", Name: "Alice", SourceConversation: "conv_test"})
	_ = store.SaveEntity(ctx, storage.Entity{ID: "e2", Type: "person", Name: "Bob", SourceConversation: "conv_test"})
	_ = store.SaveRelationships(ctx, []storage.Relationship{{
		ID: "r1", SourceEntityID: "e1", TargetEntityID: "e2",
		Type: "friend", Weight: 1.0,
	}})

	graphStore := graph.NewGraphStore()
	graphStore.AddNode("e1")
	graphStore.AddNode("e2")
	_ = graphStore.AddEdge("e1", "e2", 1.0)

	profiler := simulation.NewProfileBuilder(client, store)
	engine := simulation.NewEngine(client, store, tasks, profiler, graphStore)

	config := simulation.SimulationConfig{
		ConversationID: "conv_test",
		Steps:          2,
		ForkPoint: simulation.ForkPoint{
			Description:   "Alice decided to move to a new city",
			AffectedNodes: []string{"e1"},
			Changes:       map[string]string{"location": "new_city"},
		},
	}

	sessionID, taskID, err := engine.RunSimulation(ctx, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sessionID == "" {
		t.Fatal("expected non-empty session ID")
	}
	if taskID == "" {
		t.Fatal("expected non-empty task ID")
	}

	// Wait for task to complete.
	for {
		info, ok := tasks.Status(taskID)
		if !ok {
			t.Fatal("task not found")
		}
		if info.Status == task.StatusSucceeded || info.Status == task.StatusFailed {
			if info.Status == task.StatusFailed {
				t.Fatalf("simulation task failed: %s", info.Error)
			}
			break
		}
	}

	// Verify session was completed.
	sess, err := engine.GetSession(ctx, sessionID)
	if err != nil {
		t.Fatalf("failed to get session: %v", err)
	}
	if sess.Status != "completed" {
		t.Fatalf("expected status completed, got %s", sess.Status)
	}
	if sess.Narrative == "" {
		t.Fatal("expected non-empty narrative")
	}

	// Verify steps were saved.
	steps, err := engine.GetSessionSteps(ctx, sessionID)
	if err != nil {
		t.Fatalf("failed to get steps: %v", err)
	}
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(steps))
	}
}

func TestSimulationEngineRequiresForkDescription(t *testing.T) {
	server := newMockOllamaServer(t, "{}")
	defer server.Close()

	client := newTestLLMClient(t, server)
	store := newTestStore(t)
	tasks := task.NewManager(1)
	defer tasks.Close()

	graphStore := graph.NewGraphStore()
	profiler := simulation.NewProfileBuilder(client, store)
	engine := simulation.NewEngine(client, store, tasks, profiler, graphStore)

	_ = store.SaveConversation(context.Background(), storage.Conversation{
		ID:               "conv_required",
		Platform:         "test",
		ConversationType: "private",
		Title:            "测试对话",
	})

	_, _, err := engine.RunSimulation(context.Background(), simulation.SimulationConfig{
		ConversationID: "conv_required",
		Steps:          3,
	})
	if err == nil {
		t.Fatal("expected error for empty fork description")
	}
}

func TestSimulationEngineListSessions(t *testing.T) {
	server := newMockOllamaServer(t, "{}")
	defer server.Close()

	client := newTestLLMClient(t, server)
	store := newTestStore(t)
	tasks := task.NewManager(1)
	defer tasks.Close()

	graphStore := graph.NewGraphStore()
	profiler := simulation.NewProfileBuilder(client, store)
	engine := simulation.NewEngine(client, store, tasks, profiler, graphStore)

	sessions, err := engine.ListSessions(context.Background(), "conv_missing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sessions != nil {
		t.Fatalf("expected nil sessions for empty store, got %d", len(sessions))
	}
}
