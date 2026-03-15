package parser

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/welife-os/welife-os/engine/internal/chatir"
)

func TestTelegramParse(t *testing.T) {
	data, err := os.ReadFile("testdata/telegram_sample.json")
	if err != nil {
		t.Fatalf("reading test data: %v", err)
	}

	p := NewTelegramParser()
	opts := Options{
		SelfName: "Alice",
		TimeZone: time.UTC,
	}

	ir, err := p.Parse(context.Background(), strings.NewReader(string(data)), opts)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if ir.Platform != "telegram" {
		t.Errorf("platform = %q, want %q", ir.Platform, "telegram")
	}

	// Service message should be skipped
	if ir.Metadata.MessageCount != 3 {
		t.Errorf("message count = %d, want 3", ir.Metadata.MessageCount)
	}

	// Second message has array text field
	if !strings.Contains(ir.Messages[1].Content, "@Alice") {
		t.Errorf("message[1].Content = %q, should contain @Alice", ir.Messages[1].Content)
	}

	// Second message is a reply
	if ir.Messages[1].ReplyTo != "tg_1" {
		t.Errorf("message[1].ReplyTo = %q, want %q", ir.Messages[1].ReplyTo, "tg_1")
	}

	// Third message has photo attachment
	if ir.Messages[2].Type != chatir.MessageImage {
		t.Errorf("message[2].Type = %q, want %q", ir.Messages[2].Type, chatir.MessageImage)
	}
	if len(ir.Messages[2].Attachments) == 0 {
		t.Error("message[2] should have attachments")
	}

	// Conversation type should be group (supergroup)
	if ir.ConversationType != chatir.ConversationGroup {
		t.Errorf("conversation type = %q, want %q", ir.ConversationType, chatir.ConversationGroup)
	}
}

func TestTelegramDetect(t *testing.T) {
	tests := []struct {
		name string
		data string
		want bool
	}{
		{"valid telegram", `{"name":"test","messages":[{"date_unixtime":"123"}]}`, true},
		{"wechat csv", "talker,create_time,type,content\n", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewTelegramParser().Detect(strings.NewReader(tt.data)); got != tt.want {
				t.Errorf("Detect() = %v, want %v", got, tt.want)
			}
		})
	}
}
