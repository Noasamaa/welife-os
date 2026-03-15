package storage

import "errors"

// ErrVectorStoreUnavailable is returned when sqlite-vec extension is not loaded.
var ErrVectorStoreUnavailable = errors.New("vector store is not available: sqlite-vec extension not loaded")

// VectorStore provides vector embedding storage and similarity search.
type VectorStore interface {
	// StoreEmbedding stores a vector embedding with associated metadata.
	StoreEmbedding(id string, embedding []float32, metadata map[string]string) error

	// Search finds the nearest vectors to the query embedding.
	Search(query []float32, limit int) ([]VectorResult, error)

	// Ready reports whether the vector store is operational.
	Ready() bool
}

// VectorResult represents a single similarity search result.
type VectorResult struct {
	ID       string            `json:"id"`
	Distance float32           `json:"distance"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// NoopVectorStore is a fallback that always reports unavailable.
type NoopVectorStore struct{}

func (NoopVectorStore) StoreEmbedding(string, []float32, map[string]string) error {
	return ErrVectorStoreUnavailable
}

func (NoopVectorStore) Search([]float32, int) ([]VectorResult, error) {
	return nil, ErrVectorStoreUnavailable
}

func (NoopVectorStore) Ready() bool { return false }
