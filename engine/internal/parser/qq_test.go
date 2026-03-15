package parser

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/welife-os/welife-os/engine/internal/chatir"
)

const qqFixturePrivate = `消息分组:我的好友
================================================
消息对象:张三
================================================

2024-01-15 10:30:00 张三
你好，最近怎么样？

2024-01-15 10:31:00 我
挺好的，谢谢

2024-01-15 10:32:00 张三
周末有空吗？
一起去吃饭

================================================
`

const qqFixtureGroup = `消息分组:我的群
================================================
消息对象:项目讨论群
================================================

2024-01-15 10:30:00 张三
大家好

2024-01-15 10:31:00 李四
你好

================================================
`

func TestQQParsePrivate(t *testing.T) {
	p := NewQQParser()
	opts := Options{
		SelfName: "我",
		TimeZone: time.UTC,
	}

	ir, err := p.Parse(context.Background(), strings.NewReader(qqFixturePrivate), opts)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if ir.Platform != "qq" {
		t.Errorf("platform = %q, want %q", ir.Platform, "qq")
	}

	if ir.Metadata.MessageCount != 3 {
		t.Errorf("message count = %d, want 3", ir.Metadata.MessageCount)
	}

	if ir.ConversationType != chatir.ConversationPrivate {
		t.Errorf("conversation type = %q, want %q", ir.ConversationType, chatir.ConversationPrivate)
	}

	// First message content
	if ir.Messages[0].Content != "你好，最近怎么样？" {
		t.Errorf("messages[0].Content = %q, want %q", ir.Messages[0].Content, "你好，最近怎么样？")
	}

	// Self identification
	if ir.Messages[1].SenderID != "我" {
		t.Errorf("messages[1].SenderID = %q, want %q", ir.Messages[1].SenderID, "我")
	}

	// Multiline message
	if !strings.Contains(ir.Messages[2].Content, "周末有空吗？") ||
		!strings.Contains(ir.Messages[2].Content, "一起去吃饭") {
		t.Errorf("messages[2].Content = %q, want multiline content", ir.Messages[2].Content)
	}

	// Self participant marked
	foundSelf := false
	for _, p := range ir.Participants {
		if p.IsSelf {
			foundSelf = true
		}
	}
	if !foundSelf {
		t.Error("expected self participant to be marked")
	}
}

func TestQQParseGroup(t *testing.T) {
	p := NewQQParser()
	opts := Options{TimeZone: time.UTC}

	ir, err := p.Parse(context.Background(), strings.NewReader(qqFixtureGroup), opts)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if ir.ConversationType != chatir.ConversationGroup {
		t.Errorf("conversation type = %q, want %q", ir.ConversationType, chatir.ConversationGroup)
	}

	if ir.Metadata.MessageCount != 2 {
		t.Errorf("message count = %d, want 2", ir.Metadata.MessageCount)
	}
}

func TestQQParseEmpty(t *testing.T) {
	fixture := `消息分组:我的好友
================================================
消息对象:张三
================================================
`

	p := NewQQParser()
	_, err := p.Parse(context.Background(), strings.NewReader(fixture), Options{TimeZone: time.UTC})
	if err == nil {
		t.Error("expected error for empty messages")
	}
}

func TestQQDetect(t *testing.T) {
	tests := []struct {
		name string
		data string
		want bool
	}{
		{"valid qq private", "消息分组:我的好友\n================================================\n", true},
		{"valid qq object", "消息对象:张三\n", true},
		{"separator only", "================================================\n", true},
		{"telegram json", `{"messages":[]}`, false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewQQParser().Detect(strings.NewReader(tt.data)); got != tt.want {
				t.Errorf("Detect() = %v, want %v", got, tt.want)
			}
		})
	}
}
