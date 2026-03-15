package graph

import (
	"context"
	"fmt"
	"strings"

	"github.com/welife-os/welife-os/engine/internal/storage"
	"github.com/welife-os/welife-os/engine/internal/task"
)

// Engine orchestrates entity extraction and graph construction.
type Engine struct {
	store     *storage.Store
	extractor *Extractor
	graph     *GraphStore
	tasks     *task.Manager
}

// NewEngine creates a new graph engine.
func NewEngine(store *storage.Store, extractor *Extractor, tasks *task.Manager) *Engine {
	return &Engine{
		store:     store,
		extractor: extractor,
		graph:     NewGraphStore(),
		tasks:     tasks,
	}
}

// GraphStore returns the in-memory graph store for cloning and queries.
func (e *Engine) GraphStore() *GraphStore {
	return e.graph
}

// BuildGraph triggers async entity extraction for a conversation.
// Returns a task ID for progress tracking.
func (e *Engine) BuildGraph(ctx context.Context, conversationID string) (string, error) {
	// Verify conversation exists
	_, err := e.store.GetConversation(ctx, conversationID)
	if err != nil {
		return "", fmt.Errorf("conversation not found: %w", err)
	}

	taskID := e.tasks.Submit("graph:"+conversationID, func(taskCtx context.Context) error {
		return e.buildGraphSync(taskCtx, conversationID)
	})

	return taskID, nil
}

// buildGraphSync runs the full extraction pipeline synchronously.
func (e *Engine) buildGraphSync(ctx context.Context, conversationID string) error {
	// Load messages in batches
	const batchSize = 50
	offset := 0

	var allEntities []ExtractedEntity
	var allRelationships []ExtractedRelationship

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		msgs, err := e.store.GetMessages(ctx, conversationID, batchSize, offset)
		if err != nil {
			return fmt.Errorf("loading messages: %w", err)
		}
		if len(msgs) == 0 {
			break
		}

		// Convert to snippets for LLM
		snippets := make([]MessageSnippet, len(msgs))
		for i, m := range msgs {
			snippets[i] = MessageSnippet{
				Timestamp:  m.Timestamp,
				SenderName: m.SenderName,
				Content:    m.Content,
			}
		}

		result, err := e.extractor.Extract(ctx, snippets)
		if err != nil {
			// Log but continue with next batch
			offset += batchSize
			continue
		}

		allEntities = append(allEntities, result.Entities...)
		allRelationships = append(allRelationships, result.Relationships...)
		offset += batchSize
	}

	// Deduplicate entities by name+type
	entityMap := make(map[string]ExtractedEntity)
	for _, ent := range allEntities {
		key := strings.ToLower(string(ent.Type) + ":" + ent.Name)
		entityMap[key] = ent
	}

	// Save to database and build in-memory graph
	entityIDMap := make(map[string]string) // name -> entity ID

	var storageEntities []storage.Entity
	seq := 0
	for _, ent := range entityMap {
		seq++
		id := fmt.Sprintf("e_%s_%d", conversationID, seq)
		entityIDMap[ent.Name] = id

		storageEntities = append(storageEntities, storage.Entity{
			ID:                 id,
			Type:               string(ent.Type),
			Name:               ent.Name,
			SourceConversation: conversationID,
		})

		e.graph.AddNode(id)
	}

	if err := e.store.SaveEntities(ctx, storageEntities); err != nil {
		return fmt.Errorf("saving entities: %w", err)
	}

	// Build relationships
	var storageRels []storage.Relationship
	relSeq := 0
	for _, rel := range allRelationships {
		sourceID, ok1 := entityIDMap[rel.SourceName]
		targetID, ok2 := entityIDMap[rel.TargetName]
		if !ok1 || !ok2 {
			continue
		}

		relSeq++
		id := fmt.Sprintf("r_%s_%d", conversationID, relSeq)

		storageRels = append(storageRels, storage.Relationship{
			ID:             id,
			SourceEntityID: sourceID,
			TargetEntityID: targetID,
			Type:           rel.Type,
			Weight:         1.0,
		})

		_ = e.graph.AddEdge(sourceID, targetID, 1.0)
	}

	if len(storageRels) > 0 {
		if err := e.store.SaveRelationships(ctx, storageRels); err != nil {
			return fmt.Errorf("saving relationships: %w", err)
		}
	}

	return nil
}

// GetOverview returns the full graph for visualization.
func (e *Engine) GetOverview(ctx context.Context) (GraphOverview, error) {
	entities, err := e.store.ListEntities(ctx)
	if err != nil {
		return GraphOverview{}, err
	}

	rels, err := e.store.ListRelationships(ctx)
	if err != nil {
		return GraphOverview{}, err
	}

	return e.graph.Overview(entities, rels), nil
}
