package httpserver

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// AuthConfig configures bearer authentication and protected resource metadata.
type AuthConfig struct {
	Required             bool     `json:"required"`
	ResourceURL          string   `json:"resourceUrl,omitempty"`
	ResourceName         string   `json:"resourceName,omitempty"`
	MetadataPath         string   `json:"metadataPath,omitempty"`
	MetadataURL          string   `json:"metadataUrl,omitempty"`
	RequiredScopes       []string `json:"requiredScopes,omitzero"`
	AuthorizationServers []string `json:"authorizationServers,omitzero"`
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
	if c.MetadataPath == "" || !strings.HasPrefix(c.MetadataPath, "/") {
		return fmt.Errorf("auth metadata path must be an absolute HTTP path")
	}
	if c.ResourceURL != "" {
		if err := validateAuthURL("auth resource URL", c.ResourceURL); err != nil {
			return err
		}
	}
	if c.MetadataURL != "" {
		if err := validateAuthURL("auth metadata URL", c.MetadataURL); err != nil {
			return err
		}
	}
	for _, issuer := range c.AuthorizationServers {
		if err := validateAuthURL("authorization server issuer", issuer); err != nil {
			return err
		}
	}
	return nil
}

func validateAuthURL(name string, raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || !parsed.IsAbs() || parsed.Host == "" {
		return fmt.Errorf("%s must be an absolute URL", name)
	}
	if parsed.User != nil || parsed.Fragment != "" || parsed.RawQuery != "" {
		return fmt.Errorf("%s must not contain user info, query, or fragment", name)
	}
	switch parsed.Scheme {
	case "https":
		return nil
	case "http":
		host := parsed.Hostname()
		if strings.EqualFold(host, "localhost") {
			return nil
		}
		if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
			return nil
		}
	}
	return fmt.Errorf("%s must use HTTPS except for loopback development URLs", name)
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
