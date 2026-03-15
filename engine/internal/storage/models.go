package storage

var schemaStatements = []string{
	`
CREATE TABLE IF NOT EXISTS schema_state (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    version INTEGER NOT NULL,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`,
	`
INSERT INTO schema_state (id, version)
VALUES (1, 1)
ON CONFLICT(id) DO NOTHING;
`,
	`
CREATE TABLE IF NOT EXISTS imported_conversations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    platform TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`,
}
