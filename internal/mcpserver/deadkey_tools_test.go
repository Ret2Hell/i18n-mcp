package mcpserver_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/mcpserver"
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

func TestPruneConfirmWithClientWithoutApplyDoesNotElicit(t *testing.T) {
	elicited := false
	ctx, clientSession := newPruneConfirmClientSession(t, makeDeadKeyMCPFixture(t), func(context.Context, *mcp.ElicitRequest) (*mcp.ElicitResult, error) {
		elicited = true
		return &mcp.ElicitResult{Action: "accept", Content: map[string]any{"confirm": true}}, nil
	})

	res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.keys.prune", Arguments: map[string]any{
		"confirmWithClient": true,
		"keys":              []map[string]any{{"namespace": "common", "key": "unused"}},
	}})

	require.NoError(t, err)
	require.False(t, res.IsError)
	require.False(t, elicited)
}

func TestPruneConfirmWithClientRequiresCapability(t *testing.T) {
	ctx, clientSession := newTestClientSession(t, makeDeadKeyMCPFixture(t))

	res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.keys.prune", Arguments: map[string]any{
		"apply":             true,
		"confirmWithClient": true,
		"keys":              []map[string]any{{"namespace": "common", "key": "unused"}},
	}})

	require.NoError(t, err)
	require.True(t, res.IsError)
}

func TestPruneConfirmWithClientAcceptApplies(t *testing.T) {
	root := makeDeadKeyMCPFixture(t)
	ctx, clientSession := newPruneConfirmClientSession(t, root, func(context.Context, *mcp.ElicitRequest) (*mcp.ElicitResult, error) {
		return &mcp.ElicitResult{Action: "accept", Content: map[string]any{"confirm": true}}, nil
	})

	res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.keys.prune", Arguments: map[string]any{
		"apply":             true,
		"confirmWithClient": true,
		"keys":              []map[string]any{{"namespace": "common", "key": "unused"}},
	}})

	require.NoError(t, err)
	require.False(t, res.IsError)
	require.NotContains(t, readMCPDeadKeyFile(t, root, "messages/en/common.json"), "unused")
}

func TestPruneConfirmWithClientDeclineCancelOrFalsePreventsWrites(t *testing.T) {
	for _, tc := range []struct {
		name    string
		action  string
		content map[string]any
	}{
		{name: "decline", action: "decline"},
		{name: "cancel", action: "cancel"},
		{name: "false", action: "accept", content: map[string]any{"confirm": false}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := makeDeadKeyMCPFixture(t)
			before := readMCPDeadKeyFile(t, root, "messages/en/common.json")
			ctx, clientSession := newPruneConfirmClientSession(t, root, func(context.Context, *mcp.ElicitRequest) (*mcp.ElicitResult, error) {
				return &mcp.ElicitResult{Action: tc.action, Content: tc.content}, nil
			})

			res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.keys.prune", Arguments: map[string]any{
				"apply":             true,
				"confirmWithClient": true,
				"keys":              []map[string]any{{"namespace": "common", "key": "unused"}},
			}})

			require.NoError(t, err)
			require.True(t, res.IsError)
			require.Equal(t, before, readMCPDeadKeyFile(t, root, "messages/en/common.json"))
		})
	}
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

func readMCPDeadKeyFile(t *testing.T, root string, relPath string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, relPath))
	require.NoError(t, err)
	return string(data)
}

func newPruneConfirmClientSession(
	t *testing.T,
	root string,
	handler func(context.Context, *mcp.ElicitRequest) (*mcp.ElicitResult, error),
) (context.Context, *mcp.ClientSession) {
	t.Helper()
	ctx := t.Context()
	application, err := app.New(ctx, app.Options{ProjectRoot: root, LogLevel: "error"})
	require.NoError(t, err)

	server := mcpserver.New(application)
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.0"}, &mcp.ClientOptions{
		ElicitationHandler: handler,
	})
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	serverSession, err := server.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, serverSession.Close()) })

	clientSession, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, clientSession.Close()) })

	return ctx, clientSession
}
