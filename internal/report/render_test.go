package report

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Ret2Hell/i18n-mcp/internal/deadkey"
	"github.com/Ret2Hell/i18n-mcp/internal/diff"
	"github.com/Ret2Hell/i18n-mcp/internal/locale"
	"github.com/Ret2Hell/i18n-mcp/internal/scanner"
	"github.com/stretchr/testify/require"
)

func TestRenderMarkdownGolden(t *testing.T) {
	report := makeGoldenReport()
	got, err := RenderMarkdown(report)
	require.NoError(t, err)
	want := readGolden(t, "audit.md.golden")
	require.Equal(t, want, got)
}

func TestRenderJSONGolden(t *testing.T) {
	report := makeGoldenReport()
	got, err := RenderJSON(report)
	require.NoError(t, err)
	want := readGolden(t, "audit.json.golden")
	require.JSONEq(t, want, got)
}

func readGolden(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	return string(data)
}

func goldenTime() time.Time {
	return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
}

func makeGoldenReport() Report {
	return Report{
		GeneratedAt: goldenTime(),
		ProjectRoot: "/repo",
		Inventory: locale.Inventory{
			SourceLocale:      "en",
			TargetLocales:     []string{"fr"},
			Locales:           []string{"en", "fr"},
			Namespaces:        []string{"common"},
			CountsByLocale:    map[string]int{"en": 2, "fr": 1},
			CountsByNamespace: map[string]int{"common": 3},
		},
		Diff: diff.Report{
			SourceLocale:  "en",
			TargetLocales: []string{"fr"},
			Summary: diff.Summary{
				Total:   2,
				Current: 1,
				Missing: 1,
			},
			Items: []diff.KeyDiff{
				{Locale: "fr", Namespace: "common", Key: "bye", Status: diff.Missing, SourceValue: "Bye"},
			},
		},
		Usage: scanner.Report{GeneratedAt: goldenTime()},
		DeadKeys: deadkey.Report{
			SourceLocale: "en",
			GeneratedAt:  goldenTime(),
			Summary:      deadkey.Summary{Total: 1, ProbablyUnused: 1},
			Items: []deadkey.Item{
				{Namespace: "common", Key: "unused", FullKey: "common:unused", Status: deadkey.StatusProbablyUnused, Confidence: scanner.ConfidenceHigh, Reasons: []string{"not referenced"}},
			},
		},
		Summary: Summary{
			Locales:        2,
			Namespaces:     1,
			SourceKeys:     2,
			TargetKeys:     1,
			Missing:        1,
			Stale:          0,
			Invalid:        0,
			Extra:          0,
			ProbablyUnused: 1,
			Warnings:       1,
		},
		Warnings: []string{"fixture warning"},
	}
}
