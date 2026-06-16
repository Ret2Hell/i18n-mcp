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

	requireDiagnostic(t, got.Errors, "target_contains_source", "targetLocales")
	requireDiagnostic(t, got.Errors, "locale_pattern_missing_locale", "localeFiles[0]")
	requireDiagnostic(t, got.Errors, "invalid_indent", "format.indent")
	requireDiagnostic(t, got.Errors, "invalid_translation_mode", "translation.mode")
}

func TestValidateRejectsProjectPathsOutsideRoot(t *testing.T) {
	ctx := t.Context()
	root := t.TempDir()
	outside := t.TempDir()
	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	service := NewService(guard, "")

	cfg := Resolved{File: Defaults(), ProjectRoot: root}
	cfg.Translation.StyleGuidePath = outside + "/style.md"
	cfg.Translation.GlossaryPath = "../glossary.md"

	got := service.Validate(ctx, cfg)
	require.False(t, got.Valid)
	requireDiagnostic(t, got.Errors, "path_escapes_project", "translation.styleGuidePath")
	requireDiagnostic(t, got.Errors, "path_escapes_project", "translation.glossaryPath")
}

func TestValidateAcceptsProjectPathsInsideRoot(t *testing.T) {
	ctx := t.Context()
	root := t.TempDir()
	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	service := NewService(guard, "")

	cfg := Resolved{File: Defaults(), ProjectRoot: root}
	cfg.Translation.StyleGuidePath = "docs/style.md"
	cfg.Translation.GlossaryPath = "docs/glossary.md"

	got := service.Validate(ctx, cfg)
	require.True(t, got.Valid)
	require.Empty(t, got.Errors)
}

func requireDiagnostic(t *testing.T, diagnostics []Diagnostic, code string, field string) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code && diagnostic.Field == field {
			return
		}
	}
	require.Failf(t, "missing diagnostic", "code %q field %q not found in %#v", code, field, diagnostics)
}
