package parser

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestGenericCSVParse(t *testing.T) {
	data, err := os.ReadFile("testdata/generic_sample.csv")
	if err != nil {
		t.Fatalf("reading test data: %v", err)
	}

	p := NewGenericCSVParser()
	opts := Options{TimeZone: time.UTC}

	ir, err := p.Parse(context.Background(), strings.NewReader(string(data)), opts)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if ir.Platform != "generic" {
		t.Errorf("platform = %q, want %q", ir.Platform, "generic")
	}

	if ir.Metadata.MessageCount != 4 {
		t.Errorf("message count = %d, want 4", ir.Metadata.MessageCount)
	}

	// Check participant count (Alice + Bob)
	if len(ir.Participants) != 2 {
		t.Errorf("participant count = %d, want 2", len(ir.Participants))
	}

	// Check first message content
	if ir.Messages[0].Content != "你好" {
		t.Errorf("messages[0].Content = %q, want %q", ir.Messages[0].Content, "你好")
	}
}

func TestGenericCSVDetect(t *testing.T) {
	tests := []struct {
		name string
		data string
		want bool
	}{
		{"valid", "timestamp,sender,content\n", true},
		{"alt names", "date,from,message\n", true},
		{"missing sender", "timestamp,content\n", false},
		{"wechat format", "talker,create_time,type,content\n", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewGenericCSVParser().Detect(strings.NewReader(tt.data)); got != tt.want {
				t.Errorf("Detect() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenericCSVParseMissingColumns(t *testing.T) {
	csv := "name,value\na,1\n"
	p := NewGenericCSVParser()
	_, err := p.Parse(context.Background(), strings.NewReader(csv), Options{TimeZone: time.UTC})
	if err == nil {
		t.Error("expected error for missing columns, got nil")
	}
}
