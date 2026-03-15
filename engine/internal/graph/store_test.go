package graph

import (
	"testing"

	"github.com/welife-os/welife-os/engine/internal/storage"
)

func TestGraphStoreAddNodeAndEdge(t *testing.T) {
	gs := NewGraphStore()

	gs.AddNode("e1")
	gs.AddNode("e2")
	gs.AddNode("e3")

	if gs.NodeCount() != 3 {
		t.Errorf("NodeCount() = %d, want 3", gs.NodeCount())
	}

	if err := gs.AddEdge("e1", "e2", 1.0); err != nil {
		t.Fatalf("AddEdge: %v", err)
	}
	if err := gs.AddEdge("e2", "e3", 2.0); err != nil {
		t.Fatalf("AddEdge: %v", err)
	}

	if gs.EdgeCount() != 2 {
		t.Errorf("EdgeCount() = %d, want 2", gs.EdgeCount())
	}

	// Adding same edge should increase weight
	if err := gs.AddEdge("e1", "e2", 3.0); err != nil {
		t.Fatalf("AddEdge (duplicate): %v", err)
	}
	if gs.EdgeCount() != 2 {
		t.Errorf("EdgeCount after duplicate = %d, want 2", gs.EdgeCount())
	}
}

func TestGraphStoreNeighbors(t *testing.T) {
	gs := NewGraphStore()
	gs.AddNode("a")
	gs.AddNode("b")
	gs.AddNode("c")
	gs.AddNode("d")

	_ = gs.AddEdge("a", "b", 1.0)
	_ = gs.AddEdge("c", "a", 1.0)

	neighbors := gs.Neighbors("a")
	if len(neighbors) != 2 {
		t.Errorf("Neighbors(a) = %v, want 2 neighbors (b, c)", neighbors)
	}

	// d has no connections
	neighbors = gs.Neighbors("d")
	if len(neighbors) != 0 {
		t.Errorf("Neighbors(d) = %v, want empty", neighbors)
	}

	// Unknown node
	neighbors = gs.Neighbors("unknown")
	if neighbors != nil {
		t.Errorf("Neighbors(unknown) = %v, want nil", neighbors)
	}
}

func TestGraphStoreAddEdgeUnknownNode(t *testing.T) {
	gs := NewGraphStore()
	gs.AddNode("a")

	if err := gs.AddEdge("a", "unknown", 1.0); err == nil {
		t.Error("expected error for unknown target node")
	}
	if err := gs.AddEdge("unknown", "a", 1.0); err == nil {
		t.Error("expected error for unknown source node")
	}
}

func TestGraphStoreOverview(t *testing.T) {
	gs := NewGraphStore()
	gs.AddNode("e1")
	gs.AddNode("e2")
	_ = gs.AddEdge("e1", "e2", 1.5)

	entities := []storage.Entity{
		{ID: "e1", Type: "person", Name: "Alice"},
		{ID: "e2", Type: "topic", Name: "AI"},
	}
	rels := []storage.Relationship{
		{ID: "r1", SourceEntityID: "e1", TargetEntityID: "e2", Type: "discusses", Weight: 1.5},
	}

	overview := gs.Overview(entities, rels)

	if overview.Stats.EntityCount != 2 {
		t.Errorf("EntityCount = %d, want 2", overview.Stats.EntityCount)
	}
	if overview.Stats.RelationshipCount != 1 {
		t.Errorf("RelationshipCount = %d, want 1", overview.Stats.RelationshipCount)
	}
	if overview.Stats.EntityTypes["person"] != 1 {
		t.Errorf("EntityTypes[person] = %d, want 1", overview.Stats.EntityTypes["person"])
	}
	if len(overview.Nodes) != 2 {
		t.Errorf("Nodes count = %d, want 2", len(overview.Nodes))
	}
	if len(overview.Edges) != 1 {
		t.Errorf("Edges count = %d, want 1", len(overview.Edges))
	}
}

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			"plain json",
			`{"entities":[],"relationships":[]}`,
			`{"entities":[],"relationships":[]}`,
		},
		{
			"with code block",
			"```json\n{\"entities\":[]}\n```",
			`{"entities":[]}`,
		},
		{
			"with text around",
			"Here is the result:\n{\"entities\":[]}\nDone.",
			`{"entities":[]}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractJSON(tt.input)
			if got != tt.want {
				t.Errorf("extractJSON() = %q, want %q", got, tt.want)
			}
		})
	}
}
