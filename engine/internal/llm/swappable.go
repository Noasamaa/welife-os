package llm

import (
	"context"
	"fmt"
	"sync"
)

// maxConcurrentLLMCalls limits the number of concurrent LLM requests to
// prevent overloading the upstream provider.
const maxConcurrentLLMCalls = 8

// SwappableClient wraps an LLMClient with a read-write mutex, allowing the
// underlying client to be hot-swapped at runtime while all existing callers
// continue to use the same *SwappableClient pointer.
// It also enforces a concurrency limit on Generate and Embed calls.
type SwappableClient struct {
	mu    sync.RWMutex
	inner LLMClient
	sem   chan struct{} // concurrency semaphore
}

// NewSwappable creates a new SwappableClient wrapping the given initial client.
func NewSwappable(initial LLMClient) *SwappableClient {
	return &SwappableClient{
		inner: initial,
		sem:   make(chan struct{}, maxConcurrentLLMCalls),
	}
}

// Swap replaces the underlying LLMClient with next. All subsequent calls to
// Generate, Embed, Reachable, and Status will be delegated to next.
func (s *SwappableClient) Swap(next LLMClient) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inner = next
}

// acquireSem blocks until a semaphore slot is available or ctx is cancelled.
func (s *SwappableClient) acquireSem(ctx context.Context) error {
	select {
	case s.sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("LLM concurrency limit: %w", ctx.Err())
	}
}

func (s *SwappableClient) releaseSem() {
	<-s.sem
}

// Generate delegates to the current inner client under a read lock,
// with concurrency limiting.
func (s *SwappableClient) Generate(ctx context.Context, prompt string) (string, error) {
	if err := s.acquireSem(ctx); err != nil {
		return "", err
	}
	defer s.releaseSem()

	s.mu.RLock()
	c := s.inner
	s.mu.RUnlock()
	return c.Generate(ctx, prompt)
}

// Embed delegates to the current inner client under a read lock,
// with concurrency limiting.
func (s *SwappableClient) Embed(ctx context.Context, text string) ([]float32, error) {
	if err := s.acquireSem(ctx); err != nil {
		return nil, err
	}
	defer s.releaseSem()

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
