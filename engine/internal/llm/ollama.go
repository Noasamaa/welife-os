package llm

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/ollama/ollama/api"
)

// LLMClient is the interface that all LLM backends must satisfy.
type LLMClient interface {
	// Generate sends a prompt to the LLM and returns the complete response text.
	Generate(ctx context.Context, prompt string) (string, error)

	// Reachable checks whether the LLM service is reachable.
	Reachable(ctx context.Context) (bool, error)

	// Status returns connection and provider status information.
	Status(ctx context.Context) StatusInfo
}

// Config holds configuration for creating an LLM client.
type Config struct {
	Provider string        // "ollama" (default) | "openai-compatible"
	BaseURL  string
	Model    string
	Timeout  time.Duration
	APIKey   string // Cloud LLM only
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
	baseURL string
	model   string
	client  *api.Client
}

// New creates an Ollama LLM client. Prefer NewClient for provider-agnostic creation.
func New(cfg Config) (*Client, error) {
	parsedURL, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{Timeout: cfg.Timeout}
	return &Client{
		baseURL: cfg.BaseURL,
		model:   cfg.Model,
		client:  api.NewClient(parsedURL, httpClient),
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
