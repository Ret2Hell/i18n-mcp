package deadkey_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/deadkey"
	"github.com/Ret2Hell/i18n-mcp/internal/resources"
	"github.com/stretchr/testify/require"
)

func TestDeadReportClassifiesProbablyUnusedIgnoredKeptAndDynamic(t *testing.T) {
	a := newDeadKeyFixtureApp(t)
	report, err := a.DeadKeys.Report(t.Context(), deadkey.ReportInput{RefreshUsage: true, IncludeUsed: true})
	require.NoError(t, err)
	statuses := statusesByKey(report)
	require.Equal(t, deadkey.StatusUsed, statuses["common\x00used"])
	require.Equal(t, deadkey.StatusProbablyUnused, statuses["common\x00unused"])
	require.Equal(t, deadkey.StatusIgnored, statuses["common\x00ignored"])
	require.Equal(t, deadkey.StatusKept, statuses["common\x00kept"])
	require.Equal(t, deadkey.StatusMaybeDynamic, statuses["routes\x00show"])
}

func TestPruneDryRunDoesNotWrite(t *testing.T) {
	a := newDeadKeyFixtureApp(t)
	before := readDeadKeyFile(t, a.ProjectRoot, "messages/en/common.json")
	out, err := a.DeadKeys.Prune(t.Context(), deadkey.PruneInput{Keys: []deadkey.PruneKey{{Namespace: "common", Key: "unused"}}})
	require.NoError(t, err)
	require.True(t, out.DryRun)
	require.NotEmpty(t, out.ChangedFiles)
	require.Equal(t, before, readDeadKeyFile(t, a.ProjectRoot, "messages/en/common.json"))
}

func TestPruneRejectsUnsafeKeysByDefault(t *testing.T) {
	a := newDeadKeyFixtureApp(t)
	out, err := a.DeadKeys.Prune(t.Context(), deadkey.PruneInput{Apply: true, Keys: []deadkey.PruneKey{{Namespace: "common", Key: "kept"}}})
	require.NoError(t, err)
	require.False(t, out.DryRun)
	require.Len(t, out.Rejected, 1)
	require.Empty(t, out.ChangedFiles)
	require.Zero(t, out.Pruned)
}

func TestBuildPruneEditsCanSelectUsedKeysWhenUnsafeAllowed(t *testing.T) {
	a := newDeadKeyFixtureApp(t)

	edits, rejected, err := a.DeadKeys.BuildPruneEdits(t.Context(), deadkey.PruneInput{
		AllowUnsafe: true,
		Keys:        []deadkey.PruneKey{{Namespace: "common", Key: "used"}},
	})

	require.NoError(t, err)
	require.Empty(t, rejected)
	require.NotEmpty(t, edits)
}

func TestBuildPruneEditsUsesCachedUsageReport(t *testing.T) {
	a := newDeadKeyFixtureApp(t)
	_, err := a.DeadKeys.Report(t.Context(), deadkey.ReportInput{RefreshUsage: true, IncludeUsed: true})
	require.NoError(t, err)
	require.NoError(t, os.Remove(filepath.Join(a.ProjectRoot, "app/page.tsx")))

	edits, rejected, err := a.DeadKeys.BuildPruneEdits(t.Context(), deadkey.PruneInput{
		Keys: []deadkey.PruneKey{{Namespace: "common", Key: "used"}},
	})

	require.NoError(t, err)
	require.Empty(t, edits)
	require.Len(t, rejected, 1)
	require.Equal(t, deadkey.StatusUsed, rejected[0].Status)
}

func TestPruneRejectsMissingNamespaceOrKey(t *testing.T) {
	a := newDeadKeyFixtureApp(t)

	out, err := a.DeadKeys.Prune(t.Context(), deadkey.PruneInput{Keys: []deadkey.PruneKey{
		{Namespace: "", Key: "unused"},
		{Namespace: "common", Key: ""},
	}})

	require.NoError(t, err)
	require.Len(t, out.Rejected, 2)
	require.Equal(t, "namespace and key are required", out.Rejected[0].Reason)
	require.Equal(t, "namespace and key are required", out.Rejected[1].Reason)
}

func TestPruneNotifiesLocaleUsageDeadKeyAndDiffUpdates(t *testing.T) {
	a := newDeadKeyFixtureApp(t)
	notifier := &recordingNotifier{}
	a.DeadKeys.Notifier = notifier

	_, err := a.DeadKeys.Prune(t.Context(), deadkey.PruneInput{Apply: true, Keys: []deadkey.PruneKey{{Namespace: "common", Key: "unused"}}})

	require.NoError(t, err)
	require.ElementsMatch(t, []string{
		resources.LocaleURI("en", "common"),
		resources.LocaleURI("fr", "common"),
		resources.UsageURI,
		resources.DeadKeysURI,
		resources.DiffURI,
	}, notifier.uris)
}

func TestPruneNotifiesWhenExactlyOneFileWasWritten(t *testing.T) {
	a := newDeadKeyFixtureApp(t)
	writeDeadKeyFile(t, a.ProjectRoot, ".i18n-mcp.json", `{
  "sourceLocale": "en",
  "targetLocales": [],
  "localeFiles": ["messages/{locale}/{namespace}.json"],
  "defaultNamespace": "common",
  "format": {"sortKeys": false, "indent": 2, "trailingNewline": true},
  "translation": {"mode": "agent"}
}
`)
	require.NoError(t, os.Remove(filepath.Join(a.ProjectRoot, "messages/fr/common.json")))
	notifier := &recordingNotifier{}
	a.DeadKeys.Notifier = notifier

	_, err := a.DeadKeys.Prune(t.Context(), deadkey.PruneInput{Apply: true, Keys: []deadkey.PruneKey{{Namespace: "common", Key: "unused"}}})

	require.NoError(t, err)
	require.ElementsMatch(t, []string{
		resources.LocaleURI("en", "common"),
		resources.UsageURI,
		resources.DeadKeysURI,
		resources.DiffURI,
	}, notifier.uris)
}

func TestPruneWriteRemovesExactKeys(t *testing.T) {
	a := newDeadKeyFixtureApp(t)
	out, err := a.DeadKeys.Prune(t.Context(), deadkey.PruneInput{Apply: true, Keys: []deadkey.PruneKey{{Namespace: "common", Key: "unused"}}})
	require.NoError(t, err)
	require.False(t, out.DryRun)
	require.Equal(t, 1, out.Pruned)
	require.NotEmpty(t, out.ChangedFiles)
	for _, file := range out.ChangedFiles {
		require.True(t, file.Written, file.Path)
		require.True(t, file.Changed, file.Path)
		require.NotEmpty(t, file.Diff, file.Path)
	}
	require.Contains(t, readDeadKeyFile(t, a.ProjectRoot, "messages/en/common.json"), "used")
	require.NotContains(t, readDeadKeyFile(t, a.ProjectRoot, "messages/en/common.json"), "unused")
}

func TestPruneDryRunFalseWithoutApplyDoesNotWrite(t *testing.T) {
	a := newDeadKeyFixtureApp(t)
	before := readDeadKeyFile(t, a.ProjectRoot, "messages/en/common.json")
	dryRun := false
	out, err := a.DeadKeys.Prune(t.Context(), deadkey.PruneInput{DryRun: &dryRun, Keys: []deadkey.PruneKey{{Namespace: "common", Key: "unused"}}})
	require.NoError(t, err)
	require.True(t, out.DryRun)
	require.Equal(t, before, readDeadKeyFile(t, a.ProjectRoot, "messages/en/common.json"))
}

func TestPruneApplyTrueWithDryRunTrueDoesNotWrite(t *testing.T) {
	a := newDeadKeyFixtureApp(t)
	before := readDeadKeyFile(t, a.ProjectRoot, "messages/en/common.json")
	dryRun := true
	out, err := a.DeadKeys.Prune(t.Context(), deadkey.PruneInput{Apply: true, DryRun: &dryRun, Keys: []deadkey.PruneKey{{Namespace: "common", Key: "unused"}}})
	require.NoError(t, err)
	require.True(t, out.DryRun)
	require.Equal(t, before, readDeadKeyFile(t, a.ProjectRoot, "messages/en/common.json"))
}

func TestPruneRemovesEmptyParentsOnly(t *testing.T) {
	a := newDeadKeyFixtureApp(t)
	writeDeadKeyFile(t, a.ProjectRoot, "messages/en/common.json", `{"used":"Used","nested":{"drop":{"leaf":"Leaf"},"keep":{"leaf":"Keep"}}}
`)
	writeDeadKeyFile(t, a.ProjectRoot, "messages/fr/common.json", `{"used":"Utilise","nested":{"drop":{"leaf":"Feuille"},"keep":{"leaf":"Garder"}}}
`)
	out, err := a.DeadKeys.Prune(t.Context(), deadkey.PruneInput{Apply: true, Keys: []deadkey.PruneKey{{Namespace: "common", Key: "nested.drop.leaf"}}})
	require.NoError(t, err)
	require.Empty(t, out.Rejected)
	contents := readDeadKeyFile(t, a.ProjectRoot, "messages/en/common.json")
	require.NotContains(t, contents, "drop")
	require.Contains(t, contents, "keep")
}

func newDeadKeyFixtureApp(t *testing.T) *app.App {
	t.Helper()
	root := t.TempDir()
	writeDeadKeyFile(t, root, ".i18n-mcp.json", `{
  "sourceLocale": "en",
  "targetLocales": ["fr"],
  "localeFiles": ["messages/{locale}/{namespace}.json"],
  "defaultNamespace": "common",
  "ignoredKeyPatterns": ["common.ignored"],
  "keptKeyPatterns": ["common.kept"],
  "dynamicKeyHints": ["routes.*"],
  "format": {"sortKeys": false, "indent": 2, "trailingNewline": true},
  "translation": {"mode": "agent"}
}
`)
	writeDeadKeyFile(t, root, "messages/en/common.json", `{
  "used": "Used",
  "unused": "Unused",
  "ignored": "Ignored",
  "kept": "Kept"
}
`)
	writeDeadKeyFile(t, root, "messages/fr/common.json", `{
  "used": "Utilise",
  "unused": "Inutilise",
  "ignored": "Ignore",
  "kept": "Conserve"
}
`)
	writeDeadKeyFile(t, root, "messages/en/routes.json", `{"show":"Show route"}
`)
	writeDeadKeyFile(t, root, "app/page.tsx", `export function Page() { return t("used") }
`)
	a, err := app.New(t.Context(), app.Options{ProjectRoot: root})
	require.NoError(t, err)
	return a
}

type recordingNotifier struct {
	uris []string
}

func (n *recordingNotifier) Updated(_ context.Context, uris ...string) {
	n.uris = append(n.uris, uris...)
}

func statusesByKey(report deadkey.Report) map[string]deadkey.Status {
	out := map[string]deadkey.Status{}
	for _, item := range report.Items {
		out[item.Namespace+"\x00"+item.Key] = item.Status
	}
	return out
}

func writeDeadKeyFile(t *testing.T, root string, relPath string, contents string) {
	t.Helper()
	abs := filepath.Join(root, relPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o700))
	require.NoError(t, os.WriteFile(abs, []byte(contents), 0o600))
}

func readDeadKeyFile(t *testing.T, root string, relPath string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, relPath))
	require.NoError(t, err)
	return string(data)
}
