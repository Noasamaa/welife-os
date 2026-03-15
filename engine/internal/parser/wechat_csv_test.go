package parser

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/welife-os/welife-os/engine/internal/chatir"
)

func TestWeChatCSVParse(t *testing.T) {
	data, err := os.ReadFile("testdata/wechat_sample.csv")
	if err != nil {
		t.Fatalf("reading test data: %v", err)
	}

	p := NewWeChatCSVParser()
	opts := Options{
		SelfName: "我自己",
		SelfIDs:  []string{"wxid_self"},
		TimeZone: time.UTC,
	}

	ir, err := p.Parse(context.Background(), strings.NewReader(string(data)), opts)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if ir.Platform != "wechat" {
		t.Errorf("platform = %q, want %q", ir.Platform, "wechat")
	}

	if ir.Metadata.MessageCount != 10 {
		t.Errorf("message count = %d, want 10", ir.Metadata.MessageCount)
	}

	// Verify participants
	selfFound := false
	for _, p := range ir.Participants {
		if p.IsSelf {
			selfFound = true
			if p.Name != "我自己" {
				t.Errorf("self name = %q, want %q", p.Name, "我自己")
			}
		}
	}
	if !selfFound {
		t.Error("no self participant found")
	}

	// Verify message types
	typeTests := []struct {
		index    int
		wantType chatir.MessageType
	}{
		{0, chatir.MessageText},
		{3, chatir.MessageImage},
		{7, chatir.MessageAudio},
		{8, chatir.MessageVideo},
		{9, chatir.MessageSystem},
	}
	for _, tt := range typeTests {
		if ir.Messages[tt.index].Type != tt.wantType {
			t.Errorf("message[%d].Type = %q, want %q", tt.index, ir.Messages[tt.index].Type, tt.wantType)
		}
	}

	// Verify conversation type is group (alice + bob + self)
	if ir.ConversationType != chatir.ConversationGroup {
		t.Errorf("conversation type = %q, want %q", ir.ConversationType, chatir.ConversationGroup)
	}
}

func TestWeChatCSVDetect(t *testing.T) {
	tests := []struct {
		name string
		data string
		want bool
	}{
		{"valid header", "talker,create_time,type,content,isSend\n", true},
		{"case insensitive", "Talker,Create_Time,Type,Content,IsSend\n", true},
		{"missing talker", "sender,create_time,type,content,isSend\n", false},
		{"empty", "", false},
		{"telegram json", `{"name":"chat","messages":[]}`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs := strings.NewReader(tt.data)
			p := NewWeChatCSVParser()
			if got := p.Detect(rs); got != tt.want {
				t.Errorf("Detect() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWeChatCSVParseEmptyContent(t *testing.T) {
	csv := "talker,create_time,type,content,isSend\n"
	p := NewWeChatCSVParser()
	_, err := p.Parse(context.Background(), strings.NewReader(csv), Options{TimeZone: time.UTC})
	if err == nil {
		t.Error("expected error for empty CSV, got nil")
	}
}

func TestWeChatCSVParseMissingColumn(t *testing.T) {
	csv := "sender,time,text\na,1,hello\n"
	p := NewWeChatCSVParser()
	_, err := p.Parse(context.Background(), strings.NewReader(csv), Options{TimeZone: time.UTC})
	if err == nil {
		t.Error("expected error for missing columns, got nil")
	}
}
