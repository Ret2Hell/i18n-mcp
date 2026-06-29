package mcpserver

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseLocaleNamespaceURIRejectsInvalidURIs(t *testing.T) {
	tests := []struct {
		name string
		uri  string
	}{
		{name: "invalid URL escape", uri: "i18n://locales/%zz/common"},
		{name: "wrong scheme", uri: "https://locales/en/common"},
		{name: "wrong host", uri: "i18n://analysis/en/common"},
		{name: "missing namespace", uri: "i18n://locales/en"},
		{name: "extra segment", uri: "i18n://locales/en/common/extra"},
		{name: "empty locale", uri: "i18n://locales//common"},
		{name: "empty namespace", uri: "i18n://locales/en/"},
		{name: "bad namespace escape", uri: "i18n://locales/en/%zz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			locale, namespace, ok := parseLocaleNamespaceURI(tt.uri)

			require.False(t, ok)
			require.Empty(t, locale)
			require.Empty(t, namespace)
		})
	}
}

func TestParseLocaleNamespaceURIAcceptsEscapedParts(t *testing.T) {
	locale, namespace, ok := parseLocaleNamespaceURI("i18n://locales/pt-BR/common%20messages")

	require.True(t, ok)
	require.Equal(t, "pt-BR", locale)
	require.Equal(t, "common messages", namespace)
}
