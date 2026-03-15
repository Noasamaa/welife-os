package parser

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/welife-os/welife-os/engine/internal/chatir"
)

// WeChatCSVParser parses WeChatMsg CSV exports.
type WeChatCSVParser struct{}

func NewWeChatCSVParser() *WeChatCSVParser { return &WeChatCSVParser{} }

func (p *WeChatCSVParser) Format() Format { return FormatWeChatCSV }

func (p *WeChatCSVParser) Detect(r io.ReadSeeker) bool {
	scanner := bufio.NewScanner(io.LimitReader(r, 4096))
	if !scanner.Scan() {
		return false
	}
	header := strings.ToLower(scanner.Text())
	return strings.Contains(header, "talker") &&
		strings.Contains(header, "create_time") &&
		strings.Contains(header, "content")
}

func (p *WeChatCSVParser) Parse(ctx context.Context, r io.Reader, opts Options) (*chatir.ChatIR, error) {
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

	required := []string{"talker", "create_time", "content"}
	for _, col := range required {
		if _, ok := colIdx[col]; !ok {
			return nil, fmt.Errorf("missing required column: %s", col)
		}
	}

	tz := opts.TimeZone
	if tz == nil {
		tz = time.Local
	}

	platform := "wechat"
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

		talker := record[colIdx["talker"]]
		content := record[colIdx["content"]]
		createTimeStr := record[colIdx["create_time"]]

		ts, err := strconv.ParseInt(createTimeStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid create_time %q: %w", lineNum, createTimeStr, err)
		}
		timestamp := time.Unix(ts, 0).In(tz)

		// Determine message type from numeric type field
		msgType := chatir.MessageText
		if idx, ok := colIdx["type"]; ok && idx < len(record) {
			msgType = wechatTypeToMessageType(record[idx])
		}

		// Determine sender
		isSend := false
		if idx, ok := colIdx["issend"]; ok && idx < len(record) {
			isSend = record[idx] == "1"
		}

		senderID := talker
		if isSend {
			senderID = selfID(opts)
		}

		// Track participants
		if _, ok := participants[senderID]; !ok {
			name := senderID
			self := false
			if isSend {
				name = opts.SelfName
				if name == "" {
					name = "我"
				}
				self = true
			}
			participants[senderID] = &chatir.Participant{
				ID:     senderID,
				Name:   name,
				IsSelf: self,
			}
		}

		msgID := fmt.Sprintf("wc_%s_%d", senderID, ts)

		msg := chatir.Message{
			ID:        msgID,
			Timestamp: timestamp,
			SenderID:  senderID,
			Content:   content,
			Type:      msgType,
		}

		// Handle attachments for non-text types
		if idx, ok := colIdx["imgpath"]; ok && idx < len(record) && record[idx] != "" {
			msg.Attachments = []chatir.Attachment{{
				Type: string(msgType),
				Path: record[idx],
			}}
		}

		messages = append(messages, msg)
	}

	if len(messages) == 0 {
		return nil, fmt.Errorf("no messages found in CSV")
	}

	// Build participant list
	parts := make([]chatir.Participant, 0, len(participants))
	for _, p := range participants {
		parts = append(parts, *p)
	}

	// Determine conversation type
	convType := chatir.ConversationPrivate
	nonSelfCount := 0
	for _, p := range parts {
		if !p.IsSelf {
			nonSelfCount++
		}
	}
	if nonSelfCount > 1 {
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

func selfID(opts Options) string {
	if len(opts.SelfIDs) > 0 {
		return opts.SelfIDs[0]
	}
	return "self"
}

// wechatTypeToMessageType maps WeChatMsg numeric type codes to ChatIR MessageType.
func wechatTypeToMessageType(typeStr string) chatir.MessageType {
	switch strings.TrimSpace(typeStr) {
	case "1":
		return chatir.MessageText
	case "3":
		return chatir.MessageImage
	case "34":
		return chatir.MessageAudio
	case "43":
		return chatir.MessageVideo
	case "49":
		return chatir.MessageFile
	case "10000":
		return chatir.MessageSystem
	default:
		return chatir.MessageText
	}
}
