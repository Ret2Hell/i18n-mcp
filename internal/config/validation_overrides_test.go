package config

import (
	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestValidationOverridePrecedenceAndExplicitFalse(t *testing.T) {
	off := false
	policy := ValidationConfig{MessageFormat: "icu", Printf: true, Overrides: []ValidationOverride{
		{Key: "legal.title", MessageFormat: "i18next"},
		{KeyPrefix: "legal.", MessageFormat: "raw"},
		{Namespace: "common", Printf: &off},
		{Namespace: "common", KeyPrefix: "legal.", MessageFormat: "interpolation"},
		{KeyPrefix: "legal.deep.", MessageFormat: "raw"},
	}}
	r := policy.ForKey("common", "legal.title")
	require.Equal(t, "i18next", r.MessageFormat)
	require.False(t, r.Printf)
	require.Equal(t, "interpolation", policy.ForKey("common", "legal.body").MessageFormat)
	require.Equal(t, "raw", policy.ForKey("common", "legal.deep.body").MessageFormat)
	require.Equal(t, "icu", policy.ForKey("other", "ordinary").MessageFormat)
	require.True(t, policy.ForKey("other", "ordinary").Printf)
	require.Len(t, policy.Overrides, 5)
}

func TestValidationOverridesRejectAmbiguousScopes(t *testing.T) {
	guard, err := fsutil.NewGuard(t.TempDir())
	require.NoError(t, err)
	service := NewService(guard, "")
	for _, rules := range [][]ValidationOverride{
		{{MessageFormat: "raw"}},
		{{Key: "x", KeyPrefix: "x", MessageFormat: "raw"}},
		{{Key: "x", MessageFormat: "invalid"}},
		{{Key: "x", MessageFormat: "raw"}, {Key: "x", MessageFormat: "icu"}},
	} {
		cfg := Defaults()
		cfg.Validation.Overrides = rules
		require.False(t, service.Validate(t.Context(), Resolved{File: cfg}).Valid)
	}
}
