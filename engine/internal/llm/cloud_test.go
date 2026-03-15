package llm_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/welife-os/welife-os/engine/internal/llm"
)

func TestCloudClient_Generate(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("unexpected Authorization header: %s", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("unexpected Content-Type: %s", got)
		}

		var req struct {
			Model    string `json:"model"`
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Model != "deepseek-chat" {
			t.Errorf("expected model deepseek-chat, got %s", req.Model)
		}
		if len(req.Messages) != 1 || req.Messages[0].Role != "user" {
			t.Errorf("unexpected messages: %+v", req.Messages)
		}

		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": "Hello from cloud!"}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := llm.NewCloudClient(llm.Config{
		Provider: "openai-compatible",
		BaseURL:  srv.URL,
		APIKey:   "test-key",
		Model:    "deepseek-chat",
		Timeout:  5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewCloudClient: %v", err)
	}

	got, err := client.Generate(context.Background(), "say hello")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if got != "Hello from cloud!" {
		t.Errorf("expected 'Hello from cloud!', got %q", got)
	}
}

func TestCloudClient_Generate_APIError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{"message": "invalid api key", "type": "auth_error"},
		})
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := llm.NewCloudClient(llm.Config{
		BaseURL: srv.URL,
		APIKey:  "bad-key",
		Model:   "test-model",
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewCloudClient: %v", err)
	}

	_, err = client.Generate(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}

func TestCloudClient_Generate_EmptyChoices(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"choices": []any{}})
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := llm.NewCloudClient(llm.Config{
		BaseURL: srv.URL,
		APIKey:  "test-key",
		Model:   "test-model",
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewCloudClient: %v", err)
	}

	_, err = client.Generate(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error for empty choices")
	}
}

func TestCloudClient_Reachable(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]string{{"id": "test-model"}},
		})
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := llm.NewCloudClient(llm.Config{
		BaseURL: srv.URL,
		APIKey:  "test-key",
		Model:   "test-model",
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewCloudClient: %v", err)
	}

	ok, err := client.Reachable(context.Background())
	if err != nil {
		t.Fatalf("Reachable: %v", err)
	}
	if !ok {
		t.Error("expected reachable=true")
	}
}

func TestCloudClient_Reachable_Failure(t *testing.T) {
	client, err := llm.NewCloudClient(llm.Config{
		BaseURL: "http://127.0.0.1:1",
		APIKey:  "test-key",
		Model:   "test-model",
		Timeout: 1 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewCloudClient: %v", err)
	}

	ok, _ := client.Reachable(context.Background())
	if ok {
		t.Error("expected reachable=false for unreachable server")
	}
}

func TestCloudClient_Status(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]string{{"id": "test-model"}},
		})
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := llm.NewCloudClient(llm.Config{
		BaseURL: srv.URL,
		APIKey:  "test-key",
		Model:   "deepseek-chat",
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewCloudClient: %v", err)
	}

	status := client.Status(context.Background())
	if status.Provider != "openai-compatible" {
		t.Errorf("expected provider 'openai-compatible', got %q", status.Provider)
	}
	if !status.Reachable {
		t.Error("expected reachable=true")
	}
	if status.Model != "deepseek-chat" {
		t.Errorf("expected model 'deepseek-chat', got %q", status.Model)
	}
	if status.BaseURL != srv.URL {
		t.Errorf("expected BaseURL %q, got %q", srv.URL, status.BaseURL)
	}
}

func TestNewCloudClient_Validation(t *testing.T) {
	tests := []struct {
		name string
		cfg  llm.Config
	}{
		{name: "missing base URL", cfg: llm.Config{APIKey: "k", Model: "m"}},
		{name: "missing API key", cfg: llm.Config{BaseURL: "http://x", Model: "m"}},
		{name: "missing model", cfg: llm.Config{BaseURL: "http://x", APIKey: "k"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := llm.NewCloudClient(tc.cfg)
			if err == nil {
				t.Error("expected validation error")
			}
		})
	}
}

func TestNewClient_Factory(t *testing.T) {
	t.Run("unsupported provider", func(t *testing.T) {
		_, err := llm.NewClient(llm.Config{Provider: "unknown"})
		if err == nil {
			t.Error("expected error for unsupported provider")
		}
	})

	t.Run("openai-compatible creates CloudClient", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		client, err := llm.NewClient(llm.Config{
			Provider: "openai-compatible",
			BaseURL:  srv.URL,
			APIKey:   "test-key",
			Model:    "test-model",
		})
		if err != nil {
			t.Fatalf("NewClient: %v", err)
		}
		// Verify it satisfies the interface.
		var _ llm.LLMClient = client
	})
}
