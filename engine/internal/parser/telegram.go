package parser

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/welife-os/welife-os/engine/internal/chatir"
)

// TelegramParser parses Telegram Desktop JSON exports (result.json).
type TelegramParser struct{}

func NewTelegramParser() *TelegramParser { return &TelegramParser{} }

func (p *TelegramParser) Format() Format { return FormatTelegramJSON }

func (p *TelegramParser) Detect(r io.ReadSeeker) bool {
	scanner := bufio.NewScanner(io.LimitReader(r, 4096))
	var buf strings.Builder
	for scanner.Scan() {
		buf.WriteString(scanner.Text())
		if buf.Len() > 2048 {
			break
		}
	}
	text := buf.String()
	return strings.Contains(text, `"messages"`) &&
		(strings.Contains(text, `"date_unixtime"`) || strings.Contains(text, `"from_id"`) || strings.Contains(text, `"type":`))
}

// telegramExport represents the top-level Telegram export JSON.
type telegramExport struct {
	Name     string            `json:"name"`
	Type     string            `json:"type"`
	ID       int64             `json:"id"`
	Messages []telegramMessage `json:"messages"`
}

// telegramMessage represents a single Telegram message.
type telegramMessage struct {
	ID              int             `json:"id"`
	Type            string          `json:"type"`
	Date            string          `json:"date"`
	DateUnix        string          `json:"date_unixtime"`
	From            string          `json:"from"`
	FromID          string          `json:"from_id"`
	Text            json.RawMessage `json:"text"`
	ReplyTo         int             `json:"reply_to_message_id"`
	Photo           string          `json:"photo"`
	File            string          `json:"file"`
	Actor           string          `json:"actor"`
	ActorID         string          `json:"actor_id"`
	Action          string          `json:"action"`
}

func (p *TelegramParser) Parse(ctx context.Context, r io.Reader, opts Options) (*chatir.ChatIR, error) {
	var export telegramExport
	if err := json.NewDecoder(r).Decode(&export); err != nil {
		return nil, fmt.Errorf("decoding telegram JSON: %w", err)
	}

	tz := opts.TimeZone
	if tz == nil {
		tz = time.Local
	}

	platform := "telegram"
	if opts.Platform != "" {
		platform = opts.Platform
	}

	participants := make(map[string]*chatir.Participant)
	var messages []chatir.Message

	for _, tm := range export.Messages {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		// Skip service messages
		if tm.Type == "service" {
			continue
		}

		timestamp, err := time.Parse("2006-01-02T15:04:05", tm.Date)
		if err != nil {
			continue
		}
		timestamp = timestamp.In(tz)

		senderID := tm.FromID
		senderName := tm.From
		if senderID == "" {
			senderID = senderName
		}

		// Track participants
		isSelf := isSelfParticipant(senderID, senderName, opts)
		if _, ok := participants[senderID]; !ok {
			participants[senderID] = &chatir.Participant{
				ID:     senderID,
				Name:   senderName,
				IsSelf: isSelf,
			}
		}

		// Parse text field (can be string or array of entities)
		content := extractTelegramText(tm.Text)

		msgType := chatir.MessageText
		var attachments []chatir.Attachment

		if tm.Photo != "" {
			msgType = chatir.MessageImage
			attachments = append(attachments, chatir.Attachment{Type: "image", Path: tm.Photo})
		} else if tm.File != "" {
			msgType = chatir.MessageFile
			attachments = append(attachments, chatir.Attachment{Type: "file", Path: tm.File})
		}

		replyTo := ""
		if tm.ReplyTo > 0 {
			replyTo = fmt.Sprintf("tg_%d", tm.ReplyTo)
		}

		messages = append(messages, chatir.Message{
			ID:          fmt.Sprintf("tg_%d", tm.ID),
			Timestamp:   timestamp,
			SenderID:    senderID,
			Content:     content,
			Type:        msgType,
			ReplyTo:     replyTo,
			Attachments: attachments,
		})
	}

	if len(messages) == 0 {
		return nil, fmt.Errorf("no messages found in Telegram export")
	}

	parts := make([]chatir.Participant, 0, len(participants))
	for _, p := range participants {
		parts = append(parts, *p)
	}

	convType := chatir.ConversationPrivate
	if strings.Contains(export.Type, "group") || strings.Contains(export.Type, "supergroup") {
		convType = chatir.ConversationGroup
	} else if strings.Contains(export.Type, "channel") {
		convType = chatir.ConversationChannel
	}

	ir := &chatir.ChatIR{
		Platform:         platform,
		ConversationID:   fmt.Sprintf("tg_%d", export.ID),
		ConversationType: convType,
		Participants:     parts,
		Messages:         messages,
		Metadata: chatir.Metadata{
			ExportedAt: time.Now().UTC(),
			DateRange: [2]string{
				messages[0].Timestamp.Format(time.RFC3339),
				messages[len(messages)-1].Timestamp.Format(time.RFC3339),
			},
		},
	}
	ir.Normalize()
	if err := ir.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	return ir, nil
}

// extractTelegramText handles the polymorphic text field.
// It can be a plain string or an array of {type, text} objects.
func extractTelegramText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	// Try string first
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}

	// Try array of entities
	var entities []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &entities); err == nil {
		var b strings.Builder
		for _, e := range entities {
			b.WriteString(e.Text)
		}
		return b.String()
	}

	return string(raw)
}

// isSelfParticipant checks if a sender matches the configured self identity.
func isSelfParticipant(senderID, senderName string, opts Options) bool {
	if opts.SelfName != "" && senderName == opts.SelfName {
		return true
	}
	for _, id := range opts.SelfIDs {
		if senderID == id {
			return true
		}
	}
	return false
}
