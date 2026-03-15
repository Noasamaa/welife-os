package llm_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/welife-os/welife-os/engine/internal/llm"
)

func TestClientReachable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models":[]}`))
	}))
	defer server.Close()

	client, err := llm.New(llm.Config{
		BaseURL: server.URL,
		Model:   "qwen3.5:9b",
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	reachable, err := client.Reachable(context.Background())
	if err != nil {
		t.Fatalf("reachable: %v", err)
	}
	if !reachable {
		t.Fatal("expected ollama to be reachable")
	}
}

func TestClientStatusHandlesUnreachable(t *testing.T) {
	client, err := llm.New(llm.Config{
		BaseURL: "http://127.0.0.1:65530",
		Model:   "qwen3.5:9b",
		Timeout: 50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	status := client.Status(context.Background())
	if status.Reachable {
		t.Fatal("status should report unreachable")
	}
	if status.Provider != "ollama" {
		t.Fatalf("unexpected provider: %s", status.Provider)
	}
}
