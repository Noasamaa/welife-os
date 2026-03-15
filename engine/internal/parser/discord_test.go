package parser

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/welife-os/welife-os/engine/internal/chatir"
)

const discordFixture = `{
  "guild": {"id": "111", "name": "Test Server"},
  "channel": {"id": "222", "name": "general", "type": "TextChannel"},
  "messages": [
    {
      "id": "1001",
      "timestamp": "2024-01-15T10:30:00+00:00",
      "content": "Hello everyone",
      "author": {"id": "u1", "name": "alice", "nickname": "Alice"},
      "attachments": [],
      "embeds": []
    },
    {
      "id": "1002",
      "timestamp": "2024-01-15T10:31:00+00:00",
      "content": "Check this out",
      "author": {"id": "u2", "name": "bob", "nickname": "Bob"},
      "attachments": [
        {"id": "a1", "url": "https://cdn.example.com/file.png", "fileName": "file.png"}
      ],
      "embeds": []
    },
    {
      "id": "1003",
      "timestamp": "2024-01-15T10:32:00+00:00",
      "content": "Nice link",
      "author": {"id": "u1", "name": "alice", "nickname": "Alice"},
      "attachments": [],
      "embeds": [{"title": "Example", "description": "An example embed", "url": "https://example.com"}]
    }
  ]
}`

func TestDiscordParse(t *testing.T) {
	p := NewDiscordParser()
	opts := Options{
		SelfName: "Alice",
		TimeZone: time.UTC,
	}

	ir, err := p.Parse(context.Background(), strings.NewReader(discordFixture), opts)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if ir.Platform != "discord" {
		t.Errorf("platform = %q, want %q", ir.Platform, "discord")
	}

	if ir.Metadata.MessageCount != 3 {
		t.Errorf("message count = %d, want 3", ir.Metadata.MessageCount)
	}

	if ir.ConversationType != chatir.ConversationGroup {
		t.Errorf("conversation type = %q, want %q", ir.ConversationType, chatir.ConversationGroup)
	}

	// First message is text
	if ir.Messages[0].Content != "Hello everyone" {
		t.Errorf("messages[0].Content = %q, want %q", ir.Messages[0].Content, "Hello everyone")
	}

	// Second message has attachment
	if ir.Messages[1].Type != chatir.MessageFile {
		t.Errorf("messages[1].Type = %q, want %q", ir.Messages[1].Type, chatir.MessageFile)
	}
	if len(ir.Messages[1].Attachments) != 1 {
		t.Fatalf("messages[1].Attachments len = %d, want 1", len(ir.Messages[1].Attachments))
	}
	if ir.Messages[1].Attachments[0].Name != "file.png" {
		t.Errorf("attachment name = %q, want %q", ir.Messages[1].Attachments[0].Name, "file.png")
	}

	// Third message has embed appended
	if !strings.Contains(ir.Messages[2].Content, "[Embed: Example]") {
		t.Errorf("messages[2].Content = %q, should contain embed", ir.Messages[2].Content)
	}

	// Self participant
	foundSelf := false
	for _, p := range ir.Participants {
		if p.Name == "Alice" && p.IsSelf {
			foundSelf = true
		}
	}
	if !foundSelf {
		t.Error("expected Alice to be marked as self participant")
	}
}

func TestDiscordParseDM(t *testing.T) {
	fixture := `{
		"guild": {"id": "", "name": ""},
		"channel": {"id": "dm1", "name": "Direct Message", "type": "DirectTextChannel"},
		"messages": [
			{
				"id": "2001",
				"timestamp": "2024-01-15T10:30:00+00:00",
				"content": "Hey",
				"author": {"id": "u1", "name": "alice", "nickname": "Alice"}
			}
		]
	}`

	p := NewDiscordParser()
	ir, err := p.Parse(context.Background(), strings.NewReader(fixture), Options{TimeZone: time.UTC})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if ir.ConversationType != chatir.ConversationPrivate {
		t.Errorf("conversation type = %q, want %q", ir.ConversationType, chatir.ConversationPrivate)
	}
}

func TestDiscordParseEmpty(t *testing.T) {
	fixture := `{"guild": {"id": "111"}, "channel": {"id": "222"}, "messages": []}`

	p := NewDiscordParser()
	_, err := p.Parse(context.Background(), strings.NewReader(fixture), Options{TimeZone: time.UTC})
	if err == nil {
		t.Error("expected error for empty messages")
	}
}

func TestDiscordDetect(t *testing.T) {
	tests := []struct {
		name string
		data string
		want bool
	}{
		{"valid discord", `{"guild":{"id":"1"},"channel":{"id":"2"},"messages":[]}`, true},
		{"telegram", `{"name":"test","messages":[{"date_unixtime":"123"}]}`, false},
		{"wechat csv", "talker,create_time,type,content\n", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewDiscordParser().Detect(strings.NewReader(tt.data)); got != tt.want {
				t.Errorf("Detect() = %v, want %v", got, tt.want)
			}
		})
	}
}
