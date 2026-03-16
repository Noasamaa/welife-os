package storage

import (
	"context"
	"database/sql"
	"fmt"
)

// CreateReminderRule inserts a new reminder rule.
func (s *Store) CreateReminderRule(ctx context.Context, rule ReminderRule) error {
	enabled := 0
	if rule.Enabled {
		enabled = 1
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO reminder_rules (id, action_item_id, rule_type, entity_id, threshold_days, cron_expr, message_template, enabled)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		rule.ID, rule.ActionItemID, rule.RuleType, rule.EntityID,
		rule.ThresholdDays, rule.CronExpr, rule.MessageTemplate, enabled)
	return err
}

// ListReminderRules returns all reminder rules ordered by creation time.
func (s *Store) ListReminderRules(ctx context.Context) ([]ReminderRule, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, COALESCE(action_item_id,''), rule_type, COALESCE(entity_id,''),
		       COALESCE(threshold_days,0), COALESCE(cron_expr,''), message_template,
		       enabled, COALESCE(last_triggered_at,''), created_at
		FROM reminder_rules ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []ReminderRule
	for rows.Next() {
		var r ReminderRule
		var enabled int
		if err := rows.Scan(&r.ID, &r.ActionItemID, &r.RuleType, &r.EntityID,
			&r.ThresholdDays, &r.CronExpr, &r.MessageTemplate,
			&enabled, &r.LastTriggeredAt, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.Enabled = enabled != 0
		rules = append(rules, r)
	}
	return rules, rows.Err()
}

// ListEnabledReminderRules returns only enabled reminder rules.
func (s *Store) ListEnabledReminderRules(ctx context.Context) ([]ReminderRule, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, COALESCE(action_item_id,''), rule_type, COALESCE(entity_id,''),
		       COALESCE(threshold_days,0), COALESCE(cron_expr,''), message_template,
		       enabled, COALESCE(last_triggered_at,''), created_at
		FROM reminder_rules WHERE enabled = 1 ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []ReminderRule
	for rows.Next() {
		var r ReminderRule
		var enabled int
		if err := rows.Scan(&r.ID, &r.ActionItemID, &r.RuleType, &r.EntityID,
			&r.ThresholdDays, &r.CronExpr, &r.MessageTemplate,
			&enabled, &r.LastTriggeredAt, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.Enabled = enabled != 0
		rules = append(rules, r)
	}
	return rules, rows.Err()
}

// GetReminderRule returns a single reminder rule by ID.
func (s *Store) GetReminderRule(ctx context.Context, id string) (ReminderRule, error) {
	var r ReminderRule
	var enabled int
	err := s.db.QueryRowContext(ctx, `
		SELECT id, COALESCE(action_item_id,''), rule_type, COALESCE(entity_id,''),
		       COALESCE(threshold_days,0), COALESCE(cron_expr,''), message_template,
		       enabled, COALESCE(last_triggered_at,''), created_at
		FROM reminder_rules WHERE id = ?`, id).
		Scan(&r.ID, &r.ActionItemID, &r.RuleType, &r.EntityID,
			&r.ThresholdDays, &r.CronExpr, &r.MessageTemplate,
			&enabled, &r.LastTriggeredAt, &r.CreatedAt)
	if err == sql.ErrNoRows {
		return r, fmt.Errorf("reminder rule %q: %w", id, ErrNotFound)
	}
	r.Enabled = enabled != 0
	return r, err
}

// UpdateReminderRule updates the enabled status of a reminder rule.
func (s *Store) UpdateReminderRule(ctx context.Context, id string, enabled bool) error {
	v := 0
	if enabled {
		v = 1
	}
	result, err := s.db.ExecContext(ctx, `
		UPDATE reminder_rules SET enabled = ? WHERE id = ?`, v, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("reminder rule %q: %w", id, ErrNotFound)
	}
	return nil
}

// UpdateRuleLastTriggered sets last_triggered_at to the current timestamp.
func (s *Store) UpdateRuleLastTriggered(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE reminder_rules SET last_triggered_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
	return err
}

// DeleteReminderRule removes a reminder rule by ID.
func (s *Store) DeleteReminderRule(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM reminder_rules WHERE id = ?`, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("reminder rule %q: %w", id, ErrNotFound)
	}
	return nil
}

// CreateReminder inserts a new reminder.
func (s *Store) CreateReminder(ctx context.Context, rem Reminder) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO reminders (id, rule_id, message, status)
		VALUES (?, ?, ?, ?)`,
		rem.ID, rem.RuleID, rem.Message, rem.Status)
	return err
}

// ListPendingReminders returns all pending reminders ordered by trigger time.
func (s *Store) ListPendingReminders(ctx context.Context) ([]Reminder, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, rule_id, message, status, triggered_at, COALESCE(read_at,'')
		FROM reminders WHERE status = 'pending'
		ORDER BY triggered_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reminders []Reminder
	for rows.Next() {
		var rem Reminder
		if err := rows.Scan(&rem.ID, &rem.RuleID, &rem.Message, &rem.Status,
			&rem.TriggeredAt, &rem.ReadAt); err != nil {
			return nil, err
		}
		reminders = append(reminders, rem)
	}
	return reminders, rows.Err()
}

// MarkReminderRead sets a reminder status to 'read' with current timestamp.
func (s *Store) MarkReminderRead(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE reminders SET status = 'read', read_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("reminder %q: %w", id, ErrNotFound)
	}
	return nil
}

// DismissReminder sets a reminder status to 'dismissed'.
func (s *Store) DismissReminder(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE reminders SET status = 'dismissed' WHERE id = ?`, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("reminder %q: %w", id, ErrNotFound)
	}
	return nil
}

// LastMessageTimeForEntity returns the most recent message timestamp from an entity.
func (s *Store) LastMessageTimeForEntity(ctx context.Context, entityID string) (string, error) {
	var ts sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT MAX(timestamp) FROM messages
		WHERE sender_name = (SELECT name FROM entities WHERE id = ?)`, entityID).
		Scan(&ts)
	if err != nil {
		return "", err
	}
	if !ts.Valid {
		return "", nil
	}
	return ts.String, nil
}

// GetActionItemDueDate returns the due_date of an action item by ID.
func (s *Store) GetActionItemDueDate(ctx context.Context, id string) (string, error) {
	var dueDate sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT due_date FROM action_items WHERE id = ?`, id).
		Scan(&dueDate)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("action item %q: %w", id, ErrNotFound)
	}
	if err != nil {
		return "", err
	}
	if !dueDate.Valid {
		return "", nil
	}
	return dueDate.String, nil
}
