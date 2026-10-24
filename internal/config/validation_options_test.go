package config

import (
	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func TestValidationPolicyUsesProjectAndExplicitOverrides(t *testing.T) {
	root := t.TempDir()
	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	service := NewService(guard, "")
	for _, c := range []struct{ pkg, format string }{
		{`{"dependencies":{"next-intl":"4"}}`, "icu"},
		{`{"devDependencies":{"i18next":"25"}}`, "i18next"},
		{`{}`, ""},
	} {
		require.NoError(t, os.WriteFile(filepath.Join(root, "package.json"), []byte(c.pkg), 0600))
		options, err := service.ResolveValidation(t.Context(), ValidationConfig{})
		require.NoError(t, err)
		require.Equal(t, c.format, options.MessageFormat)
		require.False(t, options.Printf)
		options, err = service.ResolveValidation(t.Context(), ValidationConfig{MessageFormat: "interpolation", Printf: true})
		require.NoError(t, err)
		require.Equal(t, "interpolation", options.MessageFormat)
		require.True(t, options.Printf)
	}
	cfg := Defaults()
	cfg.Validation.MessageFormat = "unknown"
	result := service.Validate(t.Context(), Resolved{File: cfg})
	require.False(t, result.Valid)
}
