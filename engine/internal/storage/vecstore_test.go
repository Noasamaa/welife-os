package storage_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/welife-os/welife-os/engine/internal/storage"
)

const vecDim = 768

func openVecTestStore(t *testing.T) *storage.Store {
	t.Helper()
	store, err := storage.Open(context.Background(), storage.Config{
		Path: filepath.Join(t.TempDir(), "welife.db"),
		Key:  "test-key",
	})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

// makeVec creates a 768-dim vector with value at the given index, rest zeros.
func makeVec(index int, value float32) []float32 {
	v := make([]float32, vecDim)
	if index < vecDim {
		v[index] = value
	}
	return v
}

func TestSqliteVecStoreReady(t *testing.T) {
	store := openVecTestStore(t)
	vs := storage.NewSqliteVecStore(store.DB())
	if !vs.Ready() {
		t.Fatal("SqliteVecStore.Ready() = false, want true")
	}
}

func TestSqliteVecStoreRoundTrip(t *testing.T) {
	store := openVecTestStore(t)
	vs := storage.NewSqliteVecStore(store.DB())

	// Two orthogonal embeddings: e1 peaks at dim 0, e2 at dim 1.
	e1 := makeVec(0, 1.0)
	e2 := makeVec(1, 1.0)

	if err := vs.StoreEmbedding("msg_1", e1, nil); err != nil {
		t.Fatalf("store e1: %v", err)
	}
	if err := vs.StoreEmbedding("msg_2", e2, nil); err != nil {
		t.Fatalf("store e2: %v", err)
	}

	// Query close to e1.
	query := makeVec(0, 0.9)
	query[1] = 0.1

	results, err := vs.Search(query, 2)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].ID != "msg_1" {
		t.Errorf("closest result = %q, want msg_1", results[0].ID)
	}
}

func TestSqliteVecStoreSearchEmpty(t *testing.T) {
	store := openVecTestStore(t)
	vs := storage.NewSqliteVecStore(store.DB())

	query := makeVec(0, 1.0)
	results, err := vs.Search(query, 5)
	if err != nil {
		t.Fatalf("search empty: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}
