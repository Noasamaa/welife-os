package llm

import (
	"context"
	"sync"
)

// SwappableClient wraps an LLMClient with a read-write mutex, allowing the
// underlying client to be hot-swapped at runtime while all existing callers
// continue to use the same *SwappableClient pointer.
type SwappableClient struct {
	mu    sync.RWMutex
	inner LLMClient
}

// NewSwappable creates a new SwappableClient wrapping the given initial client.
func NewSwappable(initial LLMClient) *SwappableClient {
	return &SwappableClient{inner: initial}
}

// Swap replaces the underlying LLMClient with next. All subsequent calls to
// Generate, Embed, Reachable, and Status will be delegated to next.
func (s *SwappableClient) Swap(next LLMClient) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inner = next
}

// Generate delegates to the current inner client under a read lock.
func (s *SwappableClient) Generate(ctx context.Context, prompt string) (string, error) {
	s.mu.RLock()
	c := s.inner
	s.mu.RUnlock()
	return c.Generate(ctx, prompt)
}

// Embed delegates to the current inner client under a read lock.
func (s *SwappableClient) Embed(ctx context.Context, text string) ([]float32, error) {
	s.mu.RLock()
	c := s.inner
	s.mu.RUnlock()
	return c.Embed(ctx, text)
}

// Reachable delegates to the current inner client under a read lock.
func (s *SwappableClient) Reachable(ctx context.Context) (bool, error) {
	s.mu.RLock()
	c := s.inner
	s.mu.RUnlock()
	return c.Reachable(ctx)
}

// Status delegates to the current inner client under a read lock.
func (s *SwappableClient) Status(ctx context.Context) StatusInfo {
	s.mu.RLock()
	c := s.inner
	s.mu.RUnlock()
	return c.Status(ctx)
}
