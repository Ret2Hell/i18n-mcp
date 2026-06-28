package translate_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/state"
	"github.com/Ret2Hell/i18n-mcp/internal/translate"
	"github.com/Ret2Hell/i18n-mcp/internal/validate"
	"github.com/stretchr/testify/require"
)

func TestTranslationPlanIncludesMissingAndStale(t *testing.T) {
	ctx := t.Context()
	a := newTranslationFixtureApp(t)

	batch, err := a.Translation.Plan(ctx, translate.PlanInput{})
	require.NoError(t, err)
	require.NotEmpty(t, batch.BatchID)
	require.Len(t, batch.Items, 2)
	require.Equal(t, "fr", batch.Items[0].Locale)
	require.Equal(t, "auth", batch.Items[0].Namespace)
}

func TestTranslationValidateRejectsSourceDrift(t *testing.T) {
	ctx := t.Context()
	a := newTranslationFixtureApp(t)
	batch, err := a.Translation.Plan(ctx, translate.PlanInput{})
	require.NoError(t, err)

	out, err := a.Translation.Validate(ctx, translate.ValidationInput{
		BatchID: batch.BatchID,
		Translations: []translate.ProposedTranslation{{
			Locale:      "fr",
			Namespace:   "auth",
			Key:         "login.title",
			SourceValue: "old source",
			Value:       "Connexion",
		}},
	})
	require.NoError(t, err)
	require.Len(t, out.Rejected, 1)
	requireIssueCode(t, out.Rejected[0].Issues, "source_drift")
}

func TestTranslationApplyDryRunDoesNotWrite(t *testing.T) {
	ctx := t.Context()
	a := newTranslationFixtureApp(t)
	before := readFixtureFile(t, a.ProjectRoot, "messages/fr/auth.json")
	stateBefore := readFixtureFile(t, a.ProjectRoot, state.DefaultStatePath)

	out, err := a.Translation.Apply(ctx, translate.ApplyInput{Translations: []translate.ProposedTranslation{{
		Locale:      "fr",
		Namespace:   "auth",
		Key:         "login.title",
		SourceValue: "Log in",
		Value:       "Connexion",
	}}})
	require.NoError(t, err)
	require.True(t, out.DryRun)
	require.Len(t, out.ChangedFiles, 1)
	require.Equal(t, before, readFixtureFile(t, a.ProjectRoot, "messages/fr/auth.json"))
	require.Equal(t, stateBefore, readFixtureFile(t, a.ProjectRoot, state.DefaultStatePath))
}

func TestTranslationApplyWriteUpdatesLocaleAndState(t *testing.T) {
	ctx := t.Context()
	a := newTranslationFixtureApp(t)

	out, err := a.Translation.Apply(ctx, translate.ApplyInput{Apply: true, Translations: []translate.ProposedTranslation{{
		Locale:      "fr",
		Namespace:   "auth",
		Key:         "login.title",
		SourceValue: "Log in",
		Value:       "Connexion",
	}}})
	require.NoError(t, err)
	require.False(t, out.DryRun)
	require.Equal(t, 1, out.Applied)
	require.Contains(t, readFixtureFile(t, a.ProjectRoot, "messages/fr/auth.json"), "Connexion")

	stateBytes := readFixtureFile(t, a.ProjectRoot, state.DefaultStatePath)
	require.Contains(t, stateBytes, "translation.apply")
}

func newTranslationFixtureApp(t *testing.T) *app.App {
	t.Helper()
	root := t.TempDir()
	writeFixtureFile(t, root, ".i18n-mcp.json", `{
  "sourceLocale": "en",
  "targetLocales": ["fr"],
  "localeFiles": ["messages/{locale}/{namespace}.json"],
  "defaultNamespace": "common",
  "format": {"sortKeys": false, "indent": 2, "trailingNewline": true},
  "translation": {"mode": "agent"}
}
`)
	writeFixtureFile(t, root, "messages/en/auth.json", `{
  "login": {
    "title": "Log in",
    "subtitle": "Welcome {name}"
  }
}
`)
	writeFixtureFile(t, root, "messages/fr/auth.json", `{
  "login": {
    "subtitle": "Bienvenue {name}"
  }
}
`)
	stateFile := state.NewFile("en", testTime())
	stateFile.Entries[state.EntryKey("fr", "auth", "login.subtitle")] = state.Entry{
		Locale:             "fr",
		Namespace:          "auth",
		Key:                "login.subtitle",
		SourceHash:         state.SourceHash("Welcome {name}"),
		TranslatedFromHash: state.SourceHash("Old welcome {name}"),
		TargetHash:         state.TargetHash("Bienvenue {name}"),
		Status:             state.StatusCurrent,
		UpdatedAt:          testTime(),
		UpdatedBy:          "fixture",
	}
	data, err := json.MarshalIndent(stateFile, "", "  ")
	require.NoError(t, err)
	writeFixtureFile(t, root, state.DefaultStatePath, string(data)+"\n")

	a, err := app.New(t.Context(), app.Options{ProjectRoot: root, LogLevel: "error"})
	require.NoError(t, err)
	return a
}

func writeFixtureFile(t *testing.T, root string, relPath string, contents string) {
	t.Helper()
	abs := filepath.Join(root, relPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o700))
	require.NoError(t, os.WriteFile(abs, []byte(contents), 0o600))
}

func readFixtureFile(t *testing.T, root string, relPath string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, relPath))
	require.NoError(t, err)
	return string(data)
}

func testTime() time.Time {
	return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
}

func requireIssueCode(t *testing.T, issues []validate.Issue, code string) {
	t.Helper()
	if slices.ContainsFunc(issues, func(issue validate.Issue) bool { return issue.Code == code }) {
		return
	}
	require.Failf(t, "missing issue code", "expected %s in %#v", code, issues)
}
