package config

import (
	"slices"
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

func TestValidateRejectsEmptyLocaleFiles(t *testing.T) {
	ctx := t.Context()
	root := t.TempDir()
	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	service := NewService(guard, "")

	cfg := Resolved{File: Defaults(), ProjectRoot: root}
	cfg.LocaleFiles = nil

	got := service.Validate(ctx, cfg)
	require.False(t, got.Valid)
	requireDiagnostic(t, got.Errors, "locale_files_required", "localeFiles")
}

func TestValidateFormatIndentBoundaries(t *testing.T) {
	ctx := t.Context()
	root := t.TempDir()
	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	service := NewService(guard, "")

	tests := []struct {
		name  string
		value int
		valid bool
	}{
		{name: "minus one", value: -1, valid: false},
		{name: "zero", value: 0, valid: true},
		{name: "one", value: 1, valid: true},
		{name: "eight", value: 8, valid: true},
		{name: "nine", value: 9, valid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Resolved{File: Defaults(), ProjectRoot: root}
			cfg.Format.Indent = tt.value

			got := service.Validate(ctx, cfg)
			require.Equal(t, tt.valid, got.Valid)
			if tt.valid {
				require.Empty(t, got.Errors)
				return
			}
			requireDiagnostic(t, got.Errors, "invalid_indent", "format.indent")
		})
	}
}

func requireDiagnostic(t *testing.T, diagnostics []Diagnostic, code string, field string) {
	t.Helper()
	if slices.ContainsFunc(diagnostics, func(diagnostic Diagnostic) bool {
		return diagnostic.Code == code && diagnostic.Field == field
	}) {
		return
	}
	require.Failf(t, "missing diagnostic", "code %q field %q not found in %#v", code, field, diagnostics)
}
