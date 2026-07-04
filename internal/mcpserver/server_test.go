package mcpserver_test

import (
	"encoding/json"
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/mcpserver"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func TestResourceSubscriptions(t *testing.T) {
	ctx := t.Context()
	application, err := app.New(ctx, app.Options{ProjectRoot: t.TempDir(), LogLevel: "error"})
	require.NoError(t, err)

	server := mcpserver.New(application)
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.0"}, nil)
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	serverSession, err := server.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, serverSession.Close()) })

	clientSession, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, clientSession.Close()) })

	capabilities := clientSession.InitializeResult().Capabilities
	require.NotNil(t, capabilities.Resources)
	require.True(t, capabilities.Resources.Subscribe)

	validURI := "i18n://locales/en/common.json"
	require.NoError(t, clientSession.Subscribe(ctx, &mcp.SubscribeParams{URI: validURI}))
	require.ElementsMatch(t, []string{validURI}, application.Subscriptions.URIsForSession(clientSession.ID()))

	require.Error(t, clientSession.Subscribe(ctx, &mcp.SubscribeParams{URI: "file:///tmp/common.json"}))
	require.ElementsMatch(t, []string{validURI}, application.Subscriptions.URIsForSession(clientSession.ID()))

	require.NoError(t, clientSession.Unsubscribe(ctx, &mcp.UnsubscribeParams{URI: validURI}))
	require.Empty(t, application.Subscriptions.URIsForSession(clientSession.ID()))
}

func TestHealthTool(t *testing.T) {
	ctx := t.Context()
	projectRoot := t.TempDir()

	application, err := app.New(ctx, app.Options{ProjectRoot: projectRoot, LogLevel: "error"})
	require.NoError(t, err)

	server := mcpserver.New(application)
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.0"}, nil)
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	serverSession, err := server.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, serverSession.Close())
	})

	clientSession, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, clientSession.Close())
	})

	res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.health"})
	require.NoError(t, err)
	require.False(t, res.IsError)

	data, err := json.Marshal(res.StructuredContent)
	require.NoError(t, err)

	var out struct {
		Name        string `json:"name"`
		ProjectRoot string `json:"projectRoot"`
		Status      string `json:"status"`
	}
	require.NoError(t, json.Unmarshal(data, &out))
	require.Equal(t, "i18n-mcp", out.Name)
	require.Equal(t, projectRoot, out.ProjectRoot)
	require.Equal(t, "ok", out.Status)
}
