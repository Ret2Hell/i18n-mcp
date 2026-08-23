package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/mcpserver"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

const latestProtocolVersion = "2026-07-28"

type staticServerProvider struct {
	server *mcp.Server
}

func (p staticServerProvider) ServerForRequest(*http.Request) *mcp.Server {
	return p.server
}

func TestStatelessHTTPClientCallsToolWithoutInitialize(t *testing.T) {
	httpServer := newStatelessTestServer(t)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.0"}, nil)
	session, err := client.Connect(t.Context(), &mcp.StreamableClientTransport{
		Endpoint:             httpServer.URL + "/mcp",
		DisableStandaloneSSE: true,
	}, nil)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, session.Close()) })

	require.Equal(t, latestProtocolVersion, session.InitializeResult().ProtocolVersion)
	result, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: "i18n.health"})
	require.NoError(t, err)
	require.False(t, result.IsError)
}

func TestStatelessHTTPCompletesPruneConfirmationMRTR(t *testing.T) {
	root := statelessPruneFixture(t)
	httpServer := newStatelessTestServerWithRoot(t, root)
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.0"}, &mcp.ClientOptions{
		ElicitationHandler: func(context.Context, *mcp.ElicitRequest) (*mcp.ElicitResult, error) {
			return &mcp.ElicitResult{Action: "accept", Content: map[string]any{"confirm": true}}, nil
		},
	})
	session, err := client.Connect(t.Context(), &mcp.StreamableClientTransport{
		Endpoint:             httpServer.URL + "/mcp",
		DisableStandaloneSSE: true,
	}, nil)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, session.Close()) })

	result, err := session.CallTool(t.Context(), &mcp.CallToolParams{
		Name: "i18n.keys.prune",
		Arguments: map[string]any{
			"apply":             true,
			"confirmWithClient": true,
			"keys":              []map[string]any{{"namespace": "common", "key": "unused"}},
		},
	})
	require.NoError(t, err)
	require.False(t, result.IsError)
	contents, err := os.ReadFile(filepath.Join(root, "messages/en/common.json"))
	require.NoError(t, err)
	require.NotContains(t, string(contents), "unused")
}

func TestStatelessHTTPDoesNotCreateProtocolSession(t *testing.T) {
	httpServer := newStatelessTestServer(t)
	body := []byte(`{
		"jsonrpc":"2.0",
		"id":1,
		"method":"server/discover",
		"params":{"_meta":{
			"io.modelcontextprotocol/protocolVersion":"2026-07-28",
			"io.modelcontextprotocol/clientInfo":{"name":"test-client","version":"v0.0.0"},
			"io.modelcontextprotocol/clientCapabilities":{}
		}}
	}`)
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, httpServer.URL+"/mcp", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("MCP-Protocol-Version", latestProtocolVersion)
	req.Header.Set("Mcp-Method", "server/discover")
	req.Header.Set("Mcp-Session-Id", "legacy-session")

	response, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	responseBody, readErr := io.ReadAll(response.Body)
	require.NoError(t, response.Body.Close())
	require.NoError(t, readErr)
	require.Equal(t, http.StatusOK, response.StatusCode, string(responseBody))
	require.Empty(t, response.Header.Get("Mcp-Session-Id"))

	var message struct {
		Result struct {
			ResultType string `json:"resultType"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal(responseBody, &message))
	require.Equal(t, "complete", message.Result.ResultType)
}

func TestStatelessHTTPEndpointIsPostOnly(t *testing.T) {
	httpServer := newStatelessTestServer(t)

	for _, method := range []string{http.MethodGet, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			req, err := http.NewRequestWithContext(t.Context(), method, httpServer.URL+"/mcp", nil)
			require.NoError(t, err)
			response, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			require.NoError(t, response.Body.Close())
			require.Equal(t, http.StatusMethodNotAllowed, response.StatusCode)
			require.Equal(t, "POST", response.Header.Get("Allow"))
		})
	}
}

func newStatelessTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	return newStatelessTestServerWithRoot(t, t.TempDir())
}

func newStatelessTestServerWithRoot(t *testing.T, root string) *httptest.Server {
	t.Helper()
	application, err := app.New(t.Context(), app.Options{ProjectRoot: root, LogLevel: "error"})
	require.NoError(t, err)
	server := mcpserver.New(application)
	handler, err := newHandler(Config{JSONResponse: true}, staticServerProvider{server: server}, application.Logger)
	require.NoError(t, err)
	httpServer := httptest.NewServer(handler)
	t.Cleanup(httpServer.Close)
	return httpServer
}

func statelessPruneFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		".i18n-mcp.json": `{
			"sourceLocale":"en",
			"targetLocales":["fr"],
			"localeFiles":["messages/{locale}/{namespace}.json"],
			"defaultNamespace":"common",
			"translation":{"mode":"agent"}
		}`,
		"messages/en/common.json": `{"used":"Used","unused":"Unused"}`,
		"messages/fr/common.json": `{"used":"Utilise","unused":"Inutilise"}`,
		"app/page.tsx":            `export const value = t("used")`,
	}
	for path, content := range files {
		absolute := filepath.Join(root, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(absolute), 0o700))
		require.NoError(t, os.WriteFile(absolute, []byte(content), 0o600))
	}
	return root
}
