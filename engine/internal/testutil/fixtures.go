package testutil

import (
	"path/filepath"
	"testing"

	"github.com/welife-os/welife-os/engine/internal/server"
)

func TempDatabasePath(t testing.TB) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "welife.db")
}

func NewServerConfig(t testing.TB, ollamaURL string) server.Config {
	t.Helper()
	return server.Config{
		Host:         "127.0.0.1",
		Port:         18080,
		DatabasePath: TempDatabasePath(t),
		DatabaseKey:  "welife-phase0-test-key",
		LLMProvider:  "ollama",
		LLMBaseURL:   ollamaURL,
		LLMModel:     "qwen3.5:9b",
	}
}
