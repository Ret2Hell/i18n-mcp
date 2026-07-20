package httpserver

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/modelcontextprotocol/go-sdk/auth"
)

// TokenVerifierFactory constructs a token verifier from HTTP auth configuration.
type TokenVerifierFactory func(AuthConfig) (auth.TokenVerifier, error)

// DevStaticTokenVerifier constructs a verifier using a token from an environment
// variable. It is intended only for tests and local development.
func DevStaticTokenVerifier(cfg AuthConfig) (auth.TokenVerifier, error) {
	if cfg.DevStaticTokenEnv == "" {
		return nil, errors.New("no token verifier configured")
	}
	want := os.Getenv(cfg.DevStaticTokenEnv)
	if want == "" {
		return nil, fmt.Errorf("%s is empty", cfg.DevStaticTokenEnv)
	}
	return func(_ context.Context, token string, _ *http.Request) (*auth.TokenInfo, error) {
		if subtle.ConstantTimeCompare([]byte(token), []byte(want)) != 1 {
			return nil, auth.ErrInvalidToken
		}
		return &auth.TokenInfo{
			Scopes:     cfg.RequiredScopes,
			Expiration: time.Now().Add(5 * time.Minute),
			UserID:     "dev-static",
		}, nil
	}, nil
}

// ProtectMCPHandler requires a valid bearer token when authentication is enabled.
func ProtectMCPHandler(handler http.Handler, cfg AuthConfig, verifier auth.TokenVerifier) http.Handler {
	if !cfg.Required {
		return handler
	}
	return auth.RequireBearerToken(verifier, &auth.RequireBearerTokenOptions{
		ResourceMetadataURL: cfg.MetadataURL,
		Scopes:              cfg.RequiredScopes,
	})(handler)
}
