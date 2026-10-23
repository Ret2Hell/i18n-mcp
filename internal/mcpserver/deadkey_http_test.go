package mcpserver_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/mcpserver"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

// These transport tests explicitly register dormant dead-key handlers in a
// test-only server; the production HTTP server does not expose them.
func TestStatelessHTTPReturnsInputRequiredMRTRResult(t *testing.T) {
	root := makeDeadKeyMCPFixture(t)
	httpServer := newLegacyDeadKeyHTTPServer(t, root)
	body := `{
		"jsonrpc":"2.0",
		"id":3,
		"method":"tools/call",
		"params":{
			"name":"i18n.keys.prune",
			"arguments":{
				"apply":true,
				"confirmWithClient":true,
				"keys":[{"namespace":"common","key":"unused"}]
			},
			"_meta":{
				"io.modelcontextprotocol/protocolVersion":"2026-07-28",
				"io.modelcontextprotocol/clientInfo":{"name":"test-client","version":"v0.0.0"},
				"io.modelcontextprotocol/clientCapabilities":{"elicitation":{"form":{}}}
			}
		}
	}`
	req := deadKeyHTTPRequest(t, httpServer.URL+"/mcp", body, "2026-07-28", "tools/call")
	req.Header.Set("Mcp-Name", "i18n.keys.prune")
	response, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	responseBody, readErr := io.ReadAll(response.Body)
	require.NoError(t, response.Body.Close())
	require.NoError(t, readErr)
	require.Equal(t, http.StatusOK, response.StatusCode, string(responseBody))

	var message struct {
		Result struct {
			ResultType    string                     `json:"resultType"`
			InputRequests map[string]json.RawMessage `json:"inputRequests"`
			RequestState  string                     `json:"requestState"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal(responseBody, &message))
	require.Equal(t, "input_required", message.Result.ResultType)
	require.Contains(t, message.Result.InputRequests, "prune_confirmation")
	require.NotEmpty(t, message.Result.RequestState)
}

func TestStatelessHTTPCompletesPruneConfirmationMRTR(t *testing.T) {
	root := makeDeadKeyMCPFixture(t)
	httpServer := newLegacyDeadKeyHTTPServer(t, root)
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

func newLegacyDeadKeyHTTPServer(t *testing.T, root string) *httptest.Server {
	t.Helper()
	application, err := app.New(t.Context(), app.Options{ProjectRoot: root, LogLevel: "error"})
	require.NoError(t, err)
	server := mcpserver.New(application)
	mcpserver.RegisterDeadKeyFeaturesForTest(server, application)
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
	httpServer := httptest.NewServer(handler)
	t.Cleanup(httpServer.Close)
	return httpServer
}

func deadKeyHTTPRequest(t *testing.T, endpoint, body, version, method string) *http.Request {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, endpoint, strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("MCP-Protocol-Version", version)
	req.Header.Set("Mcp-Method", method)
	return req
}
