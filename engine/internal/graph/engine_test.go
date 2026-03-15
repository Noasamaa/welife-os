package graph

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/welife-os/welife-os/engine/internal/storage"
)

func newTestEngine(t *testing.T) (*Engine, *storage.Store) {
	t.Helper()
	store, err := storage.Open(context.Background(), storage.Config{
		Path: filepath.Join(t.TempDir(), "welife.db"),
		Key:  "test-key",
	})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	eng := NewEngine(store, nil, nil)
	return eng, store
}

func TestLoadRestoresGraphFromDatabase(t *testing.T) {
	eng, store := newTestEngine(t)
	ctx := context.Background()

	entities := []storage.Entity{
		{ID: "e1", Type: "person", Name: "Alice", SourceConversation: "conv1"},
		{ID: "e2", Type: "person", Name: "Bob", SourceConversation: "conv1"},
		{ID: "e3", Type: "topic", Name: "Travel", SourceConversation: "conv1"},
	}
	if err := store.SaveEntities(ctx, entities); err != nil {
		t.Fatalf("save entities: %v", err)
	}

	rels := []storage.Relationship{
		{ID: "r1", SourceEntityID: "e1", TargetEntityID: "e2", Type: "friend", Weight: 1.0},
		{ID: "r2", SourceEntityID: "e1", TargetEntityID: "e3", Type: "discusses", Weight: 2.0},
	}
	if err := store.SaveRelationships(ctx, rels); err != nil {
		t.Fatalf("save relationships: %v", err)
	}

	if err := eng.Load(ctx); err != nil {
		t.Fatalf("Load: %v", err)
	}

	gs := eng.GraphStore()
	if gs.NodeCount() != 3 {
		t.Errorf("NodeCount = %d, want 3", gs.NodeCount())
	}
	if gs.EdgeCount() != 2 {
		t.Errorf("EdgeCount = %d, want 2", gs.EdgeCount())
	}
}

func TestLoadRestoresNeighbors(t *testing.T) {
	eng, store := newTestEngine(t)
	ctx := context.Background()

	entities := []storage.Entity{
		{ID: "a", Type: "person", Name: "Alice", SourceConversation: "c1"},
		{ID: "b", Type: "person", Name: "Bob", SourceConversation: "c1"},
		{ID: "c", Type: "person", Name: "Carol", SourceConversation: "c1"},
	}
	if err := store.SaveEntities(ctx, entities); err != nil {
		t.Fatalf("save entities: %v", err)
	}

	rels := []storage.Relationship{
		{ID: "r1", SourceEntityID: "a", TargetEntityID: "b", Type: "friend", Weight: 1.0},
		{ID: "r2", SourceEntityID: "c", TargetEntityID: "a", Type: "colleague", Weight: 1.0},
	}
	if err := store.SaveRelationships(ctx, rels); err != nil {
		t.Fatalf("save relationships: %v", err)
	}

	if err := eng.Load(ctx); err != nil {
		t.Fatalf("Load: %v", err)
	}

	neighbors := eng.GraphStore().Neighbors("a")
	if len(neighbors) != 2 {
		t.Fatalf("Neighbors(a) = %v, want 2 neighbors (b, c)", neighbors)
	}

	found := map[string]bool{}
	for _, n := range neighbors {
		found[n] = true
	}
	if !found["b"] || !found["c"] {
		t.Errorf("Neighbors(a) = %v, want both b and c", neighbors)
	}
}

func TestLoadEmptyDatabase(t *testing.T) {
	eng, _ := newTestEngine(t)

	if err := eng.Load(context.Background()); err != nil {
		t.Fatalf("Load on empty db: %v", err)
	}

	gs := eng.GraphStore()
	if gs.NodeCount() != 0 {
		t.Errorf("NodeCount = %d, want 0", gs.NodeCount())
	}
	if gs.EdgeCount() != 0 {
		t.Errorf("EdgeCount = %d, want 0", gs.EdgeCount())
	}
}

func TestLoadPreservesWeights(t *testing.T) {
	eng, store := newTestEngine(t)
	ctx := context.Background()

	entities := []storage.Entity{
		{ID: "x", Type: "person", Name: "X", SourceConversation: "c1"},
		{ID: "y", Type: "person", Name: "Y", SourceConversation: "c1"},
	}
	if err := store.SaveEntities(ctx, entities); err != nil {
		t.Fatalf("save entities: %v", err)
	}

	rels := []storage.Relationship{
		{ID: "r1", SourceEntityID: "x", TargetEntityID: "y", Type: "link", Weight: 2.5},
	}
	if err := store.SaveRelationships(ctx, rels); err != nil {
		t.Fatalf("save relationships: %v", err)
	}

	if err := eng.Load(ctx); err != nil {
		t.Fatalf("Load: %v", err)
	}

	// Verify weight survives Clone (the simulation path).
	cloned := eng.GraphStore().Clone()
	edges := cloned.AllEdges()
	if len(edges) != 1 {
		t.Fatalf("AllEdges count = %d, want 1", len(edges))
	}
	if edges[0].Weight != 2.5 {
		t.Errorf("edge weight = %f, want 2.5", edges[0].Weight)
	}
}
