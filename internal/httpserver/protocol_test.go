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
	"strings"
	"testing"
	"time"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/mcpserver"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

const (
	latestProtocolVersion = "2026-07-28"
	modernDiscoverBody    = `{
		"jsonrpc":"2.0",
		"id":1,
		"method":"server/discover",
		"params":{"_meta":{
			"io.modelcontextprotocol/protocolVersion":"2026-07-28",
			"io.modelcontextprotocol/clientInfo":{"name":"test-client","version":"v0.0.0"},
			"io.modelcontextprotocol/clientCapabilities":{}
		}}
	}`
	modernHealthBody = `{
		"jsonrpc":"2.0",
		"id":2,
		"method":"tools/call",
		"params":{
			"name":"i18n.health",
			"arguments":{},
			"_meta":{
				"io.modelcontextprotocol/protocolVersion":"2026-07-28",
				"io.modelcontextprotocol/clientInfo":{"name":"test-client","version":"v0.0.0"},
				"io.modelcontextprotocol/clientCapabilities":{}
			}
		}
	}`
)

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
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, httpServer.URL+"/mcp", bytes.NewBufferString(modernDiscoverBody))
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
			TTLMs      int    `json:"ttlMs"`
			CacheScope string `json:"cacheScope"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal(responseBody, &message))
	require.Equal(t, "complete", message.Result.ResultType)
	require.Equal(t, 300000, message.Result.TTLMs)
	require.Equal(t, "public", message.Result.CacheScope)
}

func TestStatelessHTTPDeliversResourceSubscriptions(t *testing.T) {
	application, err := app.New(t.Context(), app.Options{ProjectRoot: t.TempDir(), LogLevel: "error"})
	require.NoError(t, err)
	server := mcpserver.New(application)
	handler, err := newHandler(Config{}, staticServerProvider{server: server}, application.Logger)
	require.NoError(t, err)
	httpServer := httptest.NewServer(handler)

	updates := make(chan string, 1)
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.0"}, &mcp.ClientOptions{
		ResourceUpdatedHandler: func(_ context.Context, req *mcp.ResourceUpdatedNotificationRequest) {
			updates <- req.Params.URI
		},
	})
	session, err := client.Connect(t.Context(), &mcp.StreamableClientTransport{Endpoint: httpServer.URL + "/mcp"}, nil)
	require.NoError(t, err)

	uri := "i18n://analysis/diff"
	require.NoError(t, session.Subscribe(t.Context(), &mcp.SubscribeParams{URI: uri}))
	require.NoError(t, server.ResourceUpdated(t.Context(), &mcp.ResourceUpdatedNotificationParams{URI: uri}))
	select {
	case got := <-updates:
		require.Equal(t, uri, got)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for resource update")
	}

	require.NoError(t, session.Close())
	httpServer.CloseClientConnections()
	require.NoError(t, server.ResourceUpdated(t.Context(), &mcp.ResourceUpdatedNotificationParams{URI: uri}))
	httpServer.Close()
}

func TestMCPOriginProtection(t *testing.T) {
	tests := []struct {
		name           string
		origin         func(serverURL string) string
		trustedOrigins []string
		wantStatus     int
	}{
		{name: "native client without origin", wantStatus: http.StatusOK},
		{name: "same origin", origin: func(serverURL string) string { return serverURL }, wantStatus: http.StatusOK},
		{name: "trusted origin", origin: func(string) string { return "https://inspector.example" }, trustedOrigins: []string{"https://inspector.example"}, wantStatus: http.StatusOK},
		{name: "untrusted origin", origin: func(string) string { return "https://attacker.example" }, wantStatus: http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			application, err := app.New(t.Context(), app.Options{ProjectRoot: t.TempDir(), LogLevel: "error"})
			require.NoError(t, err)
			server := mcpserver.New(application)
			handler, err := newHandler(Config{JSONResponse: true, TrustedOrigins: tt.trustedOrigins}, staticServerProvider{server: server}, application.Logger)
			require.NoError(t, err)
			httpServer := httptest.NewServer(handler)
			t.Cleanup(httpServer.Close)

			req := modernRequest(t, httpServer.URL+"/mcp", modernDiscoverBody, latestProtocolVersion, "server/discover")
			if tt.origin != nil {
				req.Header.Set("Origin", tt.origin(httpServer.URL))
			}
			response, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			require.NoError(t, response.Body.Close())
			require.Equal(t, tt.wantStatus, response.StatusCode)
		})
	}
}

func TestMCPRejectsMalformedTrustedOrigin(t *testing.T) {
	application, err := app.New(t.Context(), app.Options{ProjectRoot: t.TempDir(), LogLevel: "error"})
	require.NoError(t, err)
	server := mcpserver.New(application)
	_, err = newHandler(Config{TrustedOrigins: []string{"not-an-origin"}}, staticServerProvider{server: server}, application.Logger)
	require.ErrorContains(t, err, "trusted origin")
}

func TestModernHTTPHeaderValidation(t *testing.T) {
	httpServer := newStatelessTestServer(t)
	tests := []struct {
		name       string
		body       string
		version    string
		method     string
		mutate     func(*http.Request)
		wantStatus int
		wantCode   int
	}{
		{
			name:       "missing method header",
			body:       modernDiscoverBody,
			version:    latestProtocolVersion,
			method:     "server/discover",
			mutate:     func(req *http.Request) { req.Header.Del("Mcp-Method") },
			wantStatus: http.StatusBadRequest,
			wantCode:   -32020,
		},
		{
			name:       "mismatched method header",
			body:       modernDiscoverBody,
			version:    latestProtocolVersion,
			method:     "tools/list",
			wantStatus: http.StatusBadRequest,
			wantCode:   -32020,
		},
		{
			name:       "missing name header",
			body:       modernHealthBody,
			version:    latestProtocolVersion,
			method:     "tools/call",
			wantStatus: http.StatusBadRequest,
			wantCode:   -32020,
		},
		{
			name:       "mismatched name header",
			body:       modernHealthBody,
			version:    latestProtocolVersion,
			method:     "tools/call",
			mutate:     func(req *http.Request) { req.Header.Set("Mcp-Name", "i18n.keys.diff") },
			wantStatus: http.StatusBadRequest,
			wantCode:   -32020,
		},
		{
			name:       "unsupported protocol version",
			body:       strings.ReplaceAll(modernDiscoverBody, latestProtocolVersion, "2099-01-01"),
			version:    "2099-01-01",
			method:     "server/discover",
			wantStatus: http.StatusBadRequest,
			wantCode:   -32022,
		},
		{
			name:       "unknown method",
			body:       strings.Replace(modernDiscoverBody, "server/discover", "unknown/method", 1),
			version:    latestProtocolVersion,
			method:     "unknown/method",
			wantStatus: http.StatusNotFound,
			wantCode:   -32601,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := modernRequest(t, httpServer.URL+"/mcp", tt.body, tt.version, tt.method)
			if tt.mutate != nil {
				tt.mutate(req)
			}
			response, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			body, readErr := io.ReadAll(response.Body)
			require.NoError(t, response.Body.Close())
			require.NoError(t, readErr)
			require.Equal(t, tt.wantStatus, response.StatusCode, string(body))
			var message struct {
				Error struct {
					Code int `json:"code"`
				} `json:"error"`
			}
			require.NoError(t, json.Unmarshal(body, &message))
			require.Equal(t, tt.wantCode, message.Error.Code)
		})
	}
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

func modernRequest(t *testing.T, endpoint string, body string, version string, method string) *http.Request {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, endpoint, bytes.NewBufferString(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("MCP-Protocol-Version", version)
	req.Header.Set("Mcp-Method", method)
	return req
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
