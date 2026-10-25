package translate

import (
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/Ret2Hell/i18n-mcp/internal/locale"
	"github.com/stretchr/testify/require"
)

func TestBuildPathFromPattern(t *testing.T) {
	for _, tc := range []struct {
		name, pattern, namespace, defaultNamespace, want string
		ok                                               bool
	}{
		{name: "locale required", pattern: "messages/common.json"},
		{name: "namespace required", pattern: "messages/{locale}/{namespace}.json"},
		{name: "namespaced", pattern: "messages/{locale}/{namespace}.json", namespace: "admin", defaultNamespace: "common", want: "messages/fr/admin.json", ok: true},
		{name: "default namespace", pattern: "messages/{locale}.json", namespace: "common", defaultNamespace: "common", want: "messages/fr.json", ok: true},
		{name: "namespace mismatch", pattern: "messages/{locale}.json", namespace: "admin", defaultNamespace: "common"},
		{name: "empty namespace", pattern: "messages/{locale}.json", defaultNamespace: "common", want: "messages/fr.json", ok: true},
		{name: "no default configured", pattern: "messages/{locale}.json", namespace: "admin", want: "messages/fr.json", ok: true},
		{name: "repeated placeholders", pattern: "{locale}/{namespace}/{locale}-{namespace}.json", namespace: "common", want: "fr/common/fr-common.json", ok: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := buildPathFromPattern(tc.pattern, "fr", tc.namespace, tc.defaultNamespace)
			require.Equal(t, tc.ok, ok)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestTargetPathForPrefersInventoryAndFallsBackToMatchingPattern(t *testing.T) {
	s := &Service{}
	cfg := config.Resolved{DefaultNamespace: "common", LocaleFiles: []string{"flat/{locale}.json", "locales/{locale}/{namespace}.json"}}
	inv := locale.Inventory{Files: []locale.FileSummary{{Locale: "fr", Namespace: "admin", Path: "existing/fr/admin.json"}}}
	path, err := s.targetPathFor(cfg, inv, "fr", "admin")
	require.NoError(t, err)
	require.Equal(t, "existing/fr/admin.json", path)
	path, err = s.targetPathFor(cfg, inv, "de", "admin")
	require.NoError(t, err)
	require.Equal(t, "locales/de/admin.json", path)
	cfg.LocaleFiles = []string{"flat/{locale}.json"}
	_, err = s.targetPathFor(cfg, inv, "de", "admin")
	require.ErrorContains(t, err, "no locale file path for de/admin")
}
