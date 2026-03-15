package chatir

import "time"

type ChatIR struct {
	Platform         string           `json:"platform"`
	ConversationID   string           `json:"conversation_id"`
	ConversationType ConversationType `json:"conversation_type"`
	Participants     []Participant    `json:"participants"`
	Messages         []Message        `json:"messages"`
	Metadata         Metadata         `json:"metadata"`
}

type ConversationType string

const (
	ConversationPrivate ConversationType = "private"
	ConversationGroup   ConversationType = "group"
	ConversationChannel ConversationType = "channel"
)

type Participant struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	IsSelf bool   `json:"is_self"`
}

type MessageType string

const (
	MessageText   MessageType = "text"
	MessageImage  MessageType = "image"
	MessageFile   MessageType = "file"
	MessageAudio  MessageType = "audio"
	MessageVideo  MessageType = "video"
	MessageSystem MessageType = "system"
)

type Attachment struct {
	Type     string `json:"type"`
	Name     string `json:"name,omitempty"`
	Path     string `json:"path,omitempty"`
	MIMEType string `json:"mime_type,omitempty"`
}

type Message struct {
	ID          string       `json:"id"`
	Timestamp   time.Time    `json:"timestamp"`
	SenderID    string       `json:"sender_id"`
	Content     string       `json:"content"`
	Type        MessageType  `json:"type"`
	ReplyTo     string       `json:"reply_to,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

type Metadata struct {
	ExportedAt   time.Time `json:"exported_at"`
	MessageCount int       `json:"message_count"`
	DateRange    [2]string `json:"date_range,omitempty"`
}
