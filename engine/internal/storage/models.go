package storage

import "time"

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

// Conversation represents an imported chat conversation.
type Conversation struct {
	ID               string    `json:"id"`
	Platform         string    `json:"platform"`
	ConversationType string    `json:"conversation_type"`
	Title            string    `json:"title,omitempty"`
	MessageCount     int       `json:"message_count"`
	FirstMessageAt   string    `json:"first_message_at,omitempty"`
	LastMessageAt    string    `json:"last_message_at,omitempty"`
	ImportedAt       time.Time `json:"imported_at"`
}

// StoredMessage is the storage-layer representation of a chat message.
type StoredMessage struct {
	ID             string `json:"id"`
	ConversationID string `json:"conversation_id"`
	Platform       string `json:"platform"`
	SenderID       string `json:"sender_id"`
	SenderName     string `json:"sender_name"`
	Content        string `json:"content"`
	MessageType    string `json:"message_type"`
	ReplyTo        string `json:"reply_to,omitempty"`
	Timestamp      string `json:"timestamp"`
}

// StoredParticipant is the storage-layer representation of a conversation participant.
type StoredParticipant struct {
	ConversationID string `json:"conversation_id"`
	ParticipantID  string `json:"participant_id"`
	DisplayName    string `json:"display_name"`
	IsSelf         bool   `json:"is_self"`
}

// ImportJob tracks the status of a file import operation.
type ImportJob struct {
	ID             string `json:"id"`
	TaskID         string `json:"task_id"`
	FileName       string `json:"file_name"`
	Format         string `json:"format"`
	Status         string `json:"status"`
	ConversationID string `json:"conversation_id,omitempty"`
	MessageCount   int    `json:"message_count"`
	ErrorMessage   string `json:"error_message,omitempty"`
	StartedAt      string `json:"started_at"`
	CompletedAt    string `json:"completed_at,omitempty"`
}

// Entity represents a knowledge graph entity.
type Entity struct {
	ID                 string `json:"id"`
	Type               string `json:"type"`
	Name               string `json:"name"`
	Properties         string `json:"properties,omitempty"`
	SourceConversation string `json:"source_conversation,omitempty"`
}

// Relationship represents a knowledge graph edge.
type Relationship struct {
	ID              string  `json:"id"`
	SourceEntityID  string  `json:"source_entity_id"`
	TargetEntityID  string  `json:"target_entity_id"`
	Type            string  `json:"type"`
	Properties      string  `json:"properties,omitempty"`
	Weight          float64 `json:"weight"`
	SourceMessageID string  `json:"source_message_id,omitempty"`
}

// ForumSession represents a debate session.
type ForumSession struct {
	ID             string `json:"id"`
	ConversationID string `json:"conversation_id"`
	TaskID         string `json:"task_id"`
	Status         string `json:"status"`
	Summary        string `json:"summary,omitempty"`
	CreatedAt      string `json:"created_at"`
	CompletedAt    string `json:"completed_at,omitempty"`
}

// ForumMessageRecord represents a single message in a debate session.
type ForumMessageRecord struct {
	ID         string  `json:"id"`
	SessionID  string  `json:"session_id"`
	AgentName  string  `json:"agent_name"`
	Round      int     `json:"round"`
	Stance     string  `json:"stance"`
	Content    string  `json:"content"`
	Evidence   string  `json:"evidence,omitempty"`
	Confidence float64 `json:"confidence"`
	CreatedAt  string  `json:"created_at"`
}

// Report represents a generated life report.
type Report struct {
	ID             string `json:"id"`
	Type           string `json:"type"`
	ConversationID string `json:"conversation_id"`
	TaskID         string `json:"task_id"`
	Status         string `json:"status"`
	Title          string `json:"title"`
	Content        string `json:"content"`
	PeriodStart    string `json:"period_start"`
	PeriodEnd      string `json:"period_end"`
	CreatedAt      string `json:"created_at"`
	CompletedAt    string `json:"completed_at,omitempty"`
}

// MessageSearchParams defines filters for keyword-based message search.
type MessageSearchParams struct {
	Keyword        string
	ConversationID string
	SenderName     string
	After          string
	Before         string
	Limit          int
}
