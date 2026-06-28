package mcpserver_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func TestUsageDeadReportAndPruneTools(t *testing.T) {
	ctx, clientSession := newTestClientSession(t, makeDeadKeyMCPFixture(t))

	usage, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.keys.usage_scan", Arguments: map[string]any{}})
	require.NoError(t, err)
	require.False(t, usage.IsError)

	report, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.keys.dead_report", Arguments: map[string]any{"includeUsed": true}})
	require.NoError(t, err)
	require.False(t, report.IsError)

	prune, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.keys.prune", Arguments: map[string]any{
		"keys": []map[string]any{{"namespace": "common", "key": "unused"}},
	}})
	require.NoError(t, err)
	require.False(t, prune.IsError)
}

func TestAnalysisUsageAndDeadKeyResources(t *testing.T) {
	ctx, clientSession := newTestClientSession(t, makeDeadKeyMCPFixture(t))

	_, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.keys.usage_scan", Arguments: map[string]any{}})
	require.NoError(t, err)
	_, err = clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.keys.dead_report", Arguments: map[string]any{"includeUsed": true}})
	require.NoError(t, err)

	usage, err := clientSession.ReadResource(ctx, &mcp.ReadResourceParams{URI: "i18n://analysis/usage"})
	require.NoError(t, err)
	require.Len(t, usage.Contents, 1)
	var usagePayload map[string]any
	require.NoError(t, json.Unmarshal([]byte(usage.Contents[0].Text), &usagePayload))
	require.Contains(t, usagePayload, "usages")

	deadKeys, err := clientSession.ReadResource(ctx, &mcp.ReadResourceParams{URI: "i18n://analysis/dead-keys"})
	require.NoError(t, err)
	require.Len(t, deadKeys.Contents, 1)
	var deadKeyPayload map[string]any
	require.NoError(t, json.Unmarshal([]byte(deadKeys.Contents[0].Text), &deadKeyPayload))
	require.Contains(t, deadKeyPayload, "items")
}

func makeDeadKeyMCPFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeMCPDeadKeyFile(t, root, ".i18n-mcp.json", `{
  "sourceLocale": "en",
  "targetLocales": ["fr"],
  "localeFiles": ["messages/{locale}/{namespace}.json"],
  "defaultNamespace": "common",
  "dynamicKeyHints": ["routes.*"],
  "format": {"sortKeys": false, "indent": 2, "trailingNewline": true},
  "translation": {"mode": "agent"}
}
`)
	writeMCPDeadKeyFile(t, root, "messages/en/common.json", `{"used":"Used","unused":"Unused"}
`)
	writeMCPDeadKeyFile(t, root, "messages/fr/common.json", `{"used":"Utilise","unused":"Inutilise"}
`)
	writeMCPDeadKeyFile(t, root, "app/page.tsx", `export function Page() { return t("used") }
`)
	return root
}

func writeMCPDeadKeyFile(t *testing.T, root string, relPath string, contents string) {
	t.Helper()
	abs := filepath.Join(root, relPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o700))
	require.NoError(t, os.WriteFile(abs, []byte(contents), 0o600))
}
