package reminder

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/welife-os/welife-os/engine/internal/storage"
)

const checkInterval = 15 * time.Minute

// Scheduler runs periodic checks on reminder rules.
type Scheduler struct {
	store   *storage.Store
	checker *Checker
	ticker  *time.Ticker
	done    chan struct{}
}

// NewScheduler creates a new scheduler.
func NewScheduler(store *storage.Store, checker *Checker) *Scheduler {
	return &Scheduler{
		store:   store,
		checker: checker,
		done:    make(chan struct{}),
	}
}

// Start begins the periodic check loop in a goroutine.
func (s *Scheduler) Start(ctx context.Context) {
	s.ticker = time.NewTicker(checkInterval)
	go s.run(ctx)
}

// Stop stops the scheduler.
func (s *Scheduler) Stop() {
	if s.ticker != nil {
		s.ticker.Stop()
	}
	close(s.done)
}

func (s *Scheduler) run(ctx context.Context) {
	// Run once immediately on start.
	s.checkRules(ctx)

	for {
		select {
		case <-s.ticker.C:
			s.checkRules(ctx)
		case <-s.done:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (s *Scheduler) checkRules(ctx context.Context) {
	rules, err := s.store.ListEnabledReminderRules(ctx)
	if err != nil {
		log.Printf("reminder scheduler: listing rules: %v", err)
		return
	}

	now := time.Now()

	for _, rule := range rules {
		shouldFire, message, err := s.checker.Evaluate(ctx, rule, now)
		if err != nil {
			log.Printf("reminder scheduler: evaluating rule %s: %v", rule.ID, err)
			continue
		}

		if !shouldFire {
			continue
		}

		rem := storage.Reminder{
			ID:      generateID(),
			RuleID:  rule.ID,
			Message: message,
			Status:  "pending",
		}

		if err := s.store.CreateReminder(ctx, rem); err != nil {
			log.Printf("reminder scheduler: creating reminder for rule %s: %v", rule.ID, err)
			continue
		}

		if err := s.store.UpdateRuleLastTriggered(ctx, rule.ID); err != nil {
			log.Printf("reminder scheduler: updating last_triggered for rule %s: %v", rule.ID, err)
		}
	}
}

// generateID produces a unique ID for reminders.
func generateID() string {
	return fmt.Sprintf("rem_%d", time.Now().UnixNano())
}
