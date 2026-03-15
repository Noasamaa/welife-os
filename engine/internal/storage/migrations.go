package storage

import (
	"context"
	"database/sql"
	"fmt"
)

const currentSchemaVersion = 4

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

// migrateV2toV3 adds forum_sessions and forum_messages tables for debate persistence.
var migrateV2toV3 = []string{
	`CREATE TABLE IF NOT EXISTS forum_sessions (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL,
    task_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'running',
    summary TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TEXT,
    FOREIGN KEY (conversation_id) REFERENCES conversations(id)
);`,
	`CREATE INDEX IF NOT EXISTS idx_forum_sessions_conversation ON forum_sessions(conversation_id);`,
	`CREATE INDEX IF NOT EXISTS idx_forum_sessions_status ON forum_sessions(status);`,
	`CREATE TABLE IF NOT EXISTS forum_messages (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    agent_name TEXT NOT NULL,
    round INTEGER NOT NULL,
    stance TEXT NOT NULL,
    content TEXT NOT NULL,
    evidence TEXT,
    confidence REAL NOT NULL DEFAULT 0.0,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (session_id) REFERENCES forum_sessions(id)
);`,
	`CREATE INDEX IF NOT EXISTS idx_forum_messages_session ON forum_messages(session_id);`,
	`CREATE INDEX IF NOT EXISTS idx_forum_messages_round ON forum_messages(round);`,
	`UPDATE schema_state SET version = 3, updated_at = CURRENT_TIMESTAMP WHERE id = 1;`,
}

// migrateV3toV4 adds the reports table for generated life reports.
var migrateV3toV4 = []string{
	`CREATE TABLE IF NOT EXISTS reports (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    task_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'running',
    title TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL DEFAULT '{}',
    period_start TEXT NOT NULL,
    period_end TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TEXT,
    FOREIGN KEY (conversation_id) REFERENCES conversations(id)
);`,
	`CREATE INDEX IF NOT EXISTS idx_reports_type ON reports(type);`,
	`CREATE INDEX IF NOT EXISTS idx_reports_status ON reports(status);`,
	`CREATE INDEX IF NOT EXISTS idx_reports_period ON reports(period_start, period_end);`,
	`UPDATE schema_state SET version = 4, updated_at = CURRENT_TIMESTAMP WHERE id = 1;`,
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
		version = 2
	}

	if version == 2 {
		for _, stmt := range migrateV2toV3 {
			if _, err := db.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("migrating v2 to v3: %w", err)
			}
		}
		version = 3
	}

	if version == 3 {
		for _, stmt := range migrateV3toV4 {
			if _, err := db.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("migrating v3 to v4: %w", err)
			}
		}
	}

	return nil
}
