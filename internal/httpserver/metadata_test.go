package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/oauthex"
	"github.com/stretchr/testify/require"
)

func TestProtectedResourceHandler(t *testing.T) {
	cfg := DefaultAuthConfig()
	cfg.ResourceURL = "https://example.com/mcp"
	cfg.AuthorizationServers = []string{"https://issuer.example.com"}

	response := httptest.NewRecorder()
	ProtectedResourceHandler(cfg).ServeHTTP(response, httptest.NewRequest(http.MethodGet, cfg.MetadataPath, nil))

	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "application/json", response.Header().Get("Content-Type"))
	var metadata oauthex.ProtectedResourceMetadata
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &metadata))
	require.Equal(t, cfg.ResourceURL, metadata.Resource)
	require.Equal(t, cfg.ResourceName, metadata.ResourceName)
	require.Equal(t, cfg.AuthorizationServers, metadata.AuthorizationServers)
	require.Equal(t, cfg.RequiredScopes, metadata.ScopesSupported)
	require.Equal(t, []string{"header"}, metadata.BearerMethodsSupported)
}

func TestProtectedResourceHandlerMethods(t *testing.T) {
	handler := ProtectedResourceHandler(DefaultAuthConfig())

	optionsResponse := httptest.NewRecorder()
	handler.ServeHTTP(optionsResponse, httptest.NewRequest(http.MethodOptions, "/metadata", nil))
	require.Equal(t, http.StatusNoContent, optionsResponse.Code)

	postResponse := httptest.NewRecorder()
	handler.ServeHTTP(postResponse, httptest.NewRequest(http.MethodPost, "/metadata", nil))
	require.Equal(t, http.StatusMethodNotAllowed, postResponse.Code)
}

func TestInferMetadataURL(t *testing.T) {
	cfg := Config{Auth: DefaultAuthConfig()}
	cfg.Auth.ResourceURL = "https://example.com/mcp/"
	require.Equal(t, "https://example.com/mcp/.well-known/oauth-protected-resource", inferMetadataURL(cfg))

	cfg.Auth.MetadataURL = "https://metadata.example.com/resource"
	require.Equal(t, cfg.Auth.MetadataURL, inferMetadataURL(cfg))
}
