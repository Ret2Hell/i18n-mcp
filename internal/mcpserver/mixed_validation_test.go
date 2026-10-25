package mcpserver_test

import (
	"github.com/Ret2Hell/i18n-mcp/internal/diff"
	"github.com/Ret2Hell/i18n-mcp/internal/translate"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMixedFormatsAndSourceFailuresRemainDistinct(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".i18n-mcp.json", `{"sourceLocale":"en","targetLocales":["fr","de"],"localeFiles":["locales/{locale}.json"],"validation":{"messageFormat":"icu","overrides":[{"keyPrefix":"raw.","messageFormat":"raw"},{"key":"raw.name","messageFormat":"i18next"}]}}`)
	writeFile(t, root, "locales/en.json", `{"raw":{"link":"{link:https://example.test;Docs}","name":"{{name}}"},"strict":"{name}","badSource":"{link:custom}"}`)
	writeFile(t, root, "locales/fr.json", `{"raw":{"link":"{link:https://example.test;Documents}","name":"{{name}}"},"strict":"Bonjour","badSource":"{link:personnalisé}"}`)
	writeFile(t, root, "locales/de.json", `{"raw":{"link":"{link:https://example.test;Dokumente}","name":"{{name}}"},"strict":"Hallo {name}","badSource":"{link:anderes}"}`)
	ctx, session := newTestClientSession(t, root)
	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.keys.diff"})
	require.NoError(t, err)
	require.False(t, res.IsError)
	var out struct {
		Report diff.Report `json:"report"`
	}
	unmarshalStructuredContent(t, res.StructuredContent, &out)
	require.Equal(t, 1, out.Report.Summary.Invalid)
	require.Equal(t, 2, out.Report.Summary.SourceInvalid)
	require.Equal(t, 1, out.Report.SourceIssueCount)
	require.Len(t, out.Report.SourceIssues, 1)
	require.Equal(t, "en", out.Report.SourceIssues[0].Locale)
	for _, item := range out.Report.Items {
		if item.Status == diff.SourceInvalid {
			require.Empty(t, item.Validation)
			require.NotEmpty(t, item.SourceValidation)
			require.Equal(t, "source", item.SourceValidation[0].Scope)
		}
	}
	res, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.translation.plan", Arguments: map[string]any{"statuses": []string{"unknown"}}})
	require.NoError(t, err)
	require.False(t, res.IsError)
	var planned struct {
		Batch translate.Batch `json:"batch"`
	}
	unmarshalStructuredContent(t, res.StructuredContent, &planned)
	require.Equal(t, 1, planned.Batch.SourceIssueCount)
	for _, item := range planned.Batch.Items {
		if item.Key == "raw.link" {
			require.Equal(t, "raw", item.MessageFormat)
			require.Empty(t, item.Placeholders)
			require.NotEmpty(t, item.Notes)
		}
	}
	// A policy change invalidates an existing batch even when source text is unchanged.
	writeFile(t, root, ".i18n-mcp.json", `{"sourceLocale":"en","targetLocales":["fr","de"],"localeFiles":["locales/{locale}.json"],"validation":{"messageFormat":"interpolation"}}`)
	res, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.translation.validate", Arguments: map[string]any{"batchId": planned.Batch.BatchID, "translations": []any{}}})
	require.NoError(t, err)
	require.True(t, res.IsError)
}
