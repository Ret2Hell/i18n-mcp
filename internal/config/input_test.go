package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
	"github.com/stretchr/testify/require"
)

func TestMinimalInputSchemaAndDefaultNormalization(t *testing.T) {
	raw := []byte(`{"sourceLocale":"en","localeFiles":["messages/{locale}.json"]}`)
	input, err := decodeInput(raw)
	require.NoError(t, err)
	cfg := NormalizeInput(input)
	defaults := Defaults()
	require.Equal(t, defaults.Format, cfg.Format)
	require.Equal(t, defaults.CI, cfg.CI)
	require.Equal(t, "common", cfg.DefaultNamespace)
	schema, err := Schema()
	require.NoError(t, err)
	resolved, err := schema.Resolve(nil)
	require.NoError(t, err)
	var value any
	require.NoError(t, json.Unmarshal(raw, &value))
	require.NoError(t, resolved.Validate(value))
	require.Error(t, resolved.Validate(map[string]any{"sourceLocale": "en"}))
	require.Error(t, resolved.Validate(map[string]any{"localeFiles": []any{"messages/{locale}.json"}}))
}

func TestPresenceAwareInputPreservesExplicitValuesAndReplacementSemantics(t *testing.T) {
	root := t.TempDir()
	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	service := NewService(guard, "")
	raw := []byte(`{"sourceLocale":"en","localeFiles":["messages/{locale}.json"],"targetLocales":[],"translationFunctions":[],"format":{"indent":0,"trailingNewline":false},"ci":{"failOnMissing":false,"failOnInvalid":false},"validation":{"printf":false}}`)
	input, err := decodeInput(raw)
	require.NoError(t, err)
	expected := NormalizeInput(input)
	require.Zero(t, expected.Format.Indent)
	require.False(t, expected.Format.TrailingNewline)
	require.False(t, expected.CI.FailOnMissing)
	require.False(t, expected.CI.FailOnInvalid)
	require.NotNil(t, expected.TargetLocales)
	require.Empty(t, expected.TargetLocales)
	require.NotNil(t, expected.TranslationFunctions)
	require.Empty(t, expected.TranslationFunctions)
	preview, err := service.Write(t.Context(), WriteInput{Config: input})
	require.NoError(t, err)
	require.True(t, preview.Validation.Valid)
	_, err = os.Stat(filepath.Join(root, DefaultConfigFile))
	require.ErrorIs(t, err, os.ErrNotExist)
	applied, err := service.Write(t.Context(), WriteInput{Config: input, Apply: true})
	require.NoError(t, err)
	require.True(t, applied.Applied)
	got, err := service.Resolve(t.Context())
	require.NoError(t, err)
	require.Equal(t, expected, got.File)
	minimal, err := decodeInput([]byte(`{"sourceLocale":"en","localeFiles":["messages/{locale}.json"],"format":{},"ci":{},"translation":{}}`))
	require.NoError(t, err)
	_, err = service.Write(t.Context(), WriteInput{Config: minimal, Apply: true})
	require.NoError(t, err)
	got, err = service.Resolve(t.Context())
	require.NoError(t, err)
	require.True(t, got.CI.FailOnMissing)
	require.True(t, got.Format.TrailingNewline)
	require.Equal(t, 2, got.Format.Indent)
}

func TestInvalidInputIsNotDefaultedIntoValidity(t *testing.T) {
	root := t.TempDir()
	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	service := NewService(guard, "")
	for _, raw := range []string{
		`{"sourceLocale":"","localeFiles":["messages/{locale}.json"]}`,
		`{"sourceLocale":"en","localeFiles":[]}`,
		`{"sourceLocale":"en","localeFiles":["messages/{locale}.json"],"format":{"indent":9}}`,
		`{"sourceLocale":"en","localeFiles":["messages/{locale}.json"],"validation":{"messageFormat":"unsupported"}}`,
	} {
		in, err := decodeInput([]byte(raw))
		require.NoError(t, err)
		out, err := service.Write(t.Context(), WriteInput{Config: in, Apply: true})
		require.NoError(t, err)
		require.False(t, out.Validation.Valid)
		require.False(t, out.Applied)
	}
	_, err = os.Stat(filepath.Join(root, DefaultConfigFile))
	require.ErrorIs(t, err, os.ErrNotExist)
	_, err = decodeInput([]byte(`{"sourceLocale":"en","localeFiles":[],"format":{"typo":true}}`))
	require.Error(t, err)
}
