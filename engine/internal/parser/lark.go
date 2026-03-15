package parser

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/welife-os/welife-os/engine/internal/chatir"
)

// LarkParser parses Lark (飞书) JSON exports.
type LarkParser struct{}

func NewLarkParser() *LarkParser { return &LarkParser{} }

func (p *LarkParser) Format() Format { return FormatLarkJSON }

func (p *LarkParser) Detect(r io.ReadSeeker) bool {
	scanner := bufio.NewScanner(io.LimitReader(r, 4096))
	var buf strings.Builder
	for scanner.Scan() {
		buf.WriteString(scanner.Text())
		if buf.Len() > 2048 {
			break
		}
	}
	text := buf.String()
	return strings.Contains(text, `"chat_id"`) || strings.Contains(text, `"msg_type"`)
}

type larkExport struct {
	ChatID   string        `json:"chat_id"`
	ChatName string        `json:"chat_name"`
	Messages []larkMessage `json:"messages"`
}

type larkMessage struct {
	MsgID      string     `json:"msg_id"`
	MsgType    string     `json:"msg_type"`
	CreateTime string     `json:"create_time"`
	Sender     larkSender `json:"sender"`
	Content    string     `json:"content"`
}

type larkSender struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (p *LarkParser) Parse(ctx context.Context, r io.Reader, opts Options) (*chatir.ChatIR, error) {
	var export larkExport
	if err := json.NewDecoder(r).Decode(&export); err != nil {
		return nil, fmt.Errorf("decoding lark JSON: %w", err)
	}

	tz := opts.TimeZone
	if tz == nil {
		tz = time.Local
	}

	platform := "lark"
	if opts.Platform != "" {
		platform = opts.Platform
	}

	participants := make(map[string]*chatir.Participant)
	var messages []chatir.Message

	for _, lm := range export.Messages {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		unixSec, err := strconv.ParseInt(lm.CreateTime, 10, 64)
		if err != nil {
			continue
		}
		timestamp := time.Unix(unixSec, 0).In(tz)

		senderID := lm.Sender.ID
		senderName := lm.Sender.Name

		isSelf := isSelfParticipant(senderID, senderName, opts)
		if _, ok := participants[senderID]; !ok {
			participants[senderID] = &chatir.Participant{
				ID:     senderID,
				Name:   senderName,
				IsSelf: isSelf,
			}
		}

		content := extractLarkContent(lm.MsgType, lm.Content)
		msgType := larkMsgType(lm.MsgType)

		messages = append(messages, chatir.Message{
			ID:        fmt.Sprintf("lk_%s", lm.MsgID),
			Timestamp: timestamp,
			SenderID:  senderID,
			Content:   content,
			Type:      msgType,
		})
	}

	if len(messages) == 0 {
		return nil, fmt.Errorf("no messages found in Lark export")
	}

	parts := make([]chatir.Participant, 0, len(participants))
	for _, pt := range participants {
		parts = append(parts, *pt)
	}

	convType := chatir.ConversationGroup
	if len(parts) <= 2 {
		convType = chatir.ConversationPrivate
	}

	convID := export.ChatID
	if convID == "" {
		convID = messages[0].Timestamp.Format("20060102")
	}

	ir := &chatir.ChatIR{
		Platform:         platform,
		ConversationID:   fmt.Sprintf("lk_%s", convID),
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

// extractLarkContent extracts text from Lark's JSON-encoded content field.
func extractLarkContent(msgType, raw string) string {
	if msgType == "text" {
		var textContent struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal([]byte(raw), &textContent); err == nil {
			return textContent.Text
		}
	}
	if raw != "" {
		return raw
	}
	return ""
}

func larkMsgType(msgType string) chatir.MessageType {
	switch msgType {
	case "text":
		return chatir.MessageText
	case "image":
		return chatir.MessageImage
	case "file":
		return chatir.MessageFile
	case "audio":
		return chatir.MessageAudio
	case "video":
		return chatir.MessageVideo
	case "system":
		return chatir.MessageSystem
	default:
		return chatir.MessageText
	}
}
