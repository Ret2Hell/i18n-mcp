package httpserver

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Ret2Hell/i18n-mcp/internal/security"
	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/stretchr/testify/require"
)

func TestHTTPAuthRejectsUnauthenticatedMCP(t *testing.T) {
	cfg := testServerAuthConfig()
	server := httptest.NewServer(newTestHTTPHandler(t, cfg, validHTTPTestVerifier(cfg.RequiredScopes, "subject-a"), nil))
	t.Cleanup(server.Close)

	response := sendMCPRequest(t, server.URL, "")
	defer response.Body.Close()
	require.Equal(t, http.StatusUnauthorized, response.StatusCode)
	require.Contains(t, response.Header.Get("WWW-Authenticate"), "resource_metadata")
}

func TestHTTPAuthRejectsInvalidToken(t *testing.T) {
	cfg := testServerAuthConfig()
	verifier := func(context.Context, string, *http.Request) (*auth.TokenInfo, error) {
		return nil, auth.ErrInvalidToken
	}
	server := httptest.NewServer(newTestHTTPHandler(t, cfg, verifier, nil))
	t.Cleanup(server.Close)

	response := sendMCPRequest(t, server.URL, "invalid")
	defer response.Body.Close()
	require.Equal(t, http.StatusUnauthorized, response.StatusCode)
}

func TestHTTPAuthRejectsInsufficientScope(t *testing.T) {
	cfg := testServerAuthConfig()
	server := httptest.NewServer(newTestHTTPHandler(t, cfg, validHTTPTestVerifier(nil, "subject-a"), nil))
	t.Cleanup(server.Close)

	response := sendMCPRequest(t, server.URL, "test-token")
	defer response.Body.Close()
	require.Equal(t, http.StatusForbidden, response.StatusCode)
}

func TestAuthenticatedMCPRequestReachesHandlerWithSubject(t *testing.T) {
	cfg := testServerAuthConfig()
	subject := make(chan string, 1)
	downstream := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		subject <- security.SubjectFromContext(req.Context())
		w.WriteHeader(http.StatusNoContent)
	})
	server := httptest.NewServer(newTestHTTPHandler(t, cfg, validHTTPTestVerifier(cfg.RequiredScopes, "subject-a"), downstream))
	t.Cleanup(server.Close)

	response := sendMCPRequest(t, server.URL, "test-token")
	defer response.Body.Close()
	require.Equal(t, http.StatusNoContent, response.StatusCode)
	require.Equal(t, "subject-a", <-subject)
}

func TestProtectedResourceMetadataIsPublic(t *testing.T) {
	cfg := testServerAuthConfig()
	server := httptest.NewServer(newTestHTTPHandler(t, cfg, validHTTPTestVerifier(cfg.RequiredScopes, "subject-a"), nil))
	t.Cleanup(server.Close)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL+cfg.MetadataPath, nil)
	require.NoError(t, err)
	response, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Contains(t, string(body), `"resource":"https://example.test/mcp"`)
	require.Contains(t, string(body), `"resource_name":"i18n-mcp"`)
	require.Contains(t, string(body), `"authorization_servers":["https://issuer.example.test"]`)
	require.Contains(t, string(body), `"i18n:read"`)
	require.Contains(t, string(body), `"bearer_methods_supported":["header"]`)
}

func newTestHTTPHandler(t *testing.T, cfg AuthConfig, verifier auth.TokenVerifier, downstream http.Handler) http.Handler {
	t.Helper()
	if downstream == nil {
		downstream = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})
	}
	mux := http.NewServeMux()
	mux.Handle(cfg.MetadataPath, ProtectedResourceHandler(cfg))
	mux.Handle("/mcp", ProtectMCPHandler(downstream, cfg, verifier))
	return mux
}

func testServerAuthConfig() AuthConfig {
	cfg := DefaultAuthConfig()
	cfg.Required = true
	cfg.ResourceURL = "https://example.test/mcp"
	cfg.MetadataURL = "https://example.test/.well-known/oauth-protected-resource"
	cfg.AuthorizationServers = []string{"https://issuer.example.test"}
	cfg.RequiredScopes = []string{"i18n:read"}
	return cfg
}

func validHTTPTestVerifier(scopes []string, userID string) auth.TokenVerifier {
	return func(context.Context, string, *http.Request) (*auth.TokenInfo, error) {
		return &auth.TokenInfo{
			Scopes:     scopes,
			Expiration: time.Now().Add(time.Minute),
			UserID:     userID,
		}, nil
	}
}

func sendMCPRequest(t *testing.T, baseURL, token string) *http.Response {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, baseURL+"/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return response
}
