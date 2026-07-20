package httpserver

import (
	"fmt"
	"net"
	"strings"
)

// AuthConfig configures bearer authentication and protected resource metadata.
type AuthConfig struct {
	Required             bool     `json:"required"`
	ResourceURL          string   `json:"resourceUrl,omitempty"`
	ResourceName         string   `json:"resourceName,omitempty"`
	MetadataPath         string   `json:"metadataPath,omitempty"`
	MetadataURL          string   `json:"metadataUrl,omitempty"`
	RequiredScopes       []string `json:"requiredScopes,omitempty"`
	AuthorizationServers []string `json:"authorizationServers,omitempty"`
	DevStaticTokenEnv    string   `json:"-"`
}

// DefaultAuthConfig returns the default HTTP authentication configuration.
func DefaultAuthConfig() AuthConfig {
	return AuthConfig{
		Required:       false,
		ResourceName:   "i18n-mcp",
		MetadataPath:   "/.well-known/oauth-protected-resource",
		RequiredScopes: []string{"i18n:read", "i18n:write"},
	}
}

// Validate checks that the authentication configuration is safe for addr.
func (c AuthConfig) Validate(addr string) error {
	if bindsNonLocalhost(addr) && !c.Required {
		return fmt.Errorf("HTTP auth is required when binding to non-localhost address %q", addr)
	}
	if c.Required && c.ResourceURL == "" {
		return fmt.Errorf("auth resource URL is required when auth is enabled")
	}
	if c.MetadataPath == "" {
		return fmt.Errorf("auth metadata path is required")
	}
	return nil
}

func bindsNonLocalhost(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return true
	}
	if strings.EqualFold(host, "localhost") {
		return false
	}
	ip := net.ParseIP(host)
	return ip == nil || !ip.IsLoopback()
}
