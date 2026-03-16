package storage

import (
	"context"
	"database/sql"
	"fmt"
)

// CreateReport inserts a new report record.
func (s *Store) CreateReport(ctx context.Context, r Report) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO reports (id, type, conversation_id, task_id, status, title, content, period_start, period_end)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.Type, r.ConversationID, r.TaskID, r.Status, r.Title, r.Content, r.PeriodStart, r.PeriodEnd)
	return err
}

// BindReportTask stores the concrete task ID once the async report job is queued.
func (s *Store) BindReportTask(ctx context.Context, id string, taskID string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE reports SET task_id = ? WHERE id = ?`,
		taskID, id)
	return err
}

// UpdateReport updates the status, title, content, and completed_at of a report.
func (s *Store) UpdateReport(ctx context.Context, id, status, title, content string) error {
	if status == "completed" || status == "failed" {
		_, err := s.db.ExecContext(ctx, `
			UPDATE reports
			SET status = ?, title = ?, content = ?, completed_at = CURRENT_TIMESTAMP
			WHERE id = ?`, status, title, content, id)
		return err
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE reports SET status = ?, title = ?, content = ? WHERE id = ?`,
		status, title, content, id)
	return err
}

// GetReport returns a single report by ID.
func (s *Store) GetReport(ctx context.Context, id string) (Report, error) {
	var r Report
	err := s.db.QueryRowContext(ctx, `
		SELECT id, type, conversation_id, task_id, status, title, content,
		       period_start, period_end, created_at, COALESCE(completed_at,'')
		FROM reports WHERE id = ?`, id).
		Scan(&r.ID, &r.Type, &r.ConversationID, &r.TaskID, &r.Status,
			&r.Title, &r.Content, &r.PeriodStart, &r.PeriodEnd,
			&r.CreatedAt, &r.CompletedAt)
	if err == sql.ErrNoRows {
		return r, fmt.Errorf("report %q: %w", id, ErrNotFound)
	}
	return r, err
}

// ListReports returns all reports ordered by creation time.
// Note: content is omitted from list queries for performance; use GetReport for full content.
func (s *Store) ListReports(ctx context.Context) ([]Report, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, type, conversation_id, task_id, status, title,
		       period_start, period_end, created_at, COALESCE(completed_at,'')
		FROM reports ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []Report
	for rows.Next() {
		var r Report
		if err := rows.Scan(&r.ID, &r.Type, &r.ConversationID, &r.TaskID, &r.Status,
			&r.Title, &r.PeriodStart, &r.PeriodEnd, &r.CreatedAt, &r.CompletedAt); err != nil {
			return nil, err
		}
		reports = append(reports, r)
	}
	return reports, rows.Err()
}

// DeleteReport removes a report by ID.
func (s *Store) DeleteReport(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM reports WHERE id = ?`, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("report %q: %w", id, ErrNotFound)
	}
	return nil
}
