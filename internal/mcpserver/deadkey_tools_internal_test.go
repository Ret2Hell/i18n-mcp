package mcpserver

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/deadkey"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func TestPruneConfirmationRejectsInvalidRequestState(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(t *testing.T, root string, in *deadkey.PruneInput, state *string)
		error  string
	}{
		{
			name: "modified signature",
			mutate: func(_ *testing.T, _ string, _ *deadkey.PruneInput, state *string) {
				*state = "A" + (*state)[1:]
			},
			error: "signature",
		},
		{
			name: "changed input",
			mutate: func(_ *testing.T, _ string, in *deadkey.PruneInput, _ *string) {
				in.AllowUnsafe = true
			},
			error: "does not match",
		},
		{
			name: "changed plan",
			mutate: func(t *testing.T, root string, _ *deadkey.PruneInput, _ *string) {
				writeInternalPruneFile(t, root, "messages/fr/common.json", `{"used":"Utilise","unused":"Modifie"}`)
			},
			error: "does not match",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := internalPruneFixture(t)
			application, err := app.New(t.Context(), app.Options{ProjectRoot: root, LogLevel: "error"})
			require.NoError(t, err)
			handler := keysPruneTool(application)
			in := deadkey.PruneInput{
				Apply:             true,
				ConfirmWithClient: true,
				Keys:              []deadkey.PruneKey{{Namespace: "common", Key: "unused"}},
			}
			first := pruneCapableRequest()
			result, _, err := handler(t.Context(), first, in)
			require.NoError(t, err)
			require.NotEmpty(t, result.RequestState)

			state := result.RequestState
			tt.mutate(t, root, &in, &state)
			retry := pruneCapableRequest()
			retry.Params.RequestState = state
			retry.Params.InputResponses = mcp.InputResponseMap{
				pruneConfirmationRequestID: &mcp.ElicitResult{Action: "accept", Content: map[string]any{"confirm": true}},
			}
			_, _, err = handler(t.Context(), retry, in)
			require.ErrorContains(t, err, tt.error)
		})
	}
}

func TestValidatePruneConfirmation(t *testing.T) {
	tests := []struct {
		name        string
		responses   mcp.InputResponseMap
		confirmed   bool
		errorString string
	}{
		{
			name: "accepted",
			responses: mcp.InputResponseMap{
				pruneConfirmationRequestID: &mcp.ElicitResult{Action: "accept", Content: map[string]any{"confirm": true}},
			},
			confirmed: true,
		},
		{
			name: "declined",
			responses: mcp.InputResponseMap{
				pruneConfirmationRequestID: &mcp.ElicitResult{Action: "decline"},
			},
		},
		{
			name: "false",
			responses: mcp.InputResponseMap{
				pruneConfirmationRequestID: &mcp.ElicitResult{Action: "accept", Content: map[string]any{"confirm": false}},
			},
		},
		{
			name:        "missing response",
			errorString: "response is missing",
		},
		{
			name: "missing confirmation value",
			responses: mcp.InputResponseMap{
				pruneConfirmationRequestID: &mcp.ElicitResult{Action: "accept"},
			},
			errorString: "missing confirm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{InputResponses: tt.responses}}

			confirmed, err := validatePruneConfirmation(req)

			require.Equal(t, tt.confirmed, confirmed)
			if tt.errorString == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tt.errorString)
			}
		})
	}
}

func pruneCapableRequest() *mcp.CallToolRequest {
	return &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Meta: mcp.Meta{
		mcp.MetaKeyProtocolVersion: "2026-07-28",
		mcp.MetaKeyClientCapabilities: map[string]any{
			"elicitation": map[string]any{"form": map[string]any{}},
		},
	}}}
}

func internalPruneFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeInternalPruneFile(t, root, ".i18n-mcp.json", `{
		"sourceLocale":"en",
		"targetLocales":["fr"],
		"localeFiles":["messages/{locale}/{namespace}.json"],
		"defaultNamespace":"common",
		"translation":{"mode":"agent"}
	}`)
	writeInternalPruneFile(t, root, "messages/en/common.json", `{"used":"Used","unused":"Unused"}`)
	writeInternalPruneFile(t, root, "messages/fr/common.json", `{"used":"Utilise","unused":"Inutilise"}`)
	writeInternalPruneFile(t, root, "app/page.tsx", `export const value = t("used")`)
	return root
}

func writeInternalPruneFile(t *testing.T, root string, path string, content string) {
	t.Helper()
	absolute := filepath.Join(root, path)
	require.NoError(t, os.MkdirAll(filepath.Dir(absolute), 0o700))
	require.NoError(t, os.WriteFile(absolute, []byte(content), 0o600))
}
