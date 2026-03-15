package parser

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/welife-os/welife-os/engine/internal/chatir"
)

const larkFixture = `{
  "chat_id": "oc_xxx",
  "chat_name": "项目讨论群",
  "messages": [
    {
      "msg_id": "m1",
      "msg_type": "text",
      "create_time": "1705314600",
      "sender": {"id": "ou_001", "name": "张三"},
      "content": "{\"text\": \"Hello everyone\"}"
    },
    {
      "msg_id": "m2",
      "msg_type": "text",
      "create_time": "1705314660",
      "sender": {"id": "ou_002", "name": "李四"},
      "content": "{\"text\": \"Hi there\"}"
    },
    {
      "msg_id": "m3",
      "msg_type": "image",
      "create_time": "1705314720",
      "sender": {"id": "ou_001", "name": "张三"},
      "content": "{\"image_key\": \"img_xxx\"}"
    }
  ]
}`

func TestLarkParse(t *testing.T) {
	p := NewLarkParser()
	opts := Options{
		SelfName: "张三",
		TimeZone: time.UTC,
	}

	ir, err := p.Parse(context.Background(), strings.NewReader(larkFixture), opts)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if ir.Platform != "lark" {
		t.Errorf("platform = %q, want %q", ir.Platform, "lark")
	}

	if ir.Metadata.MessageCount != 3 {
		t.Errorf("message count = %d, want 3", ir.Metadata.MessageCount)
	}

	if ir.ConversationID != "lk_oc_xxx" {
		t.Errorf("conversation ID = %q, want %q", ir.ConversationID, "lk_oc_xxx")
	}

	// Text content extracted from JSON
	if ir.Messages[0].Content != "Hello everyone" {
		t.Errorf("messages[0].Content = %q, want %q", ir.Messages[0].Content, "Hello everyone")
	}

	// Second text message
	if ir.Messages[1].Content != "Hi there" {
		t.Errorf("messages[1].Content = %q, want %q", ir.Messages[1].Content, "Hi there")
	}

	// Image message type
	if ir.Messages[2].Type != chatir.MessageImage {
		t.Errorf("messages[2].Type = %q, want %q", ir.Messages[2].Type, chatir.MessageImage)
	}

	// Self participant
	foundSelf := false
	for _, p := range ir.Participants {
		if p.Name == "张三" && p.IsSelf {
			foundSelf = true
		}
	}
	if !foundSelf {
		t.Error("expected 张三 to be marked as self participant")
	}

	// Timestamp check (1705312200 = 2024-01-15 10:30:00 UTC)
	expected := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	if !ir.Messages[0].Timestamp.Equal(expected) {
		t.Errorf("messages[0].Timestamp = %v, want %v", ir.Messages[0].Timestamp, expected)
	}
}

func TestLarkParsePrivate(t *testing.T) {
	fixture := `{
		"chat_id": "oc_yyy",
		"chat_name": "Private Chat",
		"messages": [
			{
				"msg_id": "m1",
				"msg_type": "text",
				"create_time": "1705314600",
				"sender": {"id": "ou_001", "name": "Alice"},
				"content": "{\"text\": \"Hey\"}"
			}
		]
	}`

	p := NewLarkParser()
	ir, err := p.Parse(context.Background(), strings.NewReader(fixture), Options{TimeZone: time.UTC})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if ir.ConversationType != chatir.ConversationPrivate {
		t.Errorf("conversation type = %q, want %q", ir.ConversationType, chatir.ConversationPrivate)
	}
}

func TestLarkParseEmpty(t *testing.T) {
	fixture := `{"chat_id": "oc_xxx", "chat_name": "Test", "messages": []}`

	p := NewLarkParser()
	_, err := p.Parse(context.Background(), strings.NewReader(fixture), Options{TimeZone: time.UTC})
	if err == nil {
		t.Error("expected error for empty messages")
	}
}

func TestLarkDetect(t *testing.T) {
	tests := []struct {
		name string
		data string
		want bool
	}{
		{"valid lark chat_id", `{"chat_id":"oc_xxx","messages":[]}`, true},
		{"valid lark msg_type", `{"messages":[{"msg_type":"text"}]}`, true},
		{"telegram", `{"name":"test","messages":[{"date_unixtime":"123"}]}`, false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewLarkParser().Detect(strings.NewReader(tt.data)); got != tt.want {
				t.Errorf("Detect() = %v, want %v", got, tt.want)
			}
		})
	}
}
