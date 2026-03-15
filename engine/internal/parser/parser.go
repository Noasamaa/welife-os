package parser

import (
	"context"
	"io"
	"time"

	"github.com/welife-os/welife-os/engine/internal/chatir"
)

// Format identifies a chat export format.
type Format string

const (
	FormatWeChatCSV    Format = "wechat_csv"
	FormatTelegramJSON Format = "telegram_json"
	FormatWhatsAppTXT  Format = "whatsapp_txt"
	FormatGenericCSV   Format = "generic_csv"
)

// Options configures parser behavior.
type Options struct {
	SelfName string         // The user's own display name, for marking IsSelf
	SelfIDs  []string       // Alternate self identifiers
	Platform string         // Override platform name
	TimeZone *time.Location // Timezone for timestamp parsing
}

// Parser converts raw chat export data into ChatIR.
type Parser interface {
	// Parse reads from r and returns normalized ChatIR.
	Parse(ctx context.Context, r io.Reader, opts Options) (*chatir.ChatIR, error)

	// Format returns the format this parser handles.
	Format() Format

	// Detect reports whether the data at r looks like this format.
	// Must not consume more than 4096 bytes. Resets the reader position.
	Detect(r io.ReadSeeker) bool
}
