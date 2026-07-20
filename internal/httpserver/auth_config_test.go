package httpserver

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAuthConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		addr    string
		config  AuthConfig
		wantErr string
	}{
		{
			name:   "localhost without auth",
			addr:   "127.0.0.1:7339",
			config: DefaultAuthConfig(),
		},
		{
			name:    "non-localhost without auth",
			addr:    "0.0.0.0:7339",
			config:  DefaultAuthConfig(),
			wantErr: "HTTP auth is required when binding to non-localhost address",
		},
		{
			name: "auth without resource URL",
			addr: "127.0.0.1:7339",
			config: AuthConfig{
				Required:     true,
				MetadataPath: "/.well-known/oauth-protected-resource",
			},
			wantErr: "auth resource URL is required when auth is enabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate(tt.addr)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestDefaultAuthConfigScopes(t *testing.T) {
	require.Equal(t, []string{"i18n:read", "i18n:write"}, DefaultAuthConfig().RequiredScopes)
}
