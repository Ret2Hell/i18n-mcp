package openai

import (
	"cmp"
	"context"
	"fmt"
	"net/http"

	"github.com/Ret2Hell/i18n-mcp/internal/translate"
)

// Options configures an OpenAI-compatible translation provider.
type Options struct {
	Credentials *Credentials
	HTTPClient  *http.Client
}

type Client struct {
	credentials *Credentials
	httpClient  *http.Client
}

// NewClient creates an OpenAI-compatible translation provider.
func NewClient(opts Options) *Client {
	client := cmp.Or(opts.HTTPClient, http.DefaultClient)
	return new(Client{credentials: opts.Credentials, httpClient: client})
}

// Name returns the registry name for the OpenAI-compatible provider.
func (c *Client) Name() string {
	return "openai-compatible"
}

// Generate generates translation proposals with the configured OpenAI-compatible provider.
func (c *Client) Generate(ctx context.Context, req translate.ProviderRequest) (*translate.ProviderResponse, error) {
	_ = ctx
	_ = req
	_ = c.httpClient
	if c.credentials == nil {
		return nil, fmt.Errorf("openai-compatible provider credentials are not configured")
	}
	return nil, fmt.Errorf("openai-compatible provider generation is not implemented")
}
