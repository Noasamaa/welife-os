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

// ActionItem represents an actionable task extracted by the ExecutionCoach.
type ActionItem struct {
	ID              string `json:"id"`
	SourceAgent     string `json:"source_agent"`
	SourceSessionID string `json:"source_session_id,omitempty"`
	Title           string `json:"title"`
	Description     string `json:"description"`
	Priority        string `json:"priority"`
	Status          string `json:"status"`
	Category        string `json:"category"`
	RelatedEntityID string `json:"related_entity_id,omitempty"`
	DueDate         string `json:"due_date,omitempty"`
	CompletedAt     string `json:"completed_at,omitempty"`
	CreatedAt       string `json:"created_at"`
}

// ReminderRule defines a rule for generating reminders.
type ReminderRule struct {
	ID              string `json:"id"`
	ActionItemID    string `json:"action_item_id,omitempty"`
	RuleType        string `json:"rule_type"`
	EntityID        string `json:"entity_id,omitempty"`
	ThresholdDays   int    `json:"threshold_days,omitempty"`
	CronExpr        string `json:"cron_expr,omitempty"`
	MessageTemplate string `json:"message_template"`
	Enabled         bool   `json:"enabled"`
	LastTriggeredAt string `json:"last_triggered_at,omitempty"`
	CreatedAt       string `json:"created_at"`
}

// Reminder represents a fired reminder instance.
type Reminder struct {
	ID          string `json:"id"`
	RuleID      string `json:"rule_id"`
	Message     string `json:"message"`
	Status      string `json:"status"`
	TriggeredAt string `json:"triggered_at"`
	ReadAt      string `json:"read_at,omitempty"`
}

// PersonProfile is a digital avatar generated from graph entity data.
type PersonProfile struct {
	ID                    string `json:"id"`
	EntityID              string `json:"entity_id"`
	Name                  string `json:"name"`
	Personality           string `json:"personality"`
	RelationshipToSelf    string `json:"relationship_to_self"`
	BehavioralPatterns    string `json:"behavioral_patterns,omitempty"`
	SourceConversationIDs string `json:"source_conversation_ids,omitempty"`
	CreatedAt             string `json:"created_at"`
	UpdatedAt             string `json:"updated_at"`
}

// SimulationSession represents a parallel life simulation run.
type SimulationSession struct {
	ID                    string `json:"id"`
	ConversationID        string `json:"conversation_id"`
	TaskID                string `json:"task_id"`
	ForkDescription       string `json:"fork_description"`
	Status                string `json:"status"`
	StepCount             int    `json:"step_count"`
	OriginalGraphSnapshot string `json:"original_graph_snapshot,omitempty"`
	FinalGraphSnapshot    string `json:"final_graph_snapshot,omitempty"`
	Narrative             string `json:"narrative,omitempty"`
	CreatedAt             string `json:"created_at"`
	CompletedAt           string `json:"completed_at,omitempty"`
}

// SimulationStep represents one evolution step in a simulation.
type SimulationStep struct {
	ID            string `json:"id"`
	SessionID     string `json:"session_id"`
	StepNumber    int    `json:"step_number"`
	Description   string `json:"description"`
	EntityChanges string `json:"entity_changes"`
	Reactions     string `json:"reactions"`
	CreatedAt     string `json:"created_at"`
}
