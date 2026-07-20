package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/stretchr/testify/require"
)

func TestProtectMCPHandler(t *testing.T) {
	cfg := DefaultAuthConfig()
	cfg.Required = true
	cfg.MetadataURL = "https://example.com/.well-known/oauth-protected-resource"
	tests := []struct {
		name       string
		authHeader string
		verifier   auth.TokenVerifier
		wantStatus int
	}{
		{
			name:       "missing token",
			verifier:   validTokenVerifier(cfg.RequiredScopes),
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid token",
			authHeader: "Bearer invalid",
			verifier: func(context.Context, string, *http.Request) (*auth.TokenInfo, error) {
				return nil, auth.ErrInvalidToken
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "missing scope",
			authHeader: "Bearer valid",
			verifier:   validTokenVerifier([]string{"i18n:read"}),
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "valid token",
			authHeader: "Bearer valid",
			verifier:   validTokenVerifier(cfg.RequiredScopes),
			wantStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := ProtectMCPHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}), cfg, tt.verifier)
			req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
			req.Header.Set("Authorization", tt.authHeader)
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, req)

			require.Equal(t, tt.wantStatus, response.Code)
			if tt.wantStatus == http.StatusUnauthorized || tt.wantStatus == http.StatusForbidden {
				require.Contains(t, response.Header().Get("WWW-Authenticate"), `resource_metadata="`+cfg.MetadataURL+`"`)
				require.Contains(t, response.Header().Get("WWW-Authenticate"), `scope="i18n:read i18n:write"`)
			}
		})
	}
}

func TestDevStaticTokenVerifier(t *testing.T) {
	const envName = "I18N_MCP_TEST_STATIC_TOKEN"
	t.Setenv(envName, "secret")
	cfg := DefaultAuthConfig()
	cfg.DevStaticTokenEnv = envName
	verifier, err := DevStaticTokenVerifier(cfg)
	require.NoError(t, err)

	info, err := verifier(t.Context(), "secret", httptest.NewRequest(http.MethodPost, "/mcp", nil))
	require.NoError(t, err)
	require.Equal(t, cfg.RequiredScopes, info.Scopes)
	require.Equal(t, "dev-static", info.UserID)

	_, err = verifier(t.Context(), "wrong", httptest.NewRequest(http.MethodPost, "/mcp", nil))
	require.ErrorIs(t, err, auth.ErrInvalidToken)
}

func validTokenVerifier(scopes []string) auth.TokenVerifier {
	return func(context.Context, string, *http.Request) (*auth.TokenInfo, error) {
		return &auth.TokenInfo{Scopes: scopes, Expiration: time.Now().Add(time.Hour)}, nil
	}
}
