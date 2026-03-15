package llm

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/ollama/ollama/api"
)

// ErrEmbeddingUnavailable is returned when no embedding model is configured.
var ErrEmbeddingUnavailable = errors.New("embedding unavailable: no embedding model configured")

// LLMClient is the interface that all LLM backends must satisfy.
type LLMClient interface {
	// Generate sends a prompt to the LLM and returns the complete response text.
	Generate(ctx context.Context, prompt string) (string, error)

	// Embed returns a vector embedding for the given text.
	// Returns ErrEmbeddingUnavailable when no embedding model is configured.
	Embed(ctx context.Context, text string) ([]float32, error)

	// Reachable checks whether the LLM service is reachable.
	Reachable(ctx context.Context) (bool, error)

	// Status returns connection and provider status information.
	Status(ctx context.Context) StatusInfo
}

// Config holds configuration for creating an LLM client.
type Config struct {
	Provider       string        // "ollama" (default) | "openai-compatible"
	BaseURL        string
	Model          string
	EmbeddingModel string        // e.g. "nomic-embed-text"; empty disables embedding
	Timeout        time.Duration
	APIKey         string // Cloud LLM only
}

// StatusInfo holds connection and provider status returned by LLMClient.Status.
type StatusInfo struct {
	Provider  string
	Reachable bool
	BaseURL   string
	Model     string
}

// NewClient creates an LLMClient based on the Config.Provider field.
// When Provider is empty or "ollama", an Ollama client is returned.
// When Provider is "openai-compatible", a cloud client is returned.
func NewClient(cfg Config) (LLMClient, error) {
	switch cfg.Provider {
	case "", "ollama":
		return New(cfg)
	case "openai-compatible":
		return NewCloudClient(cfg)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %q", cfg.Provider)
	}
}

// Client is the Ollama-backed LLM client.
type Client struct {
	baseURL    string
	model      string
	embedModel string
	client     *api.Client
}

// New creates an Ollama LLM client. Prefer NewClient for provider-agnostic creation.
func New(cfg Config) (*Client, error) {
	parsedURL, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{Timeout: cfg.Timeout}
	return &Client{
		baseURL:    cfg.BaseURL,
		model:      cfg.Model,
		embedModel: cfg.EmbeddingModel,
		client:     api.NewClient(parsedURL, httpClient),
	}, nil
}

func (c *Client) Reachable(ctx context.Context) (bool, error) {
	_, err := c.client.List(ctx)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (c *Client) Status(ctx context.Context) StatusInfo {
	reachable, err := c.Reachable(ctx)
	if err != nil {
		reachable = false
	}

	return StatusInfo{
		Provider:  "ollama",
		Reachable: reachable,
		BaseURL:   c.baseURL,
		Model:     c.model,
	}
}

// Generate sends a prompt to the LLM and returns the complete response text.
func (c *Client) Generate(ctx context.Context, prompt string) (string, error) {
	req := &api.GenerateRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: new(bool), // false = non-streaming
	}

	var response string
	err := c.client.Generate(ctx, req, func(resp api.GenerateResponse) error {
		response += resp.Response
		return nil
	})
	if err != nil {
		return "", err
	}
	return response, nil
}

// Embed returns a vector embedding for the given text using the configured
// embedding model (e.g. nomic-embed-text). Returns ErrEmbeddingUnavailable
// when no embedding model is configured.
func (c *Client) Embed(ctx context.Context, text string) ([]float32, error) {
	if c.embedModel == "" {
		return nil, ErrEmbeddingUnavailable
	}

	resp, err := c.client.Embed(ctx, &api.EmbedRequest{
		Model: c.embedModel,
		Input: text,
	})
	if err != nil {
		return nil, fmt.Errorf("ollama embed: %w", err)
	}
	if len(resp.Embeddings) == 0 {
		return nil, fmt.Errorf("ollama embed: empty embeddings response")
	}
	return resp.Embeddings[0], nil
}
