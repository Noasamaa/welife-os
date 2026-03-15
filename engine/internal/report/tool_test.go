package report

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/welife-os/welife-os/engine/internal/storage"
)

func newToolTestStore(t *testing.T) *storage.Store {
	t.Helper()
	store, err := storage.Open(context.Background(), storage.Config{
		Path: t.TempDir() + "/report_tool_test.db",
		Key:  "test-key",
	})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func TestGraphSearchToolScopesEntitiesAndRelationshipsByConversation(t *testing.T) {
	ctx := context.Background()
	store := newToolTestStore(t)

	if err := store.SaveEntity(ctx, storage.Entity{
		ID:                 "e_conv_a_1",
		Type:               "person",
		Name:               "Alice",
		SourceConversation: "conv_a",
	}); err != nil {
		t.Fatalf("save entity: %v", err)
	}
	if err := store.SaveEntity(ctx, storage.Entity{
		ID:                 "e_conv_a_2",
		Type:               "person",
		Name:               "Bob",
		SourceConversation: "conv_a",
	}); err != nil {
		t.Fatalf("save entity: %v", err)
	}
	if err := store.SaveEntity(ctx, storage.Entity{
		ID:                 "e_conv_b_1",
		Type:               "person",
		Name:               "Mallory",
		SourceConversation: "conv_b",
	}); err != nil {
		t.Fatalf("save entity: %v", err)
	}
	if err := store.SaveRelationships(ctx, []storage.Relationship{
		{
			ID:             "rel_a",
			SourceEntityID: "e_conv_a_1",
			TargetEntityID: "e_conv_a_2",
			Type:           "friend",
			Weight:         0.8,
		},
		{
			ID:             "rel_b",
			SourceEntityID: "e_conv_b_1",
			TargetEntityID: "e_conv_a_1",
			Type:           "knows",
			Weight:         0.4,
		},
	}); err != nil {
		t.Fatalf("save relationships: %v", err)
	}

	tool := NewGraphSearchTool(store)
	result, err := tool.Execute(ctx, map[string]string{
		"conversation_id": "conv_a",
	})
	if err != nil {
		t.Fatalf("execute tool: %v", err)
	}

	var payload struct {
		Entities      []storage.Entity       `json:"entities"`
		Relationships []storage.Relationship `json:"relationships"`
	}
	if err := json.Unmarshal([]byte(result), &payload); err != nil {
		t.Fatalf("decode result: %v", err)
	}

	if len(payload.Entities) != 2 {
		t.Fatalf("entity count = %d, want 2", len(payload.Entities))
	}
	if len(payload.Relationships) != 1 {
		t.Fatalf("relationship count = %d, want 1", len(payload.Relationships))
	}
	if payload.Relationships[0].ID != "rel_a" {
		t.Fatalf("relationship id = %q, want rel_a", payload.Relationships[0].ID)
	}
}
