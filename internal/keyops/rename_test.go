package keyops_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/keyops"
	"github.com/Ret2Hell/i18n-mcp/internal/state"
	"github.com/stretchr/testify/require"
)

func TestRenameDryRunPreviewsNestedRenameWithoutWriting(t *testing.T) {
	a := newRenameFixtureApp(t)
	before := readRenameFile(t, a.ProjectRoot, "messages/en/auth.json")

	out, err := a.KeyOps.Rename(t.Context(), keyops.RenameInput{Namespace: "auth", FromKey: "login.title", ToKey: "login.heading"})
	require.NoError(t, err)
	require.True(t, out.DryRun)
	require.Len(t, out.ChangedFiles, 2)
	require.Empty(t, out.Conflicts)
	require.Equal(t, before, readRenameFile(t, a.ProjectRoot, "messages/en/auth.json"))
}

func TestRenameWriteMovesKeysAndState(t *testing.T) {
	a := newRenameFixtureApp(t)

	out, err := a.KeyOps.Rename(t.Context(), keyops.RenameInput{Apply: true, Namespace: "auth", FromKey: "login.title", ToKey: "login.heading"})
	require.NoError(t, err)
	require.False(t, out.DryRun)
	require.Equal(t, 2, out.Renamed)
	enAuth := readRenameJSON(t, a.ProjectRoot, "messages/en/auth.json")
	login, ok := enAuth["login"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "Log in", login["heading"])
	require.NotContains(t, login, "title")

	stateBytes := readRenameFile(t, a.ProjectRoot, state.DefaultStatePath)
	require.Contains(t, stateBytes, "login.heading")
	require.NotContains(t, stateBytes, "login.title")
	require.Contains(t, stateBytes, "keys.rename")
}

func TestRenameRejectsDestinationCollisionByDefault(t *testing.T) {
	a := newRenameFixtureApp(t)
	out, err := a.KeyOps.Rename(t.Context(), keyops.RenameInput{Namespace: "auth", FromKey: "login.title", ToKey: "login.subtitle"})
	require.NoError(t, err)
	require.NotEmpty(t, out.Conflicts)
}

func TestRenameOverwriteDestinationWhenExplicit(t *testing.T) {
	a := newRenameFixtureApp(t)
	out, err := a.KeyOps.Rename(t.Context(), keyops.RenameInput{Apply: true, Namespace: "auth", FromKey: "login.title", ToKey: "login.subtitle", ConflictPolicy: keyops.ConflictOverwrite})
	require.NoError(t, err)
	require.Empty(t, out.Conflicts)
	require.Contains(t, readRenameFile(t, a.ProjectRoot, "messages/en/auth.json"), "Log in")
}

func TestRenameSkipsMissingTargetLocaleKey(t *testing.T) {
	a := newRenameFixtureApp(t)
	out, err := a.KeyOps.Rename(t.Context(), keyops.RenameInput{Namespace: "auth", FromKey: "only.source", ToKey: "only.renamed"})
	require.NoError(t, err)
	require.Empty(t, out.Conflicts)
	require.NotEmpty(t, out.Warnings)
}

func TestRenameDryRunFalseWithoutApplyDoesNotWrite(t *testing.T) {
	a := newRenameFixtureApp(t)
	before := readRenameFile(t, a.ProjectRoot, "messages/en/auth.json")

	out, err := a.KeyOps.Rename(t.Context(), keyops.RenameInput{DryRun: new(false), Namespace: "auth", FromKey: "login.title", ToKey: "login.heading"})
	require.NoError(t, err)
	require.True(t, out.DryRun)
	require.Equal(t, before, readRenameFile(t, a.ProjectRoot, "messages/en/auth.json"))
}

func TestRenameApplyTrueWithDryRunTrueDoesNotWrite(t *testing.T) {
	a := newRenameFixtureApp(t)
	before := readRenameFile(t, a.ProjectRoot, "messages/en/auth.json")

	out, err := a.KeyOps.Rename(t.Context(), keyops.RenameInput{Apply: true, DryRun: new(true), Namespace: "auth", FromKey: "login.title", ToKey: "login.heading"})
	require.NoError(t, err)
	require.True(t, out.DryRun)
	require.Equal(t, before, readRenameFile(t, a.ProjectRoot, "messages/en/auth.json"))
}

func TestRenameSelectedSourceLocaleOnlyUpdatesSourceAndSourceState(t *testing.T) {
	a := newRenameFixtureApp(t)

	out, err := a.KeyOps.Rename(t.Context(), keyops.RenameInput{Apply: true, Locales: []string{"en"}, Namespace: "auth", FromKey: "login.title", ToKey: "login.heading"})
	require.NoError(t, err)
	require.Equal(t, 1, out.Renamed)
	require.Contains(t, readRenameFile(t, a.ProjectRoot, "messages/en/auth.json"), "heading")
	require.Contains(t, readRenameFile(t, a.ProjectRoot, "messages/fr/auth.json"), "title")

	stateFile := readRenameState(t, a.ProjectRoot)
	require.Contains(t, stateFile.Entries, state.EntryKey("en", "auth", "login.heading"))
	require.NotContains(t, stateFile.Entries, state.EntryKey("en", "auth", "login.title"))
	require.Contains(t, stateFile.Entries, state.EntryKey("fr", "auth", "login.title"))
}

func TestRenameRejectsAncestorOrDescendantMoves(t *testing.T) {
	a := newRenameFixtureApp(t)

	for _, tc := range []struct {
		name string
		from string
		to   string
	}{
		{name: "descendant", from: "login", to: "login.title"},
		{name: "ancestor", from: "login.title", to: "login"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := a.KeyOps.Rename(t.Context(), keyops.RenameInput{Namespace: "auth", FromKey: tc.from, ToKey: tc.to})
			require.NoError(t, err)
			require.NotEmpty(t, out.Conflicts)
		})
	}
}

func TestRenamePreservesStateEntryHashes(t *testing.T) {
	a := newRenameFixtureApp(t)
	before := readRenameState(t, a.ProjectRoot).Entries[state.EntryKey("fr", "auth", "login.title")]

	_, err := a.KeyOps.Rename(t.Context(), keyops.RenameInput{Apply: true, Namespace: "auth", FromKey: "login.title", ToKey: "login.heading"})
	require.NoError(t, err)
	after := readRenameState(t, a.ProjectRoot).Entries[state.EntryKey("fr", "auth", "login.heading")]
	require.Equal(t, before.SourceHash, after.SourceHash)
	require.Equal(t, before.TranslatedFromHash, after.TranslatedFromHash)
	require.Equal(t, before.TargetHash, after.TargetHash)
	require.True(t, after.Reviewed)
}

func newRenameFixtureApp(t *testing.T) *app.App {
	t.Helper()
	root := t.TempDir()
	writeRenameFile(t, root, ".i18n-mcp.json", `{
  "sourceLocale": "en",
  "targetLocales": ["fr"],
  "localeFiles": ["messages/{locale}/{namespace}.json"],
  "defaultNamespace": "common",
  "format": {"sortKeys": false, "indent": 2, "trailingNewline": true},
  "translation": {"mode": "agent"}
}
`)
	writeRenameFile(t, root, "messages/en/auth.json", `{
  "login": {
    "title": "Log in",
    "subtitle": "Welcome back"
  },
  "only": {
    "source": "Source only"
  }
}
`)
	writeRenameFile(t, root, "messages/fr/auth.json", `{
  "login": {
    "title": "Connexion",
    "subtitle": "Bienvenue"
  }
}
`)
	stateFile := state.NewFile("en", renameTestTime())
	stateFile.Entries[state.EntryKey("en", "auth", "login.title")] = state.Entry{
		Locale:             "en",
		Namespace:          "auth",
		Key:                "login.title",
		SourceHash:         state.SourceHash("Log in"),
		TranslatedFromHash: state.SourceHash("Log in"),
		TargetHash:         state.TargetHash("Log in"),
		Status:             state.StatusCurrent,
		Reviewed:           true,
		UpdatedAt:          renameTestTime(),
		UpdatedBy:          "fixture",
	}
	stateFile.Entries[state.EntryKey("fr", "auth", "login.title")] = state.Entry{
		Locale:             "fr",
		Namespace:          "auth",
		Key:                "login.title",
		SourceHash:         state.SourceHash("Log in"),
		TranslatedFromHash: state.SourceHash("Log in"),
		TargetHash:         state.TargetHash("Connexion"),
		Status:             state.StatusCurrent,
		Reviewed:           true,
		UpdatedAt:          renameTestTime(),
		UpdatedBy:          "fixture",
	}
	data, err := json.MarshalIndent(stateFile, "", "  ")
	require.NoError(t, err)
	writeRenameFile(t, root, state.DefaultStatePath, string(data)+"\n")

	a, err := app.New(t.Context(), app.Options{ProjectRoot: root})
	require.NoError(t, err)
	return a
}

func renameTestTime() time.Time {
	return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
}

func writeRenameFile(t *testing.T, root string, relPath string, contents string) {
	t.Helper()
	abs := filepath.Join(root, relPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o700))
	require.NoError(t, os.WriteFile(abs, []byte(contents), 0o600))
}

func readRenameFile(t *testing.T, root string, relPath string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, relPath))
	require.NoError(t, err)
	return string(data)
}

func readRenameState(t *testing.T, root string) state.File {
	t.Helper()
	var stateFile state.File
	require.NoError(t, json.Unmarshal([]byte(readRenameFile(t, root, state.DefaultStatePath)), &stateFile))
	return stateFile
}

func readRenameJSON(t *testing.T, root string, relPath string) map[string]any {
	t.Helper()
	var out map[string]any
	require.NoError(t, json.Unmarshal([]byte(readRenameFile(t, root, relPath)), &out))
	return out
}
