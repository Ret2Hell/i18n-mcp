package mcpserver_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func TestListAndGetCorePrompts(t *testing.T) {
	ctx, clientSession := newTestClientSession(t, makePromptCompletionFixture(t))

	list, err := clientSession.ListPrompts(ctx, &mcp.ListPromptsParams{})
	require.NoError(t, err)
	requirePromptNames(t, list, []string{
		"i18n_translate_batch",
		"i18n_review_translations",
		"i18n_audit_dead_keys",
		"i18n_project_bootstrap",
		"i18n_add_feature_keys",
	})

	prompt, err := clientSession.GetPrompt(ctx, &mcp.GetPromptParams{
		Name: "i18n_translate_batch",
		Arguments: map[string]string{
			"locale":  "fr",
			"batchId": "batch_example",
			"tone":    "concise",
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, prompt.Messages)
	textContent, ok := prompt.Messages[0].Content.(*mcp.TextContent)
	require.True(t, ok)
	require.Contains(t, textContent.Text, "batch_example")
}

func TestCompletionForLocalesNamespacesAndKeys(t *testing.T) {
	ctx, clientSession := newTestClientSession(t, makePromptCompletionFixture(t))

	locales, err := clientSession.Complete(ctx, &mcp.CompleteParams{
		Ref:      &mcp.CompleteReference{Type: "ref/prompt", Name: "i18n_translate_batch"},
		Argument: mcp.CompleteParamsArgument{Name: "locale", Value: "f"},
	})
	require.NoError(t, err)
	require.Contains(t, locales.Completion.Values, "fr")

	namespaces, err := clientSession.Complete(ctx, &mcp.CompleteParams{
		Ref:      &mcp.CompleteReference{Type: "ref/prompt", Name: "i18n_review_translations"},
		Argument: mcp.CompleteParamsArgument{Name: "namespace", Value: "a"},
	})
	require.NoError(t, err)
	require.Contains(t, namespaces.Completion.Values, "auth")

	keys, err := clientSession.Complete(ctx, &mcp.CompleteParams{
		Ref:      &mcp.CompleteReference{Type: "ref/prompt", Name: "i18n_review_translations"},
		Argument: mcp.CompleteParamsArgument{Name: "key", Value: "login"},
		Context:  &mcp.CompleteContext{Arguments: map[string]string{"namespace": "auth"}},
	})
	require.NoError(t, err)
	require.Contains(t, keys.Completion.Values, "login.title")
}

func TestCompletionForBatchIDs(t *testing.T) {
	ctx, clientSession := newTestClientSession(t, makePromptCompletionFixture(t))

	_, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.translation.plan", Arguments: map[string]any{}})
	require.NoError(t, err)

	batchIDs, err := clientSession.Complete(ctx, &mcp.CompleteParams{
		Ref:      &mcp.CompleteReference{Type: "ref/prompt", Name: "i18n_translate_batch"},
		Argument: mcp.CompleteParamsArgument{Name: "batchId", Value: "batch_"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, batchIDs.Completion.Values)
}

func makePromptCompletionFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writePromptFixtureFile(t, root, ".i18n-mcp.json", `{
  "sourceLocale": "en",
  "targetLocales": ["fr", "de"],
  "localeFiles": ["messages/{locale}/{namespace}.json"],
  "defaultNamespace": "common",
  "format": {"sortKeys": false, "indent": 2, "trailingNewline": true},
  "translation": {"mode": "agent"}
}
`)
	writePromptFixtureFile(t, root, "messages/en/auth.json", `{
  "login": {
    "title": "Log in",
    "subtitle": "Welcome back"
  }
}
`)
	writePromptFixtureFile(t, root, "messages/fr/auth.json", `{
  "login": {
    "title": "Connexion"
  }
}
`)
	writePromptFixtureFile(t, root, "messages/de/auth.json", `{
  "login": {
    "title": "Anmelden"
  }
}
`)
	return root
}

func writePromptFixtureFile(t *testing.T, root string, relPath string, contents string) {
	t.Helper()
	abs := filepath.Join(root, relPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o700))
	require.NoError(t, os.WriteFile(abs, []byte(contents), 0o600))
}

func requirePromptNames(t *testing.T, list *mcp.ListPromptsResult, names []string) {
	t.Helper()
	seen := map[string]bool{}
	for _, prompt := range list.Prompts {
		seen[prompt.Name] = true
	}
	for _, name := range names {
		require.True(t, seen[name], "missing prompt %s", name)
	}
}
