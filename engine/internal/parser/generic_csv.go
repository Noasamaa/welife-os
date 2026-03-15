package parser

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/welife-os/welife-os/engine/internal/chatir"
)

// GenericCSVParser parses CSV files with auto-detected column mapping.
// It looks for columns named timestamp/time/date, sender/from/author, content/message/text.
type GenericCSVParser struct{}

func NewGenericCSVParser() *GenericCSVParser { return &GenericCSVParser{} }

func (p *GenericCSVParser) Format() Format { return FormatGenericCSV }

func (p *GenericCSVParser) Detect(r io.ReadSeeker) bool {
	scanner := bufio.NewScanner(io.LimitReader(r, 4096))
	if !scanner.Scan() {
		return false
	}
	header := strings.ToLower(scanner.Text())
	hasTime := strings.Contains(header, "timestamp") || strings.Contains(header, "time") || strings.Contains(header, "date")
	hasSender := strings.Contains(header, "sender") || strings.Contains(header, "from") || strings.Contains(header, "author")
	hasContent := strings.Contains(header, "content") || strings.Contains(header, "message") || strings.Contains(header, "text")
	return hasTime && hasSender && hasContent
}

func (p *GenericCSVParser) Parse(ctx context.Context, r io.Reader, opts Options) (*chatir.ChatIR, error) {
	reader := csv.NewReader(r)
	reader.LazyQuotes = true

	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("reading CSV header: %w", err)
	}

	colIdx := make(map[string]int)
	for i, h := range header {
		colIdx[strings.TrimSpace(strings.ToLower(h))] = i
	}

	// Find timestamp column
	tsCol := findColumn(colIdx, "timestamp", "time", "date", "created_at")
	if tsCol < 0 {
		return nil, fmt.Errorf("no timestamp column found (expected: timestamp, time, date)")
	}

	// Find sender column
	senderCol := findColumn(colIdx, "sender", "from", "author", "user", "name")
	if senderCol < 0 {
		return nil, fmt.Errorf("no sender column found (expected: sender, from, author)")
	}

	// Find content column
	contentCol := findColumn(colIdx, "content", "message", "text", "body")
	if contentCol < 0 {
		return nil, fmt.Errorf("no content column found (expected: content, message, text)")
	}

	tz := opts.TimeZone
	if tz == nil {
		tz = time.Local
	}

	platform := "generic"
	if opts.Platform != "" {
		platform = opts.Platform
	}

	participants := make(map[string]*chatir.Participant)
	var messages []chatir.Message
	lineNum := 1

	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum+1, err)
		}
		lineNum++

		if tsCol >= len(record) || senderCol >= len(record) || contentCol >= len(record) {
			continue
		}

		sender := strings.TrimSpace(record[senderCol])
		content := record[contentCol]
		tsStr := strings.TrimSpace(record[tsCol])

		timestamp, err := parseFlexibleTime(tsStr, tz)
		if err != nil {
			continue
		}

		senderID := strings.ReplaceAll(strings.ToLower(sender), " ", "_")
		isSelf := isSelfParticipant(senderID, sender, opts)

		if _, ok := participants[senderID]; !ok {
			participants[senderID] = &chatir.Participant{
				ID:     senderID,
				Name:   sender,
				IsSelf: isSelf,
			}
		}

		messages = append(messages, chatir.Message{
			ID:        fmt.Sprintf("gen_%d", lineNum),
			Timestamp: timestamp,
			SenderID:  senderID,
			Content:   content,
			Type:      chatir.MessageText,
		})
	}

	if len(messages) == 0 {
		return nil, fmt.Errorf("no messages found in CSV")
	}

	parts := make([]chatir.Participant, 0, len(participants))
	for _, p := range participants {
		parts = append(parts, *p)
	}

	convType := chatir.ConversationPrivate
	if len(parts) > 2 {
		convType = chatir.ConversationGroup
	}

	ir := &chatir.ChatIR{
		Platform:         platform,
		ConversationID:   fmt.Sprintf("%s_%s", platform, messages[0].Timestamp.Format("20060102")),
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

// findColumn returns the index of the first matching column name, or -1.
func findColumn(colIdx map[string]int, names ...string) int {
	for _, name := range names {
		if idx, ok := colIdx[name]; ok {
			return idx
		}
	}
	return -1
}

// parseFlexibleTime tries multiple time formats.
func parseFlexibleTime(s string, tz *time.Location) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006/01/02 15:04:05",
		"01/02/2006 15:04:05",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.ParseInLocation(f, s, tz); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse time %q", s)
}
