package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMatchLocalePattern(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		path    string
		want    LocaleFileCandidate
		wantOK  bool
	}{
		{
			name:    "flat locale",
			pattern: "messages/{locale}.json",
			path:    "messages/en.json",
			want:    LocaleFileCandidate{Locale: "en"},
			wantOK:  true,
		},
		{
			name:    "nested namespace",
			pattern: "locales/{locale}/{namespace}.json",
			path:    "locales/de/common.json",
			want:    LocaleFileCandidate{Locale: "de", Namespace: "common"},
			wantOK:  true,
		},
		{
			name:    "namespace directory",
			pattern: "messages/{locale}/{namespace}/index.json",
			path:    "messages/en/admin/index.json",
			want:    LocaleFileCandidate{Locale: "en", Namespace: "admin"},
			wantOK:  true,
		},
		{name: "different length", pattern: "messages/{locale}.json", path: "messages/en/common.json", wantOK: false},
		{name: "literal mismatch", pattern: "messages/{locale}.json", path: "locales/en.json", wantOK: false},
		{name: "missing locale capture", pattern: "messages/common.json", path: "messages/common.json", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := matchLocalePattern(tt.pattern, tt.path)
			require.Equal(t, tt.wantOK, ok)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestCountStrings(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  int
	}{
		{name: "string", value: "hello", want: 1},
		{name: "array", value: []any{"a", 1.0, map[string]any{"nested": "b"}}, want: 2},
		{name: "object", value: map[string]any{"a": "A", "n": 1.0, "nested": map[string]any{"b": "B"}}, want: 2},
		{name: "number", value: 1.0, want: 0},
		{name: "nil", value: nil, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, countStrings(tt.value))
		})
	}
}

func TestCountJSONStrings(t *testing.T) {
	root := t.TempDir()
	valid := filepath.Join(root, "valid.json")
	invalid := filepath.Join(root, "invalid.json")
	require.NoError(t, os.WriteFile(valid, []byte(`{"a":"A","items":["B",1,{"c":"C"}],"n":3}`), 0o644))
	require.NoError(t, os.WriteFile(invalid, []byte(`{"a":`), 0o644))

	keys, bytes, err := countJSONStrings(valid)
	require.NoError(t, err)
	require.Equal(t, 3, keys)
	require.Equal(t, len(`{"a":"A","items":["B",1,{"c":"C"}],"n":3}`), bytes)

	keys, bytes, err = countJSONStrings(invalid)
	require.Error(t, err)
	require.Zero(t, keys)
	require.Equal(t, len(`{"a":`), bytes)

	keys, bytes, err = countJSONStrings(filepath.Join(root, "missing.json"))
	require.Error(t, err)
	require.Zero(t, keys)
	require.Zero(t, bytes)
}

func TestBestSourceCandidates(t *testing.T) {
	tests := []struct {
		name    string
		locales []string
		want    []string
	}{
		{name: "preferred first without duplicates", locales: []string{"fr", "en-US", "en-AU", "en"}, want: []string{"en", "en-US", "en-AU", "fr"}},
		{name: "english regional before non english", locales: []string{"fr", "en-CA", "de"}, want: []string{"en-CA", "fr", "de"}},
		{name: "no english keeps original order", locales: []string{"fr", "de"}, want: []string{"fr", "de"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, bestSourceCandidates(tt.locales))
		})
	}
}

func TestUniqueLocales(t *testing.T) {
	got := uniqueLocales([]LocaleLayout{{Files: []LocaleFileCandidate{
		{Locale: "fr"},
		{Locale: "en"},
		{Locale: "fr"},
	}}})

	require.Equal(t, []string{"en", "fr"}, got)
}
