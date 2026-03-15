package storage

import (
	"context"
	"database/sql"
	"fmt"
)

const currentSchemaVersion = 8

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

// migrateV4toV5 adds Phase 4 tables: action items, reminders, person profiles, simulation.
var migrateV4toV5 = []string{
	`CREATE TABLE IF NOT EXISTS action_items (
    id TEXT PRIMARY KEY,
    source_agent TEXT NOT NULL,
    source_session_id TEXT,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    priority TEXT NOT NULL DEFAULT 'medium',
    status TEXT NOT NULL DEFAULT 'pending',
    category TEXT NOT NULL DEFAULT 'general',
    related_entity_id TEXT,
    due_date TEXT,
    completed_at TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);`,
	`CREATE INDEX IF NOT EXISTS idx_action_items_status ON action_items(status);`,
	`CREATE INDEX IF NOT EXISTS idx_action_items_category ON action_items(category);`,
	`CREATE INDEX IF NOT EXISTS idx_action_items_priority ON action_items(priority);`,
	`CREATE TABLE IF NOT EXISTS reminder_rules (
    id TEXT PRIMARY KEY,
    action_item_id TEXT,
    rule_type TEXT NOT NULL,
    entity_id TEXT,
    threshold_days INTEGER,
    cron_expr TEXT,
    message_template TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1,
    last_triggered_at TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);`,
	`CREATE TABLE IF NOT EXISTS reminders (
    id TEXT PRIMARY KEY,
    rule_id TEXT NOT NULL,
    message TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    triggered_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    read_at TEXT,
    FOREIGN KEY (rule_id) REFERENCES reminder_rules(id)
);`,
	`CREATE INDEX IF NOT EXISTS idx_reminders_status ON reminders(status);`,
	`CREATE TABLE IF NOT EXISTS person_profiles (
    id TEXT PRIMARY KEY,
    entity_id TEXT NOT NULL,
    name TEXT NOT NULL,
    personality TEXT NOT NULL DEFAULT '{}',
    relationship_to_self TEXT NOT NULL DEFAULT '{}',
    behavioral_patterns TEXT,
    source_conversation_ids TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (entity_id) REFERENCES entities(id)
);`,
	`CREATE INDEX IF NOT EXISTS idx_person_profiles_entity ON person_profiles(entity_id);`,
	`CREATE TABLE IF NOT EXISTS simulation_sessions (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    fork_description TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'running',
    step_count INTEGER NOT NULL DEFAULT 0,
    original_graph_snapshot TEXT,
    final_graph_snapshot TEXT,
    narrative TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TEXT
);`,
	`CREATE INDEX IF NOT EXISTS idx_simulation_sessions_status ON simulation_sessions(status);`,
	`CREATE TABLE IF NOT EXISTS simulation_steps (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    step_number INTEGER NOT NULL,
    description TEXT NOT NULL,
    entity_changes TEXT NOT NULL DEFAULT '{}',
    reactions TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (session_id) REFERENCES simulation_sessions(id)
);`,
	`CREATE INDEX IF NOT EXISTS idx_simulation_steps_session ON simulation_steps(session_id);`,
	`UPDATE schema_state SET version = 5, updated_at = CURRENT_TIMESTAMP WHERE id = 1;`,
}

// migrateV5toV6 adds the vec_messages virtual table for semantic vector search.
var migrateV5toV6 = []string{
	`CREATE VIRTUAL TABLE IF NOT EXISTS vec_messages USING vec0(
    embedding float[768],
    +source_id text
);`,
	`UPDATE schema_state SET version = 6, updated_at = CURRENT_TIMESTAMP WHERE id = 1;`,
}

// migrateV6toV7 adds the system_settings key-value table for runtime configuration.
var migrateV6toV7 = []string{
	`CREATE TABLE IF NOT EXISTS system_settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);`,
	`UPDATE schema_state SET version = 7, updated_at = CURRENT_TIMESTAMP WHERE id = 1;`,
}

// migrateV7toV8 scopes simulation sessions and profile queries by conversation.
var migrateV7toV8 = []string{
	`ALTER TABLE simulation_sessions ADD COLUMN conversation_id TEXT NOT NULL DEFAULT '';`,
	`CREATE INDEX IF NOT EXISTS idx_simulation_sessions_conversation ON simulation_sessions(conversation_id);`,
	`CREATE INDEX IF NOT EXISTS idx_person_profiles_source_conversation ON person_profiles(source_conversation_ids);`,
	`UPDATE schema_state SET version = 8, updated_at = CURRENT_TIMESTAMP WHERE id = 1;`,
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
		version = 4
	}

	if version == 4 {
		for _, stmt := range migrateV4toV5 {
			if _, err := db.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("migrating v4 to v5: %w", err)
			}
		}
		version = 5
	}

	if version == 5 {
		for _, stmt := range migrateV5toV6 {
			if _, err := db.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("migrating v5 to v6: %w", err)
			}
		}
		version = 6
	}

	if version == 6 {
		for _, stmt := range migrateV6toV7 {
			if _, err := db.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("migrating v6 to v7: %w", err)
			}
		}
		version = 7
	}

	if version == 7 {
		for _, stmt := range migrateV7toV8 {
			if _, err := db.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("migrating v7 to v8: %w", err)
			}
		}
		version = 8
	}

	return nil
}
