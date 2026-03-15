package storage

import "errors"

var ErrVectorStoreNotImplemented = errors.New("sqlite-vec is not implemented in phase 0")

type VectorStore struct{}

func NewVectorStore() (*VectorStore, error) {
	return nil, ErrVectorStoreNotImplemented
}
