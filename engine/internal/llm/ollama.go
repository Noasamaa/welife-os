package llm

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/ollama/ollama/api"
)

type Config struct {
	BaseURL string
	Model   string
	Timeout time.Duration
}

type StatusInfo struct {
	Provider  string
	Reachable bool
	BaseURL   string
	Model     string
}

type Client struct {
	baseURL string
	model   string
	client  *api.Client
}

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
