package parser

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/welife-os/welife-os/engine/internal/chatir"
)

// appleEpoch is 2001-01-01 00:00:00 UTC, the reference time for iMessage timestamps.
var appleEpoch = time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)

// IMessageParser parses macOS iMessage chat.db (SQLite) exports.
type IMessageParser struct{}

func NewIMessageParser() *IMessageParser { return &IMessageParser{} }

func (p *IMessageParser) Format() Format { return FormatIMessageDB }

func (p *IMessageParser) Detect(r io.ReadSeeker) bool {
	header := make([]byte, 16)
	n, err := r.Read(header)
	if err != nil || n < 16 {
		return false
	}
	return string(header) == "SQLite format 3\x00"
}

func (p *IMessageParser) Parse(ctx context.Context, r io.Reader, opts Options) (*chatir.ChatIR, error) {
	// SQLite requires file access; write reader content to a temp file.
	tmpFile, err := os.CreateTemp("", "imessage-*.db")
	if err != nil {
		return nil, fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	bufReader := bufio.NewReaderSize(r, 64*1024)
	if _, err := io.Copy(tmpFile, bufReader); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("writing temp file: %w", err)
	}
	tmpFile.Close()

	db, err := sql.Open("sqlite3", tmpPath)
	if err != nil {
		return nil, fmt.Errorf("opening iMessage database: %w", err)
	}
	defer db.Close()

	return p.queryMessages(ctx, db, opts)
}

func (p *IMessageParser) queryMessages(ctx context.Context, db *sql.DB, opts Options) (*chatir.ChatIR, error) {
	tz := opts.TimeZone
	if tz == nil {
		tz = time.Local
	}

	platform := "imessage"
	if opts.Platform != "" {
		platform = opts.Platform
	}

	query := `SELECT m.ROWID, m.text, m.date, m.is_from_me, h.id as handle_id
		FROM message m
		LEFT JOIN handle h ON m.handle_id = h.ROWID
		WHERE m.text IS NOT NULL
		ORDER BY m.date`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying messages: %w", err)
	}
	defer rows.Close()

	participants := make(map[string]*chatir.Participant)
	var messages []chatir.Message

	selfID := "self"
	selfName := opts.SelfName
	if selfName == "" {
		selfName = "Me"
	}
	if len(opts.SelfIDs) > 0 {
		selfID = opts.SelfIDs[0]
	}

	for rows.Next() {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		var rowID int64
		var text string
		var appleDate int64
		var isFromMe int
		var handleID sql.NullString

		if err := rows.Scan(&rowID, &text, &appleDate, &isFromMe, &handleID); err != nil {
			continue
		}

		timestamp := appleEpoch.Add(time.Duration(appleDate) * time.Nanosecond).In(tz)

		var senderID, senderName string
		isSelf := isFromMe == 1
		if isSelf {
			senderID = selfID
			senderName = selfName
		} else {
			senderID = handleID.String
			if senderID == "" {
				senderID = "unknown"
			}
			senderName = senderID
		}

		if _, ok := participants[senderID]; !ok {
			participants[senderID] = &chatir.Participant{
				ID:     senderID,
				Name:   senderName,
				IsSelf: isSelf,
			}
		}

		messages = append(messages, chatir.Message{
			ID:        fmt.Sprintf("im_%d", rowID),
			Timestamp: timestamp,
			SenderID:  senderID,
			Content:   strings.TrimSpace(text),
			Type:      chatir.MessageText,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating messages: %w", err)
	}

	if len(messages) == 0 {
		return nil, fmt.Errorf("no messages found in iMessage database")
	}

	parts := make([]chatir.Participant, 0, len(participants))
	for _, pt := range participants {
		parts = append(parts, *pt)
	}

	convType := chatir.ConversationPrivate
	nonSelf := 0
	for _, pt := range parts {
		if !pt.IsSelf {
			nonSelf++
		}
	}
	if nonSelf > 1 {
		convType = chatir.ConversationGroup
	}

	ir := &chatir.ChatIR{
		Platform:         platform,
		ConversationID:   fmt.Sprintf("im_%s", messages[0].Timestamp.Format("20060102")),
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
