package server

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/welife-os/welife-os/engine/internal/graph"
)

func (s *Server) handleTriggerGraphBuild(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req struct {
		ConversationID string `json:"conversation_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.ConversationID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "conversation_id is required"})
		return
	}

	taskID, err := s.graphEngine.BuildGraph(r.Context(), req.ConversationID)
	if err != nil {
		log.Printf("graph-build: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to start graph build"})
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{
		"task_id": taskID,
		"status":  "building",
	})
}

func (s *Server) handleGraphOverview(w http.ResponseWriter, r *http.Request) {
	overview, err := s.graphEngine.GetOverview(r.Context())
	if err != nil {
		log.Printf("graph-overview: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load graph overview"})
		return
	}
	writeJSON(w, http.StatusOK, overview)
}

// handleGetGraphNode returns a single node's detail plus its direct neighbors and edges.
// GET /api/v1/graph/nodes/{id}
func (s *Server) handleGetGraphNode(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if strings.TrimSpace(id) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "node id is required"})
		return
	}

	entity, err := s.store.GetEntity(r.Context(), id)
	if err != nil {
		writeResourceError(w, "graph-node", err, "failed to get node")
		return
	}

	gs := s.graphEngine.GraphStore()
	neighborIDs := gs.Neighbors(id)

	// Look up full entity data for each neighbor.
	neighbors := make([]graph.Node, 0, len(neighborIDs))
	for _, nid := range neighborIDs {
		ne, err := s.store.GetEntity(r.Context(), nid)
		if err != nil {
			log.Printf("graph-node: failed to resolve neighbor %s: %v", nid, err)
			continue
		}
		neighbors = append(neighbors, graph.Node{ID: ne.ID, Type: ne.Type, Name: ne.Name})
	}

	// Fetch edges involving this node.
	rels, err := s.store.GetRelationships(r.Context(), id)
	if err != nil {
		log.Printf("graph-node-edges: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load edges"})
		return
	}
	edges := make([]graph.Edge, 0, len(rels))
	for _, rel := range rels {
		edges = append(edges, graph.Edge{
			ID: rel.ID, Source: rel.SourceEntityID, Target: rel.TargetEntityID,
			Type: rel.Type, Weight: rel.Weight,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"node":      graph.Node{ID: entity.ID, Type: entity.Type, Name: entity.Name},
		"neighbors": neighbors,
		"edges":     edges,
	})
}

// handleGetGraphNeighborhood returns a local subgraph centered on a node.
// GET /api/v1/graph/nodes/{id}/neighborhood?depth=1
func (s *Server) handleGetGraphNeighborhood(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if strings.TrimSpace(id) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "node id is required"})
		return
	}

	depth := 1
	if d := r.URL.Query().Get("depth"); d != "" {
		parsed, err := strconv.Atoi(d)
		if err != nil || parsed < 1 || parsed > 2 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "depth must be 1 or 2"})
			return
		}
		depth = parsed
	}

	// Verify center node exists.
	if !s.graphEngine.GraphStore().HasNode(id) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "resource not found"})
		return
	}

	// BFS to collect nodes within the requested depth (capped to prevent runaway).
	const maxBFSNodes = 500
	gs := s.graphEngine.GraphStore()
	visited := map[string]bool{id: true}
	frontier := []string{id}

	for d := 0; d < depth; d++ {
		var nextFrontier []string
		for _, nid := range frontier {
			for _, neighbor := range gs.Neighbors(nid) {
				if !visited[neighbor] {
					visited[neighbor] = true
					nextFrontier = append(nextFrontier, neighbor)
					if len(visited) >= maxBFSNodes {
						break
					}
				}
			}
			if len(visited) >= maxBFSNodes {
				break
			}
		}
		frontier = nextFrontier
	}

	// Resolve entity details and collect edges.
	nodeMap := make(map[string]graph.Node, len(visited))
	for nid := range visited {
		ent, err := s.store.GetEntity(r.Context(), nid)
		if err != nil {
			log.Printf("graph-neighborhood: failed to resolve node %s: %v", nid, err)
			continue
		}
		nodeMap[nid] = graph.Node{ID: ent.ID, Type: ent.Type, Name: ent.Name}
	}

	nodes := make([]graph.Node, 0, len(nodeMap))
	for _, n := range nodeMap {
		nodes = append(nodes, n)
	}

	// Collect edges where both endpoints are in the subgraph.
	edgeSet := make(map[string]bool)
	var edges []graph.Edge
	for nid := range visited {
		rels, err := s.store.GetRelationships(r.Context(), nid)
		if err != nil {
			log.Printf("graph-neighborhood: failed to load edges for %s: %v", nid, err)
			continue
		}
		for _, rel := range rels {
			if visited[rel.SourceEntityID] && visited[rel.TargetEntityID] && !edgeSet[rel.ID] {
				edgeSet[rel.ID] = true
				edges = append(edges, graph.Edge{
					ID: rel.ID, Source: rel.SourceEntityID, Target: rel.TargetEntityID,
					Type: rel.Type, Weight: rel.Weight,
				})
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"nodes":     nodes,
		"edges":     edges,
		"center_id": id,
	})
}

// handleGraphSearch searches nodes by name (case-insensitive).
// GET /api/v1/graph/search?q=query
func (s *Server) handleGraphSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "q query parameter is required"})
		return
	}

	const maxSearchResults = 50
	entities, err := s.store.SearchEntitiesByName(r.Context(), q, maxSearchResults)
	if err != nil {
		log.Printf("graph-search: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to search nodes"})
		return
	}

	gs := s.graphEngine.GraphStore()
	type searchResult struct {
		ID     string `json:"id"`
		Type   string `json:"type"`
		Name   string `json:"name"`
		Degree int    `json:"degree"`
	}

	results := make([]searchResult, 0, len(entities))
	for _, ent := range entities {
		results = append(results, searchResult{
			ID:     ent.ID,
			Type:   ent.Type,
			Name:   ent.Name,
			Degree: gs.Degree(ent.ID),
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"results": results,
	})
}
