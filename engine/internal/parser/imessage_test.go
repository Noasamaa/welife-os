package parser

import (
	"strings"
	"testing"
)

func TestIMessageDetect(t *testing.T) {
	tests := []struct {
		name string
		data string
		want bool
	}{
		{
			"valid sqlite header",
			"SQLite format 3\x00" + strings.Repeat("\x00", 100),
			true,
		},
		{
			"json data",
			`{"messages":[]}`,
			false,
		},
		{
			"too short",
			"SQLite",
			false,
		},
		{
			"empty",
			"",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewIMessageParser().Detect(strings.NewReader(tt.data)); got != tt.want {
				t.Errorf("Detect() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIMessageFormat(t *testing.T) {
	p := NewIMessageParser()
	if got := p.Format(); got != FormatIMessageDB {
		t.Errorf("Format() = %q, want %q", got, FormatIMessageDB)
	}
}
