package reminder_test

import (
	"context"
	"testing"
	"time"

	"github.com/welife-os/welife-os/engine/internal/reminder"
	"github.com/welife-os/welife-os/engine/internal/storage"
)

func openTestStore(t *testing.T) *storage.Store {
	t.Helper()
	store, err := storage.Open(context.Background(), storage.Config{
		Path: t.TempDir() + "/reminder_test.db",
		Key:  "test-key",
	})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func seedEntityAndMessages(t *testing.T, store *storage.Store, entityID, entityName, lastMsgTime string) {
	t.Helper()
	ctx := context.Background()

	if err := store.SaveEntity(ctx, storage.Entity{
		ID:   entityID,
		Type: "person",
		Name: entityName,
	}); err != nil {
		t.Fatalf("save entity: %v", err)
	}

	if lastMsgTime != "" {
		if err := store.SaveMessages(ctx, []storage.StoredMessage{
			{
				ID:             "msg_" + entityID,
				ConversationID: "conv_test",
				Platform:       "test",
				SenderID:       entityID,
				SenderName:     entityName,
				Content:        "hello",
				MessageType:    "text",
				Timestamp:      lastMsgTime,
			},
		}); err != nil {
			t.Fatalf("save messages: %v", err)
		}
	}
}

func seedConversation(t *testing.T, store *storage.Store) {
	t.Helper()
	db := store.DB()
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO conversations (id, platform, conversation_type, title, message_count)
		VALUES ('conv_test', 'test', 'private', 'Test', 0)`)
	if err != nil {
		t.Fatalf("seed conversation: %v", err)
	}
}

func TestCheckerContactGapFires(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	seedConversation(t, store)

	// Entity with a message 30 days ago
	lastMsg := time.Now().Add(-30 * 24 * time.Hour).Format("2006-01-02 15:04:05")
	seedEntityAndMessages(t, store, "ent_alice", "Alice", lastMsg)

	checker := reminder.NewChecker(store)
	rule := storage.ReminderRule{
		ID:              "rule_cg1",
		RuleType:        "contact_gap",
		EntityID:        "ent_alice",
		ThresholdDays:   14,
		MessageTemplate: "No contact with Alice for {{days}} days",
		Enabled:         true,
	}

	shouldFire, msg, err := checker.Evaluate(ctx, rule, time.Now())
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if !shouldFire {
		t.Fatal("expected rule to fire")
	}
	if msg == "" {
		t.Fatal("expected non-empty message")
	}
}

func TestCheckerContactGapDoesNotFire(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	seedConversation(t, store)

	// Entity with a recent message (1 day ago)
	lastMsg := time.Now().Add(-1 * 24 * time.Hour).Format("2006-01-02 15:04:05")
	seedEntityAndMessages(t, store, "ent_bob", "Bob", lastMsg)

	checker := reminder.NewChecker(store)
	rule := storage.ReminderRule{
		ID:              "rule_cg2",
		RuleType:        "contact_gap",
		EntityID:        "ent_bob",
		ThresholdDays:   14,
		MessageTemplate: "No contact for {{days}} days",
		Enabled:         true,
	}

	shouldFire, _, err := checker.Evaluate(ctx, rule, time.Now())
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if shouldFire {
		t.Fatal("expected rule NOT to fire")
	}
}

func TestCheckerContactGapNoMessages(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	// Entity with no messages at all
	if err := store.SaveEntity(ctx, storage.Entity{
		ID:   "ent_nomsgs",
		Type: "person",
		Name: "NoMessages",
	}); err != nil {
		t.Fatalf("save entity: %v", err)
	}

	checker := reminder.NewChecker(store)
	rule := storage.ReminderRule{
		ID:              "rule_cg3",
		RuleType:        "contact_gap",
		EntityID:        "ent_nomsgs",
		ThresholdDays:   7,
		MessageTemplate: "No messages from contact for {{days}} days",
		Enabled:         true,
	}

	shouldFire, _, err := checker.Evaluate(ctx, rule, time.Now())
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if !shouldFire {
		t.Fatal("expected rule to fire when no messages exist")
	}
}

func TestCheckerDeadlineFires(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	// Seed an action item with a due date 2 days from now
	dueDate := time.Now().Add(2 * 24 * time.Hour).Format("2006-01-02")
	db := store.DB()
	_, err := db.ExecContext(ctx, `
		INSERT INTO action_items (id, source_agent, title, description, priority, status, category, due_date)
		VALUES ('ai_001', 'test', 'Test Item', 'desc', 'high', 'pending', 'general', ?)`, dueDate)
	if err != nil {
		t.Fatalf("seed action item: %v", err)
	}

	checker := reminder.NewChecker(store)
	rule := storage.ReminderRule{
		ID:              "rule_dl1",
		RuleType:        "deadline",
		ActionItemID:    "ai_001",
		MessageTemplate: "Action item due {{urgency}} ({{days}} days)",
		Enabled:         true,
	}

	shouldFire, msg, err := checker.Evaluate(ctx, rule, time.Now())
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if !shouldFire {
		t.Fatal("expected deadline rule to fire within 7 days")
	}
	if msg == "" {
		t.Fatal("expected non-empty message")
	}
}

func TestCheckerDeadlineDoesNotFire(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	// Seed an action item with a due date 30 days from now
	dueDate := time.Now().Add(30 * 24 * time.Hour).Format("2006-01-02")
	db := store.DB()
	_, err := db.ExecContext(ctx, `
		INSERT INTO action_items (id, source_agent, title, description, priority, status, category, due_date)
		VALUES ('ai_002', 'test', 'Far Away', 'desc', 'low', 'pending', 'general', ?)`, dueDate)
	if err != nil {
		t.Fatalf("seed action item: %v", err)
	}

	checker := reminder.NewChecker(store)
	rule := storage.ReminderRule{
		ID:              "rule_dl2",
		RuleType:        "deadline",
		ActionItemID:    "ai_002",
		MessageTemplate: "Due {{urgency}}",
		Enabled:         true,
	}

	shouldFire, _, err := checker.Evaluate(ctx, rule, time.Now())
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if shouldFire {
		t.Fatal("expected deadline rule NOT to fire when due date is far away")
	}
}

func TestCheckerPeriodicFirstRun(t *testing.T) {
	checker := reminder.NewChecker(openTestStore(t))
	ctx := context.Background()

	rule := storage.ReminderRule{
		ID:              "rule_p1",
		RuleType:        "periodic",
		CronExpr:        "daily",
		MessageTemplate: "Daily check-in",
		Enabled:         true,
		LastTriggeredAt: "",
	}

	shouldFire, msg, err := checker.Evaluate(ctx, rule, time.Now())
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if !shouldFire {
		t.Fatal("expected periodic rule to fire on first run")
	}
	if msg != "Daily check-in" {
		t.Fatalf("expected 'Daily check-in', got %q", msg)
	}
}

func TestCheckerPeriodicNotYet(t *testing.T) {
	checker := reminder.NewChecker(openTestStore(t))
	ctx := context.Background()

	rule := storage.ReminderRule{
		ID:              "rule_p2",
		RuleType:        "periodic",
		CronExpr:        "daily",
		MessageTemplate: "Daily check-in",
		Enabled:         true,
		LastTriggeredAt: time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
	}

	shouldFire, _, err := checker.Evaluate(ctx, rule, time.Now())
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if shouldFire {
		t.Fatal("expected periodic rule NOT to fire (only 1 hour since last trigger)")
	}
}

func TestCheckerPeriodicReady(t *testing.T) {
	checker := reminder.NewChecker(openTestStore(t))
	ctx := context.Background()

	rule := storage.ReminderRule{
		ID:              "rule_p3",
		RuleType:        "periodic",
		CronExpr:        "daily",
		MessageTemplate: "Daily check-in",
		Enabled:         true,
		LastTriggeredAt: time.Now().Add(-25 * time.Hour).Format(time.RFC3339),
	}

	shouldFire, _, err := checker.Evaluate(ctx, rule, time.Now())
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if !shouldFire {
		t.Fatal("expected periodic rule to fire (25 hours since last trigger)")
	}
}

func TestCheckerPeriodicEveryDuration(t *testing.T) {
	checker := reminder.NewChecker(openTestStore(t))
	ctx := context.Background()

	rule := storage.ReminderRule{
		ID:              "rule_p4",
		RuleType:        "periodic",
		CronExpr:        "@every 2h",
		MessageTemplate: "Check every 2 hours",
		Enabled:         true,
		LastTriggeredAt: time.Now().Add(-3 * time.Hour).Format(time.RFC3339),
	}

	shouldFire, _, err := checker.Evaluate(ctx, rule, time.Now())
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if !shouldFire {
		t.Fatal("expected @every 2h rule to fire (3 hours since last trigger)")
	}
}

func TestCheckerUnknownRuleType(t *testing.T) {
	checker := reminder.NewChecker(openTestStore(t))
	ctx := context.Background()

	rule := storage.ReminderRule{
		ID:       "rule_unknown",
		RuleType: "bogus",
	}

	_, _, err := checker.Evaluate(ctx, rule, time.Now())
	if err == nil {
		t.Fatal("expected error for unknown rule type")
	}
}

func TestReminderStoreCRUD(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	rule := storage.ReminderRule{
		ID:              "rule_crud",
		RuleType:        "periodic",
		CronExpr:        "weekly",
		MessageTemplate: "Weekly review",
		Enabled:         true,
	}
	if err := store.CreateReminderRule(ctx, rule); err != nil {
		t.Fatalf("create rule: %v", err)
	}

	got, err := store.GetReminderRule(ctx, "rule_crud")
	if err != nil {
		t.Fatalf("get rule: %v", err)
	}
	if got.RuleType != "periodic" {
		t.Fatalf("expected periodic, got %s", got.RuleType)
	}
	if !got.Enabled {
		t.Fatal("expected enabled=true")
	}

	// Disable the rule
	if err := store.UpdateReminderRule(ctx, "rule_crud", false); err != nil {
		t.Fatalf("update rule: %v", err)
	}
	got, _ = store.GetReminderRule(ctx, "rule_crud")
	if got.Enabled {
		t.Fatal("expected enabled=false after update")
	}

	// Create a reminder
	rem := storage.Reminder{
		ID:      "rem_001",
		RuleID:  "rule_crud",
		Message: "Weekly review reminder",
		Status:  "pending",
	}
	if err := store.CreateReminder(ctx, rem); err != nil {
		t.Fatalf("create reminder: %v", err)
	}

	pending, err := store.ListPendingReminders(ctx)
	if err != nil {
		t.Fatalf("list pending: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending, got %d", len(pending))
	}

	// Mark read
	if err := store.MarkReminderRead(ctx, "rem_001"); err != nil {
		t.Fatalf("mark read: %v", err)
	}
	pending, _ = store.ListPendingReminders(ctx)
	if len(pending) != 0 {
		t.Fatalf("expected 0 pending after mark read, got %d", len(pending))
	}

	// Delete rule
	if err := store.DeleteReminderRule(ctx, "rule_crud"); err != nil {
		t.Fatalf("delete rule: %v", err)
	}
	_, err = store.GetReminderRule(ctx, "rule_crud")
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestDismissReminder(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	// Create rule first (FK constraint)
	if err := store.CreateReminderRule(ctx, storage.ReminderRule{
		ID:              "rule_dismiss",
		RuleType:        "periodic",
		CronExpr:        "daily",
		MessageTemplate: "test",
		Enabled:         true,
	}); err != nil {
		t.Fatalf("create rule: %v", err)
	}

	if err := store.CreateReminder(ctx, storage.Reminder{
		ID:      "rem_dismiss",
		RuleID:  "rule_dismiss",
		Message: "dismiss me",
		Status:  "pending",
	}); err != nil {
		t.Fatalf("create reminder: %v", err)
	}

	if err := store.DismissReminder(ctx, "rem_dismiss"); err != nil {
		t.Fatalf("dismiss: %v", err)
	}

	pending, _ := store.ListPendingReminders(ctx)
	if len(pending) != 0 {
		t.Fatalf("expected 0 pending after dismiss, got %d", len(pending))
	}
}
