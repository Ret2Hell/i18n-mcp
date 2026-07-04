package report_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/Ret2Hell/i18n-mcp/internal/report"
	"github.com/stretchr/testify/require"
)

func TestGenerateJSONStoresLatestAndSummarizesReports(t *testing.T) {
	a := newReportFixtureApp(t)

	out, err := a.Reports.Generate(t.Context(), report.GenerateInput{Format: report.FormatJSON, RefreshUsage: true})

	require.NoError(t, err)
	require.Equal(t, report.FormatJSON, out.Format)
	require.NotEmpty(t, out.Text)
	require.JSONEq(t, out.Text, out.Text)
	require.Equal(t, 2, out.Report.Summary.Locales)
	require.Equal(t, 1, out.Report.Summary.Namespaces)
	require.Equal(t, 2, out.Report.Summary.SourceKeys)
	require.Equal(t, 1, out.Report.Summary.TargetKeys)
	require.Equal(t, 1, out.Report.Summary.Missing)
	require.Equal(t, 1, out.Report.Summary.ProbablyUnused)

	latest, ok := a.Reports.Latest()
	require.True(t, ok)
	require.Equal(t, out.Format, latest.Format)
	require.Equal(t, out.Text, latest.Text)

	var decoded report.Report
	require.NoError(t, json.Unmarshal([]byte(out.Text), &decoded))
	require.Equal(t, out.Report.Summary, decoded.Summary)
}

func TestGenerateMarkdownAndUnsupportedFormat(t *testing.T) {
	a := newReportFixtureApp(t)

	out, err := a.Reports.Generate(t.Context(), report.GenerateInput{Format: report.FormatMarkdown, RefreshUsage: true})
	require.NoError(t, err)
	require.Equal(t, report.FormatMarkdown, out.Format)
	require.Contains(t, out.Text, "# i18n Audit Report")

	progress := &recordingProgress{}
	_, err = a.Reports.Generate(t.Context(), report.GenerateInput{Format: report.Format("xml"), RefreshUsage: true, Progress: progress})
	require.ErrorContains(t, err, `unsupported report format "xml"`)
	require.Equal(t, "rendering report", progress.lastMessage())
}

func TestGenerateEmitsStableProgressStages(t *testing.T) {
	a := newReportFixtureApp(t)
	progress := &recordingProgress{}

	_, err := a.Reports.Generate(t.Context(), report.GenerateInput{Format: report.FormatJSON, RefreshUsage: true, Progress: progress})

	require.NoError(t, err)
	require.Equal(t, []progressStep{
		{message: "loading locale inventory", current: 1, total: 5},
		{message: "computing translation diff", current: 2, total: 5},
		{message: "scanning source usage", current: 3, total: 5},
		{message: "building dead-key report", current: 4, total: 5},
		{message: "rendering report", current: 5, total: 5},
	}, progress.steps)
}

func TestEvaluatePolicyReportsEnabledFailuresOnly(t *testing.T) {
	r := report.Report{Summary: report.Summary{Missing: 2, Stale: 3, Invalid: 4, ProbablyUnused: 5}}
	policy := config.CIConfig{FailOnMissing: true, FailOnInvalid: true, FailOnDeadKeys: true}

	got := report.EvaluatePolicy(r, policy)

	require.Equal(t, []report.Failure{
		{Code: "missing", Message: "missing translations detected", Count: 2},
		{Code: "invalid", Message: "invalid translations detected", Count: 4},
		{Code: "dead_keys", Message: "probably unused keys detected", Count: 5},
	}, got)
}

func TestEvaluatePolicyIgnoresZeroCountsAndDisabledRules(t *testing.T) {
	r := report.Report{Summary: report.Summary{Missing: 0, Stale: 3, Invalid: 0, ProbablyUnused: 5}}
	policy := config.CIConfig{FailOnMissing: true, FailOnInvalid: true, FailOnDeadKeys: false}

	got := report.EvaluatePolicy(r, policy)

	require.Empty(t, got)
}

type progressStep struct {
	message string
	current int
	total   int
}

type recordingProgress struct {
	steps []progressStep
}

func (r *recordingProgress) Step(_ context.Context, message string, current int, total int) {
	r.steps = append(r.steps, progressStep{message: message, current: current, total: total})
}

func (r *recordingProgress) lastMessage() string {
	if len(r.steps) == 0 {
		return ""
	}
	return r.steps[len(r.steps)-1].message
}

func newReportFixtureApp(t *testing.T) *app.App {
	t.Helper()
	root := t.TempDir()
	writeReportFile(t, root, ".i18n-mcp.json", `{
  "sourceLocale": "en",
  "targetLocales": ["fr"],
  "localeFiles": ["messages/{locale}/{namespace}.json"],
  "defaultNamespace": "common",
  "format": {"sortKeys": false, "indent": 2, "trailingNewline": true},
  "translation": {"mode": "agent"}
}
`)
	writeReportFile(t, root, "messages/en/common.json", `{
  "used": "Used",
  "missing": "Missing"
}
`)
	writeReportFile(t, root, "messages/fr/common.json", `{
  "used": "Utilise"
}
`)
	writeReportFile(t, root, "app/page.tsx", `export function Page() { return t("used") }
`)
	a, err := app.New(t.Context(), app.Options{ProjectRoot: root})
	require.NoError(t, err)
	return a
}

func writeReportFile(t *testing.T, root string, relPath string, contents string) {
	t.Helper()
	abs := filepath.Join(root, relPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o700))
	require.NoError(t, os.WriteFile(abs, []byte(contents), 0o600))
}
