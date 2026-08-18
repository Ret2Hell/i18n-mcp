package app

import (
	"net/http"

	"github.com/Ret2Hell/i18n-mcp/internal/providers/openai"
	"github.com/Ret2Hell/i18n-mcp/internal/translate"
)

// BuildProviderRegistry builds the startup provider registry from secure local configuration.
func BuildProviderRegistry(env func(string) (string, bool), httpClient *http.Client) *translate.ProviderRegistry {
	creds, err := openai.LoadCredentials(env)
	if err != nil {
		registry := translate.NewProviderRegistry()
		registry.MarkUnavailable("openai-compatible", err)
		return registry
	}
	provider := openai.NewClient(openai.Options{
		Credentials: creds,
		HTTPClient:  httpClient,
	})
	return translate.NewProviderRegistry(provider)
}
