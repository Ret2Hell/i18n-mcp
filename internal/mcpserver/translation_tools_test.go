package mcpserver_test

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/Ret2Hell/i18n-mcp/internal/state"
	"github.com/Ret2Hell/i18n-mcp/internal/translate"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func TestTranslationApplyToolDryRunByDefaultAndApplyWrites(t *testing.T) {
	root := makeTranslationApplyFixture(t)
	ctx, clientSession := newTestClientSession(t, root)

	args := map[string]any{
		"translations": []map[string]any{{
			"locale":      "fr",
			"namespace":   "common",
			"key":         "hello",
			"sourceValue": "Hello",
			"value":       "Bonjour",
		}},
	}
	dryRun, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.translation.apply", Arguments: args})
	require.NoError(t, err)
	require.False(t, dryRun.IsError)
	out := decodeApplyOutput(t, dryRun.StructuredContent)
	require.True(t, out.DryRun)
	require.Zero(t, out.Applied)
	require.Len(t, out.ChangedFiles, 1)
	require.False(t, out.ChangedFiles[0].Written)
	require.NoFileExists(t, filepath.Join(root, "messages", "fr.json"))
	require.NoFileExists(t, filepath.Join(root, state.DefaultStatePath))

	args["apply"] = true
	applied, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.translation.apply", Arguments: args})
	require.NoError(t, err)
	require.False(t, applied.IsError)
	out = decodeApplyOutput(t, applied.StructuredContent)
	require.False(t, out.DryRun)
	require.Equal(t, 1, out.Applied)
	require.Equal(t, 1, out.StateUpdates)
	require.FileExists(t, filepath.Join(root, "messages", "fr.json"))
	require.FileExists(t, filepath.Join(root, state.DefaultStatePath))
}

func TestTranslationApplyToolRejectsWithStructuredError(t *testing.T) {
	root := makeTranslationApplyFixture(t)
	ctx, clientSession := newTestClientSession(t, root)

	res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "i18n.translation.apply",
		Arguments: map[string]any{
			"apply": true,
			"translations": []map[string]any{{
				"locale":      "fr",
				"namespace":   "common",
				"key":         "hello",
				"sourceValue": "Old source",
				"value":       "Bonjour",
			}},
		},
	})
	require.NoError(t, err)
	require.True(t, res.IsError)
	out := decodeApplyOutput(t, res.StructuredContent)
	require.Len(t, out.Rejected, 1)
	require.NoFileExists(t, filepath.Join(root, "messages", "fr.json"))
	require.NoFileExists(t, filepath.Join(root, state.DefaultStatePath))
}

func TestTranslationPlanValidateAndApplyTools(t *testing.T) {
	ctx, clientSession := newTestClientSession(t, makeTranslationMCPFixture(t))

	plan, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.translation.plan", Arguments: map[string]any{}})
	require.NoError(t, err)
	require.False(t, plan.IsError)

	translationArgs := map[string]any{
		"translations": []map[string]any{{
			"locale":      "fr",
			"namespace":   "auth",
			"key":         "login.title",
			"sourceValue": "Log in",
			"value":       "Connexion",
		}},
	}
	validate, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.translation.validate", Arguments: translationArgs})
	require.NoError(t, err)
	require.False(t, validate.IsError)

	apply, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.translation.apply", Arguments: translationArgs})
	require.NoError(t, err)
	require.False(t, apply.IsError)
	out := decodeApplyOutput(t, apply.StructuredContent)
	require.True(t, out.DryRun)
	require.Len(t, out.ChangedFiles, 1)
}

func decodeApplyOutput(t *testing.T, structured any) translate.ApplyOutput {
	t.Helper()
	data, err := json.Marshal(structured)
	require.NoError(t, err)
	var out translate.ApplyOutput
	require.NoError(t, json.Unmarshal(data, &out))
	return out
}

func makeTranslationApplyFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, root, ".i18n-mcp.json", `{
  "sourceLocale": "en",
  "targetLocales": ["fr"],
  "localeFiles": ["messages/{locale}.json"],
  "defaultNamespace": "common"
}`)
	writeFile(t, root, "messages/en.json", `{
  "hello": "Hello"
}`)
	return root
}

func makeTranslationMCPFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, root, ".i18n-mcp.json", `{
  "sourceLocale": "en",
  "targetLocales": ["fr"],
  "localeFiles": ["messages/{locale}/{namespace}.json"],
  "defaultNamespace": "common",
  "format": {"sortKeys": false, "indent": 2, "trailingNewline": true},
  "translation": {"mode": "agent"}
}`)
	writeFile(t, root, "messages/en/auth.json", `{
  "login": {
    "title": "Log in",
    "subtitle": "Welcome {name}"
  }
}`)
	writeFile(t, root, "messages/fr/auth.json", `{
  "login": {
    "subtitle": "Bienvenue {name}"
  }
}`)

	stateFile := state.NewFile("en", translationMCPTestTime())
	stateFile.Entries[state.EntryKey("fr", "auth", "login.subtitle")] = state.Entry{
		Locale:             "fr",
		Namespace:          "auth",
		Key:                "login.subtitle",
		SourceHash:         state.SourceHash("Welcome {name}"),
		TranslatedFromHash: state.SourceHash("Old welcome {name}"),
		TargetHash:         state.TargetHash("Bienvenue {name}"),
		Status:             state.StatusCurrent,
		UpdatedAt:          translationMCPTestTime(),
		UpdatedBy:          "fixture",
	}
	data, err := json.MarshalIndent(stateFile, "", "  ")
	require.NoError(t, err)
	writeFile(t, root, state.DefaultStatePath, string(data)+"\n")
	return root
}

func translationMCPTestTime() time.Time {
	return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
}
