package storage

import (
	"context"
	"database/sql"
	"fmt"
)

const currentSchemaVersion = 2

// migrateV1toV2 adds Phase 1 tables for conversations, messages, participants,
// attachments, import jobs, entities, and relationships.
var migrateV1toV2 = []string{
	`CREATE TABLE IF NOT EXISTS conversations (
    id TEXT PRIMARY KEY,
    platform TEXT NOT NULL,
    conversation_type TEXT NOT NULL DEFAULT 'private',
    title TEXT,
    message_count INTEGER NOT NULL DEFAULT 0,
    first_message_at TEXT,
    last_message_at TEXT,
    imported_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);`,
	`CREATE TABLE IF NOT EXISTS messages (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL,
    platform TEXT NOT NULL,
    sender_id TEXT NOT NULL,
    sender_name TEXT NOT NULL,
    content TEXT NOT NULL,
    message_type TEXT NOT NULL DEFAULT 'text',
    reply_to TEXT,
    timestamp TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (conversation_id) REFERENCES conversations(id)
);`,
	`CREATE INDEX IF NOT EXISTS idx_messages_conversation ON messages(conversation_id);`,
	`CREATE INDEX IF NOT EXISTS idx_messages_timestamp ON messages(timestamp);`,
	`CREATE INDEX IF NOT EXISTS idx_messages_sender ON messages(sender_id);`,
	`CREATE TABLE IF NOT EXISTS participants (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    conversation_id TEXT NOT NULL,
    participant_id TEXT NOT NULL,
    display_name TEXT NOT NULL,
    is_self INTEGER NOT NULL DEFAULT 0,
    UNIQUE(conversation_id, participant_id),
    FOREIGN KEY (conversation_id) REFERENCES conversations(id)
);`,
	`CREATE TABLE IF NOT EXISTS attachments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    message_id TEXT NOT NULL,
    type TEXT NOT NULL,
    name TEXT,
    path TEXT,
    mime_type TEXT,
    FOREIGN KEY (message_id) REFERENCES messages(id)
);`,
	`CREATE TABLE IF NOT EXISTS import_jobs (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    file_name TEXT NOT NULL,
    format TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    conversation_id TEXT,
    message_count INTEGER DEFAULT 0,
    error_message TEXT,
    started_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TEXT
);`,
	`CREATE TABLE IF NOT EXISTS entities (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    name TEXT NOT NULL,
    properties TEXT,
    source_conversation TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);`,
	`CREATE INDEX IF NOT EXISTS idx_entities_type ON entities(type);`,
	`CREATE INDEX IF NOT EXISTS idx_entities_name ON entities(name);`,
	`CREATE TABLE IF NOT EXISTS relationships (
    id TEXT PRIMARY KEY,
    source_entity_id TEXT NOT NULL,
    target_entity_id TEXT NOT NULL,
    type TEXT NOT NULL,
    properties TEXT,
    weight REAL NOT NULL DEFAULT 1.0,
    source_message_id TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (source_entity_id) REFERENCES entities(id),
    FOREIGN KEY (target_entity_id) REFERENCES entities(id)
);`,
	`CREATE INDEX IF NOT EXISTS idx_relationships_source ON relationships(source_entity_id);`,
	`CREATE INDEX IF NOT EXISTS idx_relationships_target ON relationships(target_entity_id);`,
	`CREATE INDEX IF NOT EXISTS idx_relationships_type ON relationships(type);`,
	`UPDATE schema_state SET version = 2, updated_at = CURRENT_TIMESTAMP WHERE id = 1;`,
}

// migrate checks the current schema version and applies pending migrations.
func migrate(ctx context.Context, db *sql.DB) error {
	var version int
	err := db.QueryRowContext(ctx, "SELECT version FROM schema_state WHERE id = 1").Scan(&version)
	if err != nil {
		return fmt.Errorf("reading schema version: %w", err)
	}

	if version >= currentSchemaVersion {
		return nil
	}

	if version == 1 {
		for _, stmt := range migrateV1toV2 {
			if _, err := db.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("migrating v1 to v2: %w", err)
			}
		}
	}

	return nil
}
