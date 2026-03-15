package parser

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/welife-os/welife-os/engine/internal/chatir"
)

// qqTimestampRe matches QQ message timestamp lines: YYYY-MM-DD HH:MM:SS sender
var qqTimestampRe = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}\s+\d{1,2}:\d{2}:\d{2})\s+(.+)$`)

const qqSeparator = "================================================"

// QQParser parses QQ chat export TXT files.
type QQParser struct{}

func NewQQParser() *QQParser { return &QQParser{} }

func (p *QQParser) Format() Format { return FormatQQExport }

func (p *QQParser) Detect(r io.ReadSeeker) bool {
	scanner := bufio.NewScanner(io.LimitReader(r, 4096))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "消息分组") ||
			strings.Contains(line, "消息对象") ||
			line == qqSeparator {
			return true
		}
	}
	return false
}

func (p *QQParser) Parse(ctx context.Context, r io.Reader, opts Options) (*chatir.ChatIR, error) {
	tz := opts.TimeZone
	if tz == nil {
		tz = time.Local
	}

	platform := "qq"
	if opts.Platform != "" {
		platform = opts.Platform
	}

	participants := make(map[string]*chatir.Participant)
	var messages []chatir.Message
	msgSeq := 0

	var currentSenderID string
	var currentTimestamp time.Time
	var contentLines []string
	isGroup := false
	conversationName := ""

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		line := scanner.Text()

		// Check for group header
		if strings.HasPrefix(line, "消息分组:") {
			groupName := strings.TrimPrefix(line, "消息分组:")
			if strings.Contains(groupName, "群") {
				isGroup = true
			}
			continue
		}

		// Check for conversation target
		if strings.HasPrefix(line, "消息对象:") {
			conversationName = strings.TrimPrefix(line, "消息对象:")
			continue
		}

		// Skip separator lines
		if line == qqSeparator {
			continue
		}

		// Check for timestamp line
		matches := qqTimestampRe.FindStringSubmatch(line)
		if matches != nil {
			// Flush previous message
			if currentSenderID != "" && len(contentLines) > 0 {
				msgSeq++
				flushed := strings.TrimRight(strings.Join(contentLines, "\n"), "\n")
				messages = append(messages, chatir.Message{
					ID:        fmt.Sprintf("qq_%d", msgSeq),
					Timestamp: currentTimestamp,
					SenderID:  currentSenderID,
					Content:   flushed,
					Type:      chatir.MessageText,
				})
			}

			ts, err := time.ParseInLocation("2006-01-02 15:04:05", matches[1], tz)
			if err != nil {
				currentSenderID = ""
				contentLines = nil
				continue
			}
			currentTimestamp = ts
			senderName := strings.TrimSpace(matches[2])

			senderID := strings.ReplaceAll(strings.ToLower(senderName), " ", "_")
			isSelf := senderName == "我" || isSelfParticipant(senderID, senderName, opts)

			if _, ok := participants[senderID]; !ok {
				participants[senderID] = &chatir.Participant{
					ID:     senderID,
					Name:   senderName,
					IsSelf: isSelf,
				}
			}

			currentSenderID = senderID
			contentLines = nil
			continue
		}

		// Content line (may be empty)
		if currentSenderID != "" {
			if line != "" || len(contentLines) > 0 {
				contentLines = append(contentLines, line)
			}
		}
	}

	// Flush last message
	if currentSenderID != "" && len(contentLines) > 0 {
		msgSeq++
		flushed := strings.TrimRight(strings.Join(contentLines, "\n"), "\n")
		messages = append(messages, chatir.Message{
			ID:        fmt.Sprintf("qq_%d", msgSeq),
			Timestamp: currentTimestamp,
			SenderID:  currentSenderID,
			Content:   flushed,
			Type:      chatir.MessageText,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading QQ export: %w", err)
	}

	if len(messages) == 0 {
		return nil, fmt.Errorf("no messages found in QQ export")
	}

	parts := make([]chatir.Participant, 0, len(participants))
	for _, pt := range participants {
		parts = append(parts, *pt)
	}

	convType := chatir.ConversationPrivate
	if isGroup {
		convType = chatir.ConversationGroup
	}

	convID := conversationName
	if convID == "" {
		convID = messages[0].Timestamp.Format("20060102")
	}

	ir := &chatir.ChatIR{
		Platform:         platform,
		ConversationID:   fmt.Sprintf("qq_%s", convID),
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
