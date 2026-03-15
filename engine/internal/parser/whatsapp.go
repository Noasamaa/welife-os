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

// whatsappLineRe matches WhatsApp export lines: MM/DD/YYYY, HH:MM AM/PM - Sender: Message
var whatsappLineRe = regexp.MustCompile(`^(\d{1,2}/\d{1,2}/\d{2,4}),\s+(\d{1,2}:\d{2}\s*[APap][Mm])\s+-\s+(.+)$`)

// WhatsAppParser parses WhatsApp "Export Chat" TXT files.
type WhatsAppParser struct{}

func NewWhatsAppParser() *WhatsAppParser { return &WhatsAppParser{} }

func (p *WhatsAppParser) Format() Format { return FormatWhatsAppTXT }

func (p *WhatsAppParser) Detect(r io.ReadSeeker) bool {
	scanner := bufio.NewScanner(io.LimitReader(r, 4096))
	for scanner.Scan() {
		if whatsappLineRe.MatchString(scanner.Text()) {
			return true
		}
	}
	return false
}

func (p *WhatsAppParser) Parse(ctx context.Context, r io.Reader, opts Options) (*chatir.ChatIR, error) {
	tz := opts.TimeZone
	if tz == nil {
		tz = time.Local
	}

	platform := "whatsapp"
	if opts.Platform != "" {
		platform = opts.Platform
	}

	participants := make(map[string]*chatir.Participant)
	var messages []chatir.Message
	var currentMsg *chatir.Message
	msgSeq := 0

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		line := scanner.Text()
		matches := whatsappLineRe.FindStringSubmatch(line)

		if matches == nil {
			// Continuation of previous multiline message
			if currentMsg != nil {
				currentMsg.Content += "\n" + line
			}
			continue
		}

		// Flush previous message
		if currentMsg != nil {
			messages = append(messages, *currentMsg)
		}

		dateStr := matches[1]
		timeStr := matches[2]
		rest := matches[3]

		timestamp, err := parseWhatsAppTime(dateStr, timeStr, tz)
		if err != nil {
			continue
		}

		// Check if this is a "Sender: Message" or a system message
		colonIdx := strings.Index(rest, ": ")
		if colonIdx == -1 {
			// System message (e.g., "Alice added Charlie")
			msgSeq++
			currentMsg = &chatir.Message{
				ID:        fmt.Sprintf("wa_%d", msgSeq),
				Timestamp: timestamp,
				SenderID:  "system",
				Content:   rest,
				Type:      chatir.MessageSystem,
			}
			if _, ok := participants["system"]; !ok {
				participants["system"] = &chatir.Participant{ID: "system", Name: "System"}
			}
			continue
		}

		sender := rest[:colonIdx]
		content := rest[colonIdx+2:]

		senderID := strings.ReplaceAll(strings.ToLower(sender), " ", "_")

		isSelf := isSelfParticipant(senderID, sender, opts)
		if _, ok := participants[senderID]; !ok {
			participants[senderID] = &chatir.Participant{
				ID:     senderID,
				Name:   sender,
				IsSelf: isSelf,
			}
		}

		msgType := chatir.MessageText
		if content == "<Media omitted>" {
			msgType = chatir.MessageFile
		}

		msgSeq++
		currentMsg = &chatir.Message{
			ID:        fmt.Sprintf("wa_%d", msgSeq),
			Timestamp: timestamp,
			SenderID:  senderID,
			Content:   content,
			Type:      msgType,
		}
	}

	// Flush last message
	if currentMsg != nil {
		messages = append(messages, *currentMsg)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading WhatsApp export: %w", err)
	}

	if len(messages) == 0 {
		return nil, fmt.Errorf("no messages found in WhatsApp export")
	}

	parts := make([]chatir.Participant, 0, len(participants))
	for _, p := range participants {
		parts = append(parts, *p)
	}

	convType := chatir.ConversationPrivate
	nonSelf := 0
	for _, p := range parts {
		if !p.IsSelf && p.ID != "system" {
			nonSelf++
		}
	}
	if nonSelf > 1 {
		convType = chatir.ConversationGroup
	}

	ir := &chatir.ChatIR{
		Platform:         platform,
		ConversationID:   fmt.Sprintf("wa_%s", messages[0].Timestamp.Format("20060102")),
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

// parseWhatsAppTime parses WhatsApp date/time strings.
func parseWhatsAppTime(dateStr, timeStr string, tz *time.Location) (time.Time, error) {
	combined := dateStr + ", " + timeStr
	// Try 4-digit year
	t, err := time.ParseInLocation("1/2/2006, 3:04 PM", combined, tz)
	if err == nil {
		return t, nil
	}
	// Try 2-digit year
	t, err = time.ParseInLocation("1/2/06, 3:04 PM", combined, tz)
	if err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("cannot parse WhatsApp timestamp %q", combined)
}
