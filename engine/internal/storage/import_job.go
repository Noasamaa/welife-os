package storage

import (
	"context"
	"database/sql"
	"fmt"
)

// CreateImportJob inserts a new import job record.
func (s *Store) CreateImportJob(ctx context.Context, job ImportJob) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO import_jobs (id, task_id, file_name, format, status)
		VALUES (?, ?, ?, ?, ?)`,
		job.ID, job.TaskID, job.FileName, job.Format, job.Status)
	return err
}

// UpdateImportJob updates mutable fields of an import job.
func (s *Store) UpdateImportJob(ctx context.Context, id string, status string, convID string, msgCount int, errMsg string) error {
	if status == "succeeded" || status == "failed" {
		_, err := s.db.ExecContext(ctx, `
			UPDATE import_jobs SET status = ?, conversation_id = ?, message_count = ?,
			       error_message = ?, completed_at = CURRENT_TIMESTAMP
			WHERE id = ?`,
			status, convID, msgCount, errMsg, id)
		return err
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE import_jobs SET status = ?, conversation_id = ?, message_count = ?, error_message = ?
		WHERE id = ?`,
		status, convID, msgCount, errMsg, id)
	return err
}

// GetImportJob returns a single import job by ID.
func (s *Store) GetImportJob(ctx context.Context, id string) (ImportJob, error) {
	var j ImportJob
	err := s.db.QueryRowContext(ctx, `
		SELECT id, task_id, file_name, format, status,
		       COALESCE(conversation_id,''), message_count,
		       COALESCE(error_message,''), started_at, COALESCE(completed_at,'')
		FROM import_jobs WHERE id = ?`, id).
		Scan(&j.ID, &j.TaskID, &j.FileName, &j.Format, &j.Status,
			&j.ConversationID, &j.MessageCount, &j.ErrorMessage, &j.StartedAt, &j.CompletedAt)
	if err == sql.ErrNoRows {
		return j, fmt.Errorf("import job %q not found", id)
	}
	return j, err
}

// ListImportJobs returns all import jobs ordered by start time.
func (s *Store) ListImportJobs(ctx context.Context) ([]ImportJob, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, task_id, file_name, format, status,
		       COALESCE(conversation_id,''), message_count,
		       COALESCE(error_message,''), started_at, COALESCE(completed_at,'')
		FROM import_jobs ORDER BY started_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []ImportJob
	for rows.Next() {
		var j ImportJob
		if err := rows.Scan(&j.ID, &j.TaskID, &j.FileName, &j.Format, &j.Status,
			&j.ConversationID, &j.MessageCount, &j.ErrorMessage, &j.StartedAt, &j.CompletedAt); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}
