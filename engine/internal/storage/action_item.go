package storage

import (
	"context"
	"database/sql"
	"fmt"
)

// CreateActionItem inserts a new action item record.
func (s *Store) CreateActionItem(ctx context.Context, item ActionItem) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO action_items (id, source_agent, source_session_id, title, description,
			priority, status, category, related_entity_id, due_date)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.SourceAgent, item.SourceSessionID, item.Title, item.Description,
		item.Priority, item.Status, item.Category, item.RelatedEntityID, item.DueDate)
	return err
}

// ListActionItems returns action items filtered by optional status and category.
// Pass empty strings to skip filtering.
func (s *Store) ListActionItems(ctx context.Context, status, category string) ([]ActionItem, error) {
	query := `SELECT id, source_agent, COALESCE(source_session_id,''), title, description,
		priority, status, category, COALESCE(related_entity_id,''),
		COALESCE(due_date,''), COALESCE(completed_at,''), created_at
		FROM action_items WHERE 1=1`
	var args []any

	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	if category != "" {
		query += " AND category = ?"
		args = append(args, category)
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []ActionItem
	for rows.Next() {
		var item ActionItem
		if err := rows.Scan(&item.ID, &item.SourceAgent, &item.SourceSessionID,
			&item.Title, &item.Description, &item.Priority, &item.Status,
			&item.Category, &item.RelatedEntityID, &item.DueDate,
			&item.CompletedAt, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// GetActionItem returns a single action item by ID.
func (s *Store) GetActionItem(ctx context.Context, id string) (ActionItem, error) {
	var item ActionItem
	err := s.db.QueryRowContext(ctx, `
		SELECT id, source_agent, COALESCE(source_session_id,''), title, description,
			priority, status, category, COALESCE(related_entity_id,''),
			COALESCE(due_date,''), COALESCE(completed_at,''), created_at
		FROM action_items WHERE id = ?`, id).
		Scan(&item.ID, &item.SourceAgent, &item.SourceSessionID,
			&item.Title, &item.Description, &item.Priority, &item.Status,
			&item.Category, &item.RelatedEntityID, &item.DueDate,
			&item.CompletedAt, &item.CreatedAt)
	if err == sql.ErrNoRows {
		return item, fmt.Errorf("action item %q not found", id)
	}
	return item, err
}

// UpdateActionItemStatus updates the status of an action item.
// When status is "completed", completed_at is set to the current timestamp.
func (s *Store) UpdateActionItemStatus(ctx context.Context, id, status string) error {
	var result sql.Result
	var err error

	if status == "completed" {
		result, err = s.db.ExecContext(ctx, `
			UPDATE action_items SET status = ?, completed_at = CURRENT_TIMESTAMP
			WHERE id = ?`, status, id)
	} else {
		result, err = s.db.ExecContext(ctx, `
			UPDATE action_items SET status = ?, completed_at = NULL
			WHERE id = ?`, status, id)
	}
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("action item %q not found", id)
	}
	return nil
}

// DeleteActionItem removes an action item by ID.
func (s *Store) DeleteActionItem(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM action_items WHERE id = ?`, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("action item %q not found", id)
	}
	return nil
}
