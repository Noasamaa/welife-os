package parser

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/welife-os/welife-os/engine/internal/chatir"
)

func TestWhatsAppParse(t *testing.T) {
	data, err := os.ReadFile("testdata/whatsapp_sample.txt")
	if err != nil {
		t.Fatalf("reading test data: %v", err)
	}

	p := NewWhatsAppParser()
	opts := Options{TimeZone: time.UTC}

	ir, err := p.Parse(context.Background(), strings.NewReader(string(data)), opts)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if ir.Platform != "whatsapp" {
		t.Errorf("platform = %q, want %q", ir.Platform, "whatsapp")
	}

	// 5 messages + 1 system + 1 charlie = 7 lines, but system is one of them
	// Alice(3) + Bob(2) + system(1) + Charlie(1) = 7 messages
	if ir.Metadata.MessageCount != 7 {
		t.Errorf("message count = %d, want 7", ir.Metadata.MessageCount)
	}

	// Third message should contain multiline content
	found := false
	for _, msg := range ir.Messages {
		if strings.Contains(msg.Content, "absolutely amazing") {
			found = true
			if !strings.Contains(msg.Content, "\n") {
				t.Error("multiline message should contain newline")
			}
		}
	}
	if !found {
		t.Error("multiline message not found")
	}

	// Check media omitted
	hasMedia := false
	for _, msg := range ir.Messages {
		if msg.Type == chatir.MessageFile && msg.Content == "<Media omitted>" {
			hasMedia = true
		}
	}
	if !hasMedia {
		t.Error("media omitted message not found")
	}

	// Check system message
	hasSystem := false
	for _, msg := range ir.Messages {
		if msg.Type == chatir.MessageSystem {
			hasSystem = true
		}
	}
	if !hasSystem {
		t.Error("system message not found")
	}
}

func TestWhatsAppDetect(t *testing.T) {
	tests := []struct {
		name string
		data string
		want bool
	}{
		{"valid", "02/09/2024, 8:34 PM - Alice: Hello\n", true},
		{"csv header", "timestamp,sender,content\n", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewWhatsAppParser().Detect(strings.NewReader(tt.data)); got != tt.want {
				t.Errorf("Detect() = %v, want %v", got, tt.want)
			}
		})
	}
}
