package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// CloudClient implements LLMClient using the OpenAI-compatible chat completions API.
// It works with DeepSeek, Qwen (通义千问), OpenAI, and any compatible provider.
type CloudClient struct {
	baseURL    string
	apiKey     string
	model      string
	embedModel string
	httpClient *http.Client
}

// NewCloudClient creates a cloud LLM client. Prefer NewClient for provider-agnostic creation.
func NewCloudClient(cfg Config) (*CloudClient, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("cloud LLM: base URL is required")
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("cloud LLM: API key is required")
	}
	if cfg.Model == "" {
		return nil, fmt.Errorf("cloud LLM: model name is required")
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 120 * time.Second
	}

	baseURL := strings.TrimRight(cfg.BaseURL, "/")

	return &CloudClient{
		baseURL:    baseURL,
		apiKey:     cfg.APIKey,
		model:      cfg.Model,
		embedModel: cfg.EmbeddingModel,
		httpClient: &http.Client{Timeout: timeout},
	}, nil
}

// chatRequest is the request body for /v1/chat/completions.
type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatResponse is the response body from /v1/chat/completions.
type chatResponse struct {
	Choices []chatChoice `json:"choices"`
	Error   *apiError    `json:"error,omitempty"`
}

type chatChoice struct {
	Message chatMessage `json:"message"`
}

type apiError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

// Generate sends a prompt to the cloud LLM and returns the assistant's reply.
func (c *CloudClient) Generate(ctx context.Context, prompt string) (string, error) {
	reqBody := chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "user", Content: prompt},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("cloud LLM: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("cloud LLM: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("cloud LLM: request failed: %w", err)
	}
	defer resp.Body.Close()

	// Limit response body to 10 MB to prevent DoS from oversized responses.
	const maxResponseSize = 10 << 20
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return "", fmt.Errorf("cloud LLM: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("cloud LLM: HTTP %d: %s", resp.StatusCode, truncate(string(respBody), 200))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("cloud LLM: parse response: %w", err)
	}

	if chatResp.Error != nil {
		return "", fmt.Errorf("cloud LLM: API error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("cloud LLM: no choices in response")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// Reachable checks connectivity by calling /v1/models.
func (c *CloudClient) Reachable(ctx context.Context) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/models", nil)
	if err != nil {
		return false, fmt.Errorf("cloud LLM: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("cloud LLM: HTTP %d", resp.StatusCode)
	}
	return true, nil
}

// Status returns the connection status for the cloud LLM provider.
func (c *CloudClient) Status(ctx context.Context) StatusInfo {
	reachable, err := c.Reachable(ctx)
	if err != nil {
		reachable = false
	}

	return StatusInfo{
		Provider:  "openai-compatible",
		Reachable: reachable,
		BaseURL:   c.baseURL,
		Model:     c.model,
	}
}

// truncate shortens a string to maxLen, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// embeddingRequest is the request body for /v1/embeddings.
type embeddingRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

// embeddingResponse is the response body from /v1/embeddings.
type embeddingResponse struct {
	Data  []embeddingData `json:"data"`
	Error *apiError       `json:"error,omitempty"`
}

type embeddingData struct {
	Embedding []float32 `json:"embedding"`
}

// Embed returns a vector embedding using the OpenAI-compatible /v1/embeddings endpoint.
func (c *CloudClient) Embed(ctx context.Context, text string) ([]float32, error) {
	if c.embedModel == "" {
		return nil, ErrEmbeddingUnavailable
	}

	reqBody := embeddingRequest{
		Model: c.embedModel,
		Input: text,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("cloud embed: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("cloud embed: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cloud embed: request failed: %w", err)
	}
	defer resp.Body.Close()

	// Limit response body to 50 MB to prevent DoS from oversized responses.
	const maxEmbedResponseSize = 50 << 20
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxEmbedResponseSize))
	if err != nil {
		return nil, fmt.Errorf("cloud embed: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cloud embed: HTTP %d: %s", resp.StatusCode, truncate(string(respBody), 200))
	}

	var embedResp embeddingResponse
	if err := json.Unmarshal(respBody, &embedResp); err != nil {
		return nil, fmt.Errorf("cloud embed: parse response: %w", err)
	}

	if embedResp.Error != nil {
		return nil, fmt.Errorf("cloud embed: API error: %s", embedResp.Error.Message)
	}

	if len(embedResp.Data) == 0 || len(embedResp.Data[0].Embedding) == 0 {
		return nil, fmt.Errorf("cloud embed: empty embedding in response")
	}

	return embedResp.Data[0].Embedding, nil
}
