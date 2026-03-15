package graph

import (
	"fmt"
	"sync"

	"github.com/welife-os/welife-os/engine/internal/storage"
	"gonum.org/v1/gonum/graph/simple"
)

// GraphStore holds an in-memory weighted directed graph backed by gonum.
type GraphStore struct {
	g       *simple.WeightedDirectedGraph
	nodeIDs map[string]int64 // entity ID -> gonum node ID
	idToKey map[int64]string // gonum node ID -> entity ID
	seq     int64
	mu      sync.RWMutex
}

// NewGraphStore creates an empty in-memory graph.
func NewGraphStore() *GraphStore {
	return &GraphStore{
		g:       simple.NewWeightedDirectedGraph(0, 0),
		nodeIDs: make(map[string]int64),
		idToKey: make(map[int64]string),
	}
}

// AddNode adds an entity as a graph node. Returns the gonum node ID.
func (gs *GraphStore) AddNode(entityID string) int64 {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	if id, ok := gs.nodeIDs[entityID]; ok {
		return id
	}

	gs.seq++
	n := simple.Node(gs.seq)
	gs.g.AddNode(n)
	gs.nodeIDs[entityID] = gs.seq
	gs.idToKey[gs.seq] = entityID
	return gs.seq
}

// AddEdge adds a weighted directed edge between two entities.
func (gs *GraphStore) AddEdge(fromEntityID, toEntityID string, weight float64) error {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	fromID, ok := gs.nodeIDs[fromEntityID]
	if !ok {
		return fmt.Errorf("node %q not found", fromEntityID)
	}
	toID, ok := gs.nodeIDs[toEntityID]
	if !ok {
		return fmt.Errorf("node %q not found", toEntityID)
	}

	e := gs.g.WeightedEdge(fromID, toID)
	if e != nil {
		// Edge exists, increase weight
		gs.g.RemoveEdge(fromID, toID)
		gs.g.SetWeightedEdge(simple.WeightedEdge{
			F: simple.Node(fromID), T: simple.Node(toID), W: e.Weight() + weight,
		})
	} else {
		gs.g.SetWeightedEdge(simple.WeightedEdge{
			F: simple.Node(fromID), T: simple.Node(toID), W: weight,
		})
	}
	return nil
}

// Neighbors returns entity IDs directly connected to the given entity.
func (gs *GraphStore) Neighbors(entityID string) []string {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	nodeID, ok := gs.nodeIDs[entityID]
	if !ok {
		return nil
	}

	seen := make(map[string]bool)
	var result []string

	// Outgoing neighbors
	to := gs.g.From(nodeID)
	for to.Next() {
		key := gs.idToKey[to.Node().ID()]
		if !seen[key] {
			seen[key] = true
			result = append(result, key)
		}
	}

	// Incoming neighbors
	from := gs.g.To(nodeID)
	for from.Next() {
		key := gs.idToKey[from.Node().ID()]
		if !seen[key] {
			seen[key] = true
			result = append(result, key)
		}
	}

	return result
}

// NodeCount returns the number of nodes.
func (gs *GraphStore) NodeCount() int {
	gs.mu.RLock()
	defer gs.mu.RUnlock()
	return gs.g.Nodes().Len()
}

// EdgeCount returns the number of edges.
func (gs *GraphStore) EdgeCount() int {
	gs.mu.RLock()
	defer gs.mu.RUnlock()
	return gs.g.Edges().Len()
}

// Overview builds a GraphOverview from the in-memory graph and entity storage.
func (gs *GraphStore) Overview(entities []storage.Entity, relationships []storage.Relationship) GraphOverview {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	nodes := make([]Node, len(entities))
	for i, e := range entities {
		nodes[i] = Node{
			ID:   e.ID,
			Type: e.Type,
			Name: e.Name,
		}
	}

	edges := make([]Edge, len(relationships))
	for i, r := range relationships {
		edges[i] = Edge{
			ID:     r.ID,
			Source: r.SourceEntityID,
			Target: r.TargetEntityID,
			Type:   r.Type,
			Weight: r.Weight,
		}
	}

	typeCounts := make(map[string]int)
	for _, e := range entities {
		typeCounts[e.Type]++
	}

	return GraphOverview{
		Nodes: nodes,
		Edges: edges,
		Stats: Stats{
			EntityCount:       len(entities),
			RelationshipCount: len(relationships),
			EntityTypes:       typeCounts,
		},
	}
}

// GraphOverview is the visualization-ready graph representation.
type GraphOverview struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
	Stats Stats  `json:"stats"`
}

// Node is a graph node for visualization.
type Node struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Name string `json:"name"`
}

// Edge is a graph edge for visualization.
type Edge struct {
	ID     string  `json:"id"`
	Source string  `json:"source"`
	Target string  `json:"target"`
	Type   string  `json:"type"`
	Weight float64 `json:"weight"`
}

// Stats holds graph statistics.
type Stats struct {
	EntityCount       int            `json:"entity_count"`
	RelationshipCount int            `json:"relationship_count"`
	EntityTypes       map[string]int `json:"entity_types"`
}
