package storage

import (
	"context"
	"database/sql"
	"fmt"
)

// SaveConversation inserts or replaces a conversation record.
func (s *Store) SaveConversation(ctx context.Context, conv Conversation) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO conversations (id, platform, conversation_type, title, message_count, first_message_at, last_message_at, imported_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			message_count = excluded.message_count,
			first_message_at = excluded.first_message_at,
			last_message_at = excluded.last_message_at`,
		conv.ID, conv.Platform, conv.ConversationType, conv.Title,
		conv.MessageCount, conv.FirstMessageAt, conv.LastMessageAt,
	)
	return err
}

// SaveConversationBundle atomically persists a conversation, its messages, and participants.
func (s *Store) SaveConversationBundle(ctx context.Context, conv Conversation, msgs []StoredMessage, parts []StoredParticipant) error {
	for _, m := range msgs {
		if m.ConversationID != conv.ID {
			return fmt.Errorf("message %s does not belong to conversation %s", m.ID, conv.ID)
		}
	}
	for _, p := range parts {
		if p.ConversationID != conv.ID {
			return fmt.Errorf("participant %s does not belong to conversation %s", p.ParticipantID, conv.ID)
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO conversations (id, platform, conversation_type, title, message_count, first_message_at, last_message_at, imported_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			message_count = excluded.message_count,
			first_message_at = excluded.first_message_at,
			last_message_at = excluded.last_message_at`,
		conv.ID, conv.Platform, conv.ConversationType, conv.Title,
		conv.MessageCount, conv.FirstMessageAt, conv.LastMessageAt,
	); err != nil {
		return err
	}

	msgStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO messages (id, conversation_id, platform, sender_id, sender_name, content, message_type, reply_to, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			conversation_id = excluded.conversation_id,
			platform = excluded.platform,
			sender_id = excluded.sender_id,
			sender_name = excluded.sender_name,
			content = excluded.content,
			message_type = excluded.message_type,
			reply_to = excluded.reply_to,
			timestamp = excluded.timestamp`)
	if err != nil {
		return err
	}
	defer msgStmt.Close()

	for _, m := range msgs {
		if _, err := msgStmt.ExecContext(ctx, m.ID, m.ConversationID, m.Platform,
			m.SenderID, m.SenderName, m.Content, m.MessageType, m.ReplyTo, m.Timestamp); err != nil {
			return fmt.Errorf("inserting message %s: %w", m.ID, err)
		}
	}

	partStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO participants (conversation_id, participant_id, display_name, is_self)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(conversation_id, participant_id) DO UPDATE SET display_name = excluded.display_name`)
	if err != nil {
		return err
	}
	defer partStmt.Close()

	for _, p := range parts {
		isSelf := 0
		if p.IsSelf {
			isSelf = 1
		}
		if _, err := partStmt.ExecContext(ctx, p.ConversationID, p.ParticipantID, p.DisplayName, isSelf); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// ListConversations returns all imported conversations ordered by import time.
func (s *Store) ListConversations(ctx context.Context) ([]Conversation, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, platform, conversation_type, COALESCE(title,''), message_count,
		       COALESCE(first_message_at,''), COALESCE(last_message_at,''), imported_at
		FROM conversations ORDER BY imported_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var convs []Conversation
	for rows.Next() {
		var c Conversation
		var importedAt string
		if err := rows.Scan(&c.ID, &c.Platform, &c.ConversationType, &c.Title,
			&c.MessageCount, &c.FirstMessageAt, &c.LastMessageAt, &importedAt); err != nil {
			return nil, err
		}
		convs = append(convs, c)
	}
	return convs, rows.Err()
}

// GetConversation returns a single conversation by ID.
func (s *Store) GetConversation(ctx context.Context, id string) (Conversation, error) {
	var c Conversation
	var importedAt string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, platform, conversation_type, COALESCE(title,''), message_count,
		       COALESCE(first_message_at,''), COALESCE(last_message_at,''), imported_at
		FROM conversations WHERE id = ?`, id).
		Scan(&c.ID, &c.Platform, &c.ConversationType, &c.Title,
			&c.MessageCount, &c.FirstMessageAt, &c.LastMessageAt, &importedAt)
	if err == sql.ErrNoRows {
		return c, fmt.Errorf("conversation %q not found", id)
	}
	return c, err
}

// SaveMessages inserts messages in a single transaction.
func (s *Store) SaveMessages(ctx context.Context, msgs []StoredMessage) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO messages (id, conversation_id, platform, sender_id, sender_name, content, message_type, reply_to, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			conversation_id = excluded.conversation_id,
			platform = excluded.platform,
			sender_id = excluded.sender_id,
			sender_name = excluded.sender_name,
			content = excluded.content,
			message_type = excluded.message_type,
			reply_to = excluded.reply_to,
			timestamp = excluded.timestamp`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, m := range msgs {
		if _, err := stmt.ExecContext(ctx, m.ID, m.ConversationID, m.Platform,
			m.SenderID, m.SenderName, m.Content, m.MessageType, m.ReplyTo, m.Timestamp); err != nil {
			return fmt.Errorf("inserting message %s: %w", m.ID, err)
		}
	}
	return tx.Commit()
}

// GetMessages returns paginated messages for a conversation.
func (s *Store) GetMessages(ctx context.Context, conversationID string, limit, offset int) ([]StoredMessage, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, conversation_id, platform, sender_id, sender_name, content, message_type,
		       COALESCE(reply_to,''), timestamp
		FROM messages WHERE conversation_id = ?
		ORDER BY timestamp ASC LIMIT ? OFFSET ?`,
		conversationID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []StoredMessage
	for rows.Next() {
		var m StoredMessage
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Platform, &m.SenderID,
			&m.SenderName, &m.Content, &m.MessageType, &m.ReplyTo, &m.Timestamp); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

// SaveParticipants inserts participants for a conversation.
func (s *Store) SaveParticipants(ctx context.Context, parts []StoredParticipant) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO participants (conversation_id, participant_id, display_name, is_self)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(conversation_id, participant_id) DO UPDATE SET display_name = excluded.display_name`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, p := range parts {
		isSelf := 0
		if p.IsSelf {
			isSelf = 1
		}
		if _, err := stmt.ExecContext(ctx, p.ConversationID, p.ParticipantID, p.DisplayName, isSelf); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// MessageCount returns the total number of messages for a conversation.
func (s *Store) MessageCount(ctx context.Context, conversationID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM messages WHERE conversation_id = ?`, conversationID).Scan(&count)
	return count, err
}

// SearchMessages performs keyword-based search on messages with optional filters.
func (s *Store) SearchMessages(ctx context.Context, params MessageSearchParams) ([]StoredMessage, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = 100
	}

	query := `SELECT id, conversation_id, platform, sender_id, sender_name, content,
	                 message_type, COALESCE(reply_to,''), timestamp
	          FROM messages WHERE 1=1`
	var args []any

	if params.Keyword != "" {
		query += ` AND content LIKE ?`
		args = append(args, "%"+params.Keyword+"%")
	}
	if params.ConversationID != "" {
		query += ` AND conversation_id = ?`
		args = append(args, params.ConversationID)
	}
	if params.SenderName != "" {
		query += ` AND sender_name = ?`
		args = append(args, params.SenderName)
	}
	if params.After != "" {
		query += ` AND timestamp >= ?`
		args = append(args, params.After)
	}
	if params.Before != "" {
		query += ` AND timestamp <= ?`
		args = append(args, params.Before)
	}

	query += ` ORDER BY timestamp ASC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []StoredMessage
	for rows.Next() {
		var m StoredMessage
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Platform, &m.SenderID,
			&m.SenderName, &m.Content, &m.MessageType, &m.ReplyTo, &m.Timestamp); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}
