package storage

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
)

// SqliteVecStore implements VectorStore using the sqlite-vec vec0 virtual table.
type SqliteVecStore struct {
	db *sql.DB
}

// NewSqliteVecStore creates a vector store backed by the vec_messages table.
func NewSqliteVecStore(db *sql.DB) *SqliteVecStore {
	return &SqliteVecStore{db: db}
}

// StoreEmbedding inserts a vector into the vec_messages virtual table.
func (s *SqliteVecStore) StoreEmbedding(id string, embedding []float32, _ map[string]string) error {
	blob := serializeFloat32(embedding)
	_, err := s.db.Exec(
		"INSERT OR REPLACE INTO vec_messages(rowid, embedding, source_id) VALUES (NULL, ?, ?)",
		blob, id,
	)
	if err != nil {
		return fmt.Errorf("vecstore: store embedding: %w", err)
	}
	return nil
}

// Search performs a KNN similarity search against the vec_messages table.
func (s *SqliteVecStore) Search(query []float32, limit int) ([]VectorResult, error) {
	if limit <= 0 {
		limit = 10
	}

	blob := serializeFloat32(query)
	rows, err := s.db.Query(
		"SELECT source_id, distance FROM vec_messages WHERE embedding MATCH ? AND k = ?",
		blob, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("vecstore: search: %w", err)
	}
	defer rows.Close()

	var results []VectorResult
	for rows.Next() {
		var r VectorResult
		if err := rows.Scan(&r.ID, &r.Distance); err != nil {
			return nil, fmt.Errorf("vecstore: scan result: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// Ready checks whether the vec0 virtual table is accessible.
func (s *SqliteVecStore) Ready() bool {
	var version string
	err := s.db.QueryRow("SELECT vec_version()").Scan(&version)
	return err == nil && version != ""
}

// serializeFloat32 converts a float32 slice to little-endian binary format
// as expected by sqlite-vec's vec0 virtual table.
func serializeFloat32(v []float32) []byte {
	buf := make([]byte, len(v)*4)
	for i, f := range v {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(f))
	}
	return buf
}
