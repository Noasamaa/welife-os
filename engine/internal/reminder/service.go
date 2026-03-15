package reminder

import (
	"context"

	"github.com/welife-os/welife-os/engine/internal/storage"
)

// Service provides high-level reminder operations and manages the scheduler.
type Service struct {
	store     *storage.Store
	scheduler *Scheduler
}

// NewService creates a new reminder service.
func NewService(store *storage.Store) *Service {
	checker := NewChecker(store)
	scheduler := NewScheduler(store, checker)
	return &Service{
		store:     store,
		scheduler: scheduler,
	}
}

// Start begins the background scheduler.
func (s *Service) Start(ctx context.Context) {
	s.scheduler.Start(ctx)
}

// Stop halts the background scheduler.
func (s *Service) Stop() {
	s.scheduler.Stop()
}

// ListPending returns all pending reminders.
func (s *Service) ListPending(ctx context.Context) ([]storage.Reminder, error) {
	return s.store.ListPendingReminders(ctx)
}

// MarkRead marks a reminder as read.
func (s *Service) MarkRead(ctx context.Context, id string) error {
	return s.store.MarkReminderRead(ctx, id)
}

// Dismiss dismisses a reminder.
func (s *Service) Dismiss(ctx context.Context, id string) error {
	return s.store.DismissReminder(ctx, id)
}
