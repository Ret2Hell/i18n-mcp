package httpserver

import (
	"cmp"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
)

// ProtectedResourceHandler serves RFC 9728 protected resource metadata.
func ProtectedResourceHandler(cfg AuthConfig) http.Handler {
	name := cmp.Or(cfg.ResourceName, "i18n-mcp")
	return auth.ProtectedResourceMetadataHandler(&oauthex.ProtectedResourceMetadata{
		Resource:               cfg.ResourceURL,
		ResourceName:           name,
		AuthorizationServers:   cfg.AuthorizationServers,
		ScopesSupported:        cfg.RequiredScopes,
		BearerMethodsSupported: []string{"header"},
	})
}
