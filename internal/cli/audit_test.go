package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAuditCommandWritesReportToStdout(t *testing.T) {
	root := makeAuditFixture(t, false)
	opts := &RootOptions{}
	cmd := newRootCommand(opts)
	cmd.SetContext(t.Context())
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--project", root, "audit", "--output", "markdown"})

	err := cmd.Execute()
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "# i18n Audit Report")
	require.Empty(t, stderr.String())
}

func TestAuditCommandReturnsAuditErrorWhenPolicyFails(t *testing.T) {
	root := makeAuditFixture(t, true)
	opts := &RootOptions{}
	cmd := newRootCommand(opts)
	cmd.SetContext(t.Context())
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--project", root, "audit", "--output", "markdown"})

	err := cmd.Execute()
	require.Error(t, err)
	require.True(t, IsAuditError(err))
	require.Contains(t, stdout.String(), "# i18n Audit Report")
	require.Contains(t, stderr.String(), "audit failure: missing translations detected (1)")
}

func makeAuditFixture(t *testing.T, failOnMissing bool) string {
	t.Helper()
	root := t.TempDir()
	writeAuditFile(t, root, ".i18n-mcp.json", auditConfig(failOnMissing))
	writeAuditFile(t, root, "messages/en.json", `{
  "hello": "Hello",
  "bye": "Bye"
}`)
	fr := `{
  "hello": "Bonjour",
  "bye": "Au revoir"
}`
	if failOnMissing {
		fr = `{
  "hello": "Bonjour"
}`
	}
	writeAuditFile(t, root, "messages/fr.json", fr)
	return root
}

func auditConfig(failOnMissing bool) string {
	return `{
  "sourceLocale": "en",
  "targetLocales": ["fr"],
  "localeFiles": ["messages/{locale}.json"],
  "defaultNamespace": "common",
  "ci": {"failOnMissing": ` + strconv.FormatBool(failOnMissing) + `, "failOnInvalid": false}
}`
}

func writeAuditFile(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}
