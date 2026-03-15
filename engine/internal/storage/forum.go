package storage

import (
	"context"
	"database/sql"
	"fmt"
)

// CreateSession inserts a new forum debate session.
func (s *Store) CreateSession(ctx context.Context, session ForumSession) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO forum_sessions (id, conversation_id, task_id, status, created_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)`,
		session.ID, session.ConversationID, session.TaskID, session.Status)
	return err
}

// UpdateSession updates the status, task_id, summary, and completed_at of a session.
func (s *Store) UpdateSession(ctx context.Context, id, status, taskID, summary string) error {
	if status == "completed" || status == "failed" {
		_, err := s.db.ExecContext(ctx, `
			UPDATE forum_sessions
			SET status = ?, task_id = ?, summary = ?, completed_at = CURRENT_TIMESTAMP
			WHERE id = ?`, status, taskID, summary, id)
		return err
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE forum_sessions SET status = ?, task_id = ?, summary = ? WHERE id = ?`,
		status, taskID, summary, id)
	return err
}

// GetSession returns a single forum session by ID.
func (s *Store) GetSession(ctx context.Context, id string) (ForumSession, error) {
	var sess ForumSession
	err := s.db.QueryRowContext(ctx, `
		SELECT id, conversation_id, task_id, status, COALESCE(summary,''),
		       created_at, COALESCE(completed_at,'')
		FROM forum_sessions WHERE id = ?`, id).
		Scan(&sess.ID, &sess.ConversationID, &sess.TaskID, &sess.Status,
			&sess.Summary, &sess.CreatedAt, &sess.CompletedAt)
	if err == sql.ErrNoRows {
		return sess, fmt.Errorf("session %q not found", id)
	}
	return sess, err
}

// ListSessions returns all forum sessions ordered by creation time.
func (s *Store) ListSessions(ctx context.Context) ([]ForumSession, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, conversation_id, task_id, status, COALESCE(summary,''),
		       created_at, COALESCE(completed_at,'')
		FROM forum_sessions ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []ForumSession
	for rows.Next() {
		var sess ForumSession
		if err := rows.Scan(&sess.ID, &sess.ConversationID, &sess.TaskID, &sess.Status,
			&sess.Summary, &sess.CreatedAt, &sess.CompletedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, sess)
	}
	return sessions, rows.Err()
}

// SaveForumMessage inserts a single forum message.
func (s *Store) SaveForumMessage(ctx context.Context, msg ForumMessageRecord) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO forum_messages (id, session_id, agent_name, round, stance, content, evidence, confidence)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		msg.ID, msg.SessionID, msg.AgentName, msg.Round, msg.Stance,
		msg.Content, msg.Evidence, msg.Confidence)
	return err
}

// SaveForumMessages inserts multiple forum messages in a transaction.
func (s *Store) SaveForumMessages(ctx context.Context, msgs []ForumMessageRecord) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO forum_messages (id, session_id, agent_name, round, stance, content, evidence, confidence)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, msg := range msgs {
		if _, err := stmt.ExecContext(ctx, msg.ID, msg.SessionID, msg.AgentName,
			msg.Round, msg.Stance, msg.Content, msg.Evidence, msg.Confidence); err != nil {
			return fmt.Errorf("saving forum message %s: %w", msg.ID, err)
		}
	}
	return tx.Commit()
}

// GetForumMessages returns all messages for a debate session ordered by round and time.
func (s *Store) GetForumMessages(ctx context.Context, sessionID string) ([]ForumMessageRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, session_id, agent_name, round, stance, content,
		       COALESCE(evidence,''), confidence, created_at
		FROM forum_messages WHERE session_id = ?
		ORDER BY round ASC, created_at ASC`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []ForumMessageRecord
	for rows.Next() {
		var m ForumMessageRecord
		if err := rows.Scan(&m.ID, &m.SessionID, &m.AgentName, &m.Round, &m.Stance,
			&m.Content, &m.Evidence, &m.Confidence, &m.CreatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}
