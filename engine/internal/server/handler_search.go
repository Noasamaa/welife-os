package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/welife-os/welife-os/engine/internal/storage"
)

func (s *Server) handleBuildEmbeddings(w http.ResponseWriter, r *http.Request) {
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

	if !s.vectorStore.Ready() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "vector store is not available"})
		return
	}

	taskID := s.taskManager.Submit("embed:"+req.ConversationID, func(taskCtx context.Context) error {
		return s.buildEmbeddingsSync(taskCtx, req.ConversationID)
	})

	writeJSON(w, http.StatusAccepted, map[string]string{
		"task_id": taskID,
		"status":  "building",
	})
}

func (s *Server) buildEmbeddingsSync(ctx context.Context, conversationID string) error {
	const batchSize = 50
	offset := 0
	embedded := 0

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		msgs, err := s.store.GetMessages(ctx, conversationID, batchSize, offset)
		if err != nil {
			return fmt.Errorf("loading messages: %w", err)
		}
		if len(msgs) == 0 {
			break
		}

		for _, m := range msgs {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			vec, err := s.llmClient.Embed(ctx, m.Content)
			if err != nil {
				log.Printf("embed: skip message %s: %v", m.ID, err)
				continue
			}

			if err := s.vectorStore.StoreEmbedding(m.ID, vec, nil); err != nil {
				log.Printf("embed: store failed for %s: %v", m.ID, err)
				continue
			}
			embedded++
		}
		offset += batchSize
	}

	log.Printf("embed: built %d embeddings for conversation %s", embedded, conversationID)
	return nil
}

func (s *Server) handleSemanticSearch(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Query == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "query is required"})
		return
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}

	if !s.vectorStore.Ready() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "vector store is not available"})
		return
	}

	queryVec, err := s.llmClient.Embed(r.Context(), req.Query)
	if err != nil {
		log.Printf("semantic-search: embed query: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to embed query"})
		return
	}

	results, err := s.vectorStore.Search(queryVec, req.Limit)
	if err != nil {
		log.Printf("semantic-search: search: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "search failed"})
		return
	}

	if results == nil {
		results = []storage.VectorResult{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"results": results,
	})
}
