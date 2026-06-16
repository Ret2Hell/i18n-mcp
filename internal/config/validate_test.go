package config

import (
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
	"github.com/stretchr/testify/require"
)

func TestValidateDefaultsAreValid(t *testing.T) {
	ctx := t.Context()
	root := t.TempDir()
	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	service := NewService(guard, "")
	cfg, err := service.Resolve(ctx)
	require.NoError(t, err)

	got := service.Validate(ctx, cfg)
	require.True(t, got.Valid)
	require.Empty(t, got.Errors)
}

func TestValidateRejectsBadConfig(t *testing.T) {
	ctx := t.Context()
	root := t.TempDir()
	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	service := NewService(guard, "")

	cfg := Resolved{
		File: File{
			SourceLocale:  "en",
			TargetLocales: []string{"en"},
			LocaleFiles:   []string{"messages/*.json"},
			Format:        FormatConfig{Indent: 20},
			Translation:   TranslationConfig{Mode: "bad"},
		},
		ProjectRoot: root,
	}

	got := service.Validate(ctx, cfg)
	require.False(t, got.Valid)
	require.NotEmpty(t, got.Errors)

	codes := map[string]bool{}
	for _, diagnostic := range got.Errors {
		codes[diagnostic.Code] = true
	}
	require.True(t, codes["target_contains_source"])
	require.True(t, codes["locale_pattern_missing_locale"])
	require.True(t, codes["invalid_indent"])
	require.True(t, codes["invalid_translation_mode"])
}
