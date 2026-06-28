package diff

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
	"github.com/Ret2Hell/i18n-mcp/internal/locale"
	"github.com/Ret2Hell/i18n-mcp/internal/state"
	"github.com/Ret2Hell/i18n-mcp/internal/validate"
	"github.com/stretchr/testify/require"
)

func TestAnalyzeCoversAllStatuses(t *testing.T) {
	root := makeDiffFixture(t)
	report := analyzeFixture(t, root)

	require.Equal(t, "en", report.SourceLocale)
	require.Equal(t, []string{"de", "fr"}, report.TargetLocales)
	require.Equal(t, 11, report.Summary.Total)
	require.Equal(t, 1, report.Summary.Current)
	require.Equal(t, 6, report.Summary.Missing)
	require.Equal(t, 1, report.Summary.Stale)
	require.Equal(t, 1, report.Summary.Extra)
	require.Equal(t, 1, report.Summary.Invalid)
	require.Equal(t, 1, report.Summary.Unknown)

	statuses := statusesByLocaleKey(report.Items)
	require.Equal(t, Current, statuses["fr/common/current"])
	require.Equal(t, Stale, statuses["fr/common/stale"])
	require.Equal(t, Unknown, statuses["fr/common/unknown"])
	require.Equal(t, Invalid, statuses["fr/common/invalid"])
	require.Equal(t, Missing, statuses["fr/common/missing"])
	require.Equal(t, Extra, statuses["fr/common/extra"])
	require.Equal(t, Missing, statuses["de/common/current"])
}

func TestAnalyzeIsDeterministicallySorted(t *testing.T) {
	report := analyzeFixture(t, makeDiffFixture(t))

	for i := range len(report.Items) - 1 {
		require.False(t, lessKeyDiff(report.Items[i+1], report.Items[i]), "items are not sorted at index %d", i+1)
	}
}

func TestSummarizeEmptyItemsHasNilByLocale(t *testing.T) {
	for _, items := range [][]KeyDiff{nil, []KeyDiff{}} {
		summary := Summarize(items)

		require.Zero(t, summary.Total)
		require.Nil(t, summary.ByLocale)
	}
}

func TestSummarizeCountsByStatusAndLocale(t *testing.T) {
	summary := Summarize([]KeyDiff{
		{Locale: "fr", Status: Current},
		{Locale: "fr", Status: Missing},
		{Locale: "de", Status: Missing},
		{Locale: "de", Status: Invalid},
	})

	require.Equal(t, 4, summary.Total)
	require.Equal(t, 1, summary.Current)
	require.Equal(t, 2, summary.Missing)
	require.Equal(t, 1, summary.Invalid)
	require.Equal(t, StatusCounts{Current: 1, Missing: 1}, summary.ByLocale["fr"])
	require.Equal(t, StatusCounts{Missing: 1, Invalid: 1}, summary.ByLocale["de"])
}

func analyzeFixture(t *testing.T, root string) Report {
	t.Helper()
	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	configService := config.NewService(guard, "")
	localeService := locale.NewService(guard, configService)
	stateStore := state.NewStore(guard)
	stateService := state.NewService(stateStore, localeService)
	service := NewService(localeService, stateService, validate.NewService())
	report, err := service.Analyze(t.Context())
	require.NoError(t, err)
	return report
}

func makeDiffFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeDiffFile(t, root, ".i18n-mcp.json", `{
  "sourceLocale": "en",
  "targetLocales": ["fr", "de"],
  "localeFiles": ["messages/{locale}.json"],
  "defaultNamespace": "common"
}`)
	writeDiffFile(t, root, "messages/en.json", `{
  "current": "Current source",
  "stale": "New stale source",
  "unknown": "Unknown source",
  "invalid": "Hello {name}",
  "missing": "Missing source"
}`)
	writeDiffFile(t, root, "messages/fr.json", `{
  "current": "Source actuelle",
  "stale": "Ancienne traduction",
  "unknown": "Traduction inconnue",
  "invalid": "Bonjour",
  "extra": "Texte en trop"
}`)

	stateFile := state.NewFile("en", time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))
	stateFile.Entries[state.EntryKey("fr", "common", "current")] = state.Entry{
		Locale:             "fr",
		Namespace:          "common",
		Key:                "current",
		SourceHash:         state.SourceHash("Current source"),
		TranslatedFromHash: state.SourceHash("Current source"),
		TargetHash:         state.TargetHash("Source actuelle"),
		Status:             state.StatusCurrent,
		UpdatedAt:          time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
	}
	stateFile.Entries[state.EntryKey("fr", "common", "stale")] = state.Entry{
		Locale:             "fr",
		Namespace:          "common",
		Key:                "stale",
		SourceHash:         state.SourceHash("Old stale source"),
		TranslatedFromHash: state.SourceHash("Old stale source"),
		TargetHash:         state.TargetHash("Ancienne traduction"),
		Status:             state.StatusCurrent,
		UpdatedAt:          time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
	}
	data, err := json.MarshalIndent(stateFile, "", "  ")
	require.NoError(t, err)
	writeDiffFile(t, root, state.DefaultStatePath, string(data)+"\n")
	return root
}

func writeDiffFile(t *testing.T, root string, relPath string, contents string) {
	t.Helper()
	abs := filepath.Join(root, relPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o700))
	require.NoError(t, os.WriteFile(abs, []byte(contents), 0o600))
}

func statusesByLocaleKey(items []KeyDiff) map[string]KeyStatus {
	out := make(map[string]KeyStatus, len(items))
	for _, item := range items {
		out[item.Locale+"/"+item.Namespace+"/"+item.Key] = item.Status
	}
	return out
}
