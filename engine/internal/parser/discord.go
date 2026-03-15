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

// DiscordParser parses DiscordChatExporter JSON exports.
type DiscordParser struct{}

func NewDiscordParser() *DiscordParser { return &DiscordParser{} }

func (p *DiscordParser) Format() Format { return FormatDiscordJSON }

func (p *DiscordParser) Detect(r io.ReadSeeker) bool {
	scanner := bufio.NewScanner(io.LimitReader(r, 4096))
	var buf strings.Builder
	for scanner.Scan() {
		buf.WriteString(scanner.Text())
		if buf.Len() > 2048 {
			break
		}
	}
	text := buf.String()
	hasMessages := strings.Contains(text, `"messages"`)
	hasGuildOrChannel := strings.Contains(text, `"guild"`) || strings.Contains(text, `"channel"`)
	return hasMessages && hasGuildOrChannel
}

type discordExport struct {
	Guild   discordGuild     `json:"guild"`
	Channel discordChannel   `json:"channel"`
	Messages []discordMessage `json:"messages"`
}

type discordGuild struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type discordChannel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type discordMessage struct {
	ID          string              `json:"id"`
	Timestamp   string              `json:"timestamp"`
	Content     string              `json:"content"`
	Author      discordAuthor       `json:"author"`
	Attachments []discordAttachment `json:"attachments"`
	Embeds      []discordEmbed      `json:"embeds"`
	Type        string              `json:"type"`
}

type discordAuthor struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Nickname string `json:"nickname"`
}

type discordAttachment struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	FileName string `json:"fileName"`
}

type discordEmbed struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url"`
}

func (p *DiscordParser) Parse(ctx context.Context, r io.Reader, opts Options) (*chatir.ChatIR, error) {
	var export discordExport
	if err := json.NewDecoder(r).Decode(&export); err != nil {
		return nil, fmt.Errorf("decoding discord JSON: %w", err)
	}

	tz := opts.TimeZone
	if tz == nil {
		tz = time.Local
	}

	platform := "discord"
	if opts.Platform != "" {
		platform = opts.Platform
	}

	participants := make(map[string]*chatir.Participant)
	var messages []chatir.Message

	for _, dm := range export.Messages {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		timestamp, err := time.Parse(time.RFC3339, dm.Timestamp)
		if err != nil {
			// Try without timezone
			timestamp, err = time.Parse("2006-01-02T15:04:05+00:00", dm.Timestamp)
			if err != nil {
				continue
			}
		}
		timestamp = timestamp.In(tz)

		senderID := dm.Author.ID
		senderName := dm.Author.Nickname
		if senderName == "" {
			senderName = dm.Author.Name
		}

		isSelf := isSelfParticipant(senderID, senderName, opts)
		if _, ok := participants[senderID]; !ok {
			participants[senderID] = &chatir.Participant{
				ID:     senderID,
				Name:   senderName,
				IsSelf: isSelf,
			}
		}

		msgType := chatir.MessageText
		var attachments []chatir.Attachment

		for _, att := range dm.Attachments {
			attachments = append(attachments, chatir.Attachment{
				Type: "file",
				Name: att.FileName,
				Path: att.URL,
			})
		}

		if len(attachments) > 0 {
			msgType = chatir.MessageFile
		}

		content := dm.Content
		for _, embed := range dm.Embeds {
			if embed.Title != "" {
				content += "\n[Embed: " + embed.Title + "]"
			}
		}

		messages = append(messages, chatir.Message{
			ID:          fmt.Sprintf("dc_%s", dm.ID),
			Timestamp:   timestamp,
			SenderID:    senderID,
			Content:     content,
			Type:        msgType,
			Attachments: attachments,
		})
	}

	if len(messages) == 0 {
		return nil, fmt.Errorf("no messages found in Discord export")
	}

	parts := make([]chatir.Participant, 0, len(participants))
	for _, pt := range participants {
		parts = append(parts, *pt)
	}

	convType := chatir.ConversationGroup
	if export.Guild.ID == "" {
		convType = chatir.ConversationPrivate
	}

	convID := export.Channel.ID
	if convID == "" {
		convID = fmt.Sprintf("dc_%s", messages[0].Timestamp.Format("20060102"))
	}

	ir := &chatir.ChatIR{
		Platform:         platform,
		ConversationID:   fmt.Sprintf("dc_%s", convID),
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
