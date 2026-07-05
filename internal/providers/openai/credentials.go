package openai

import (
	"cmp"
	"fmt"
)

// EnvLookup resolves an environment variable value.
type EnvLookup func(string) (string, bool)

// Credentials contains OpenAI-compatible provider settings loaded from secure sources.
type Credentials struct {
	APIKey  string
	BaseURL string
	Model   string
}

// LoadCredentials loads OpenAI-compatible provider credentials from environment lookup.
func LoadCredentials(lookup EnvLookup) (*Credentials, error) {
	apiKey, ok := lookup("I18N_MCP_OPENAI_API_KEY")
	if !ok || apiKey == "" {
		return nil, fmt.Errorf("I18N_MCP_OPENAI_API_KEY is required for openai-compatible provider")
	}
	baseURL, _ := lookup("I18N_MCP_OPENAI_BASE_URL")
	baseURL = cmp.Or(baseURL, "https://api.openai.com/v1")
	model, ok := lookup("I18N_MCP_OPENAI_MODEL")
	if !ok || model == "" {
		return nil, fmt.Errorf("I18N_MCP_OPENAI_MODEL is required for openai-compatible provider")
	}
	return new(Credentials{APIKey: apiKey, BaseURL: baseURL, Model: model}), nil
}

// Redact returns a safe display value for secrets.
func Redact(value string) string {
	if value == "" {
		return ""
	}
	return "[redacted]"
}
