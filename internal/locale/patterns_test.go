package locale

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidatePatternRejectsInvalidPatterns(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		wantErr string
	}{
		{name: "empty", pattern: "  ", wantErr: "empty"},
		{name: "absolute filepath", pattern: "/messages/{locale}.json", wantErr: "project-relative"},
		{name: "backslash", pattern: `messages\{locale}.json`, wantErr: "slash separators"},
		{name: "clean current directory", pattern: ".", wantErr: "traversal"},
		{name: "clean parent directory", pattern: "..", wantErr: "traversal"},
		{name: "climbs out", pattern: "../messages/{locale}.json", wantErr: "traversal"},
		{name: "contains traversal", pattern: "messages/../{locale}.json", wantErr: "traversal"},
		{name: "split traversal", pattern: "messages/..//{locale}.json", wantErr: "traversal"},
		{name: "missing locale", pattern: "messages/en.json", wantErr: "must contain {locale}"},
		{name: "duplicate locale", pattern: "{locale}/messages/{locale}.json", wantErr: "exactly one {locale}"},
		{name: "duplicate namespace", pattern: "{locale}/{namespace}/{namespace}.json", wantErr: "at most one {namespace}"},
		{name: "not json", pattern: "messages/{locale}.yaml", wantErr: "JSON files"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePattern(tt.pattern)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		relPath string
		want    FileRef
		wantOK  bool
		wantErr string
	}{
		{
			name:    "flat locale file",
			pattern: "messages/{locale}.json",
			relPath: "./messages/en.json",
			want:    FileRef{Locale: "en", Path: "messages/en.json", Pattern: "messages/{locale}.json"},
			wantOK:  true,
		},
		{
			name:    "namespaced locale file",
			pattern: "locales/{locale}/{namespace}.json",
			relPath: "locales/de/common.json",
			want:    FileRef{Locale: "de", Namespace: "common", Path: "locales/de/common.json", Pattern: "locales/{locale}/{namespace}.json"},
			wantOK:  true,
		},
		{
			name:    "path mismatch",
			pattern: "messages/{locale}.json",
			relPath: "other/en.json",
			wantOK:  false,
		},
		{
			name:    "invalid pattern",
			pattern: "messages/en.json",
			relPath: "messages/en.json",
			wantOK:  false,
			wantErr: "must contain {locale}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok, err := MatchPattern(tt.pattern, tt.relPath)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantOK, ok)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestLessFileRef(t *testing.T) {
	base := FileRef{Locale: "en", Namespace: "common", Path: "messages/en.json", Pattern: "messages/{locale}.json"}

	tests := []struct {
		name string
		a    FileRef
		b    FileRef
		want bool
	}{
		{name: "equal", a: base, b: base, want: false},
		{name: "locale less", a: FileRef{Locale: "de"}, b: base, want: true},
		{name: "namespace less", a: FileRef{Locale: "en", Namespace: "admin"}, b: base, want: true},
		{name: "path less", a: FileRef{Locale: "en", Namespace: "common", Path: "a.json"}, b: base, want: true},
		{name: "pattern less", a: FileRef{Locale: "en", Namespace: "common", Path: "messages/en.json", Pattern: "a/{locale}.json"}, b: base, want: true},
		{name: "locale greater", a: base, b: FileRef{Locale: "de"}, want: false},
		{name: "namespace greater", a: base, b: FileRef{Locale: "en", Namespace: "admin"}, want: false},
		{name: "path greater", a: base, b: FileRef{Locale: "en", Namespace: "common", Path: "a.json"}, want: false},
		{name: "pattern greater", a: base, b: FileRef{Locale: "en", Namespace: "common", Path: "messages/en.json", Pattern: "a/{locale}.json"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, lessFileRef(tt.a, tt.b))
		})
	}
}
