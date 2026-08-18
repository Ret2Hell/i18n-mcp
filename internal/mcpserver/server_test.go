package mcpserver_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/mcpserver"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func TestServerCapabilities(t *testing.T) {
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
	require.Nil(t, capabilities.Logging)
	require.NotNil(t, capabilities.Completions)
	require.NotNil(t, capabilities.Tools)
	require.NotNil(t, capabilities.Prompts)
	require.NotNil(t, capabilities.Resources)
}

func TestResourceSubscriptions(t *testing.T) {
	ctx := t.Context()
	application, err := app.New(ctx, app.Options{ProjectRoot: t.TempDir(), LogLevel: "error"})
	require.NoError(t, err)

	server := mcpserver.New(application)
	updates := make(chan string, 1)
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.0"}, &mcp.ClientOptions{
		ResourceUpdatedHandler: func(_ context.Context, req *mcp.ResourceUpdatedNotificationRequest) {
			updates <- req.Params.URI
		},
	})
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	serverSession, err := server.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, serverSession.Close()) })

	clientSession, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, clientSession.Close()) })

	capabilities := clientSession.InitializeResult().Capabilities
	require.NotNil(t, capabilities.Resources)

	uri := "i18n://analysis/diff"
	require.NoError(t, clientSession.Subscribe(ctx, &mcp.SubscribeParams{URI: uri}))
	deadline := time.NewTimer(time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer deadline.Stop()
	defer ticker.Stop()
	for {
		require.NoError(t, server.ResourceUpdated(ctx, &mcp.ResourceUpdatedNotificationParams{URI: uri}))
		select {
		case got := <-updates:
			require.Equal(t, uri, got)
			goto subscribed
		case <-ticker.C:
		case <-deadline.C:
			t.Fatal("timed out waiting for subscribed resource update")
		}
	}

subscribed:

	require.NoError(t, clientSession.Unsubscribe(ctx, &mcp.UnsubscribeParams{URI: uri}))
	time.Sleep(50 * time.Millisecond)
	for len(updates) > 0 {
		<-updates
	}
	require.NoError(t, server.ResourceUpdated(ctx, &mcp.ResourceUpdatedNotificationParams{URI: uri}))
	select {
	case got := <-updates:
		t.Fatalf("received update %q after unsubscribe", got)
	case <-time.After(50 * time.Millisecond):
	}
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
