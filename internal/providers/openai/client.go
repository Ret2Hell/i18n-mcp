package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Ret2Hell/i18n-mcp/internal/translate"
)

const maxResponseBodyBytes = 4 << 20

// Options configures an OpenAI-compatible translation provider.
type Options struct {
	Credentials *Credentials
	HTTPClient  *http.Client
}

// Client implements an OpenAI-compatible translation provider.
type Client struct {
	creds *Credentials
	http  *http.Client
}

// NewClient creates an OpenAI-compatible translation provider.
func NewClient(opts Options) *Client {
	hc := opts.HTTPClient
	if hc == nil {
		hc = new(http.Client{Timeout: 60 * time.Second})
	}
	return new(Client{creds: opts.Credentials, http: hc})
}

// Name returns the registry name for the OpenAI-compatible provider.
func (c *Client) Name() string { return "openai-compatible" }

// Generate generates translation proposals with the configured OpenAI-compatible provider.
func (c *Client) Generate(ctx context.Context, req translate.ProviderRequest) (*translate.ProviderResponse, error) {
	if c.creds == nil {
		return nil, fmt.Errorf("openai-compatible provider is missing credentials")
	}
	prompt, err := buildPrompt(req)
	if err != nil {
		return nil, err
	}
	body := chatRequest{
		Model:       c.creds.Model,
		Temperature: 0.2,
		Messages: []chatMessage{
			{Role: "system", Content: "You generate JSON translation proposals. Return only valid JSON."},
			{Role: "user", Content: prompt},
		},
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("encoding provider request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.creds.BaseURL, "/")+"/chat/completions", bytes.NewReader(encoded))
	if err != nil {
		return nil, fmt.Errorf("creating provider request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.creds.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	started := time.Now().UTC()
	httpRes, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("sending provider request: %w", err)
	}
	defer httpRes.Body.Close()
	if httpRes.StatusCode < 200 || httpRes.StatusCode >= 300 {
		return nil, fmt.Errorf("provider request failed with status %d", httpRes.StatusCode)
	}
	responseBody, err := io.ReadAll(io.LimitReader(httpRes.Body, maxResponseBodyBytes+1))
	if err != nil {
		return nil, fmt.Errorf("reading provider response: %w", err)
	}
	if len(responseBody) > maxResponseBodyBytes {
		return nil, fmt.Errorf("provider response exceeds %d bytes", maxResponseBodyBytes)
	}
	var decoded chatResponse
	if err := json.Unmarshal(responseBody, &decoded); err != nil {
		return nil, fmt.Errorf("decoding provider response: %w", err)
	}
	if len(decoded.Choices) == 0 {
		return nil, fmt.Errorf("provider returned no choices")
	}
	var proposals []translate.ProposedTranslation
	if err := json.Unmarshal([]byte(decoded.Choices[0].Message.Content), &proposals); err != nil {
		return nil, fmt.Errorf("decoding provider proposals: %w", err)
	}
	return new(translate.ProviderResponse{
		Proposals: proposals,
		Usage: translate.ProviderUsage{
			InputTokens:  decoded.Usage.PromptTokens,
			OutputTokens: decoded.Usage.CompletionTokens,
			TotalTokens:  decoded.Usage.TotalTokens,
		},
		StartedAt: started,
		EndedAt:   time.Now().UTC(),
		Metadata:  map[string]any{"provider": c.Name(), "model": c.creds.Model},
	}), nil
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}
