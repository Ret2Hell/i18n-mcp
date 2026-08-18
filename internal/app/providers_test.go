package app

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildProviderRegistryPreservesCredentialError(t *testing.T) {
	registry := BuildProviderRegistry(func(string) (string, bool) { return "", false }, http.DefaultClient)

	provider, err := registry.Get("openai-compatible")

	require.Nil(t, provider)
	require.ErrorContains(t, err, "I18N_MCP_OPENAI_API_KEY is required")
}

func TestBuildProviderRegistryConfiguresOpenAICompatibleProvider(t *testing.T) {
	values := map[string]string{
		"I18N_MCP_OPENAI_API_KEY": "secret",
		"I18N_MCP_OPENAI_MODEL":   "test-model",
	}
	registry := BuildProviderRegistry(func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}, http.DefaultClient)

	provider, err := registry.Get("openai-compatible")

	require.NoError(t, err)
	require.Equal(t, "openai-compatible", provider.Name())
}
