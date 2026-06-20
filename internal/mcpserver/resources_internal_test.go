package mcpserver

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseLocaleNamespaceURI(t *testing.T) {
	tests := []struct {
		name          string
		uri           string
		wantLocale    string
		wantNamespace string
		wantOK        bool
	}{
		{name: "simple", uri: "i18n://locales/en/common", wantLocale: "en", wantNamespace: "common", wantOK: true},
		{name: "escaped", uri: "i18n://locales/pt-BR/admin%20panel", wantLocale: "pt-BR", wantNamespace: "admin panel", wantOK: true},
		{name: "malformed uri", uri: "%zz", wantOK: false},
		{name: "wrong scheme", uri: "http://locales/en/common", wantOK: false},
		{name: "wrong host", uri: "i18n://config/en/common", wantOK: false},
		{name: "missing namespace", uri: "i18n://locales/en", wantOK: false},
		{name: "extra path segment", uri: "i18n://locales/en/common/extra", wantOK: false},
		{name: "bad locale escape", uri: "i18n://locales/%zz/common", wantOK: false},
		{name: "bad namespace escape", uri: "i18n://locales/en/%zz", wantOK: false},
		{name: "empty locale", uri: "i18n://locales//common", wantOK: false},
		{name: "empty namespace", uri: "i18n://locales/en/", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			localeCode, namespace, ok := parseLocaleNamespaceURI(tt.uri)
			require.Equal(t, tt.wantOK, ok)
			require.Equal(t, tt.wantLocale, localeCode)
			require.Equal(t, tt.wantNamespace, namespace)
		})
	}
}
