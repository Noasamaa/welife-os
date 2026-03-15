package task

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type Status string

const (
	StatusQueued    Status = "queued"
	StatusRunning   Status = "running"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
)

type JobFunc func(context.Context) error

type Info struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Status    Status    `json:"status"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type job struct {
	id string
	fn JobFunc
}

type Manager struct {
	cancel func()
	jobs   chan job
	wg     sync.WaitGroup
	close  sync.Once

	mu    sync.RWMutex
	seq   uint64
	tasks map[string]Info
}

const taskTTL = 1 * time.Hour

func NewManager(workerCount int) *Manager {
	if workerCount < 1 {
		workerCount = 1
	}

	ctx, cancel := context.WithCancel(context.Background())
	manager := &Manager{
		cancel: cancel,
		jobs:   make(chan job, workerCount*2),
		tasks:  make(map[string]Info),
	}

	for i := 0; i < workerCount; i++ {
		manager.wg.Add(1)
		go manager.worker(ctx)
	}

	manager.wg.Add(1)
	go manager.cleaner(ctx)

	return manager
}

func (m *Manager) Submit(name string, fn JobFunc) string {
	id := fmt.Sprintf("task_%04d", atomic.AddUint64(&m.seq, 1))
	now := time.Now()

	m.mu.Lock()
	m.tasks[id] = Info{
		ID:        id,
		Name:      name,
		Status:    StatusQueued,
		CreatedAt: now,
		UpdatedAt: now,
	}
	m.mu.Unlock()

	m.jobs <- job{id: id, fn: fn}
	return id
}

func (m *Manager) Status(id string) (Info, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	info, ok := m.tasks[id]
	return info, ok
}

func (m *Manager) List() []Info {
	m.mu.RLock()
	defer m.mu.RUnlock()

	items := make([]Info, 0, len(m.tasks))
	for _, item := range m.tasks {
		items = append(items, item)
	}
	sort.Slice(items, func(i int, j int) bool {
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
	return items
}

func (m *Manager) Close() error {
	m.close.Do(func() {
		m.cancel()
		close(m.jobs)
		m.wg.Wait()
	})
	return nil
}

func (m *Manager) worker(ctx context.Context) {
	defer m.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-m.jobs:
			if !ok {
				return
			}
			m.runJob(ctx, job)
		}
	}
}

func (m *Manager) runJob(ctx context.Context, current job) {
	m.update(current.id, func(info *Info) {
		info.Status = StatusRunning
		info.UpdatedAt = time.Now()
	})

	err := current.fn(ctx)
	if err != nil && !errors.Is(err, context.Canceled) {
		m.update(current.id, func(info *Info) {
			info.Status = StatusFailed
			info.Error = err.Error()
			info.UpdatedAt = time.Now()
		})
		return
	}

	m.update(current.id, func(info *Info) {
		info.Status = StatusSucceeded
		info.UpdatedAt = time.Now()
	})
}

func (m *Manager) update(id string, updateFn func(*Info)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	info := m.tasks[id]
	updateFn(&info)
	m.tasks[id] = info
}

// cleaner periodically removes completed/failed tasks older than taskTTL.
func (m *Manager) cleaner(ctx context.Context) {
	defer m.wg.Done()
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.pruneExpired()
		}
	}
}

func (m *Manager) pruneExpired() {
	cutoff := time.Now().Add(-taskTTL)
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, info := range m.tasks {
		if (info.Status == StatusSucceeded || info.Status == StatusFailed) && info.UpdatedAt.Before(cutoff) {
			delete(m.tasks, id)
		}
	}
}
