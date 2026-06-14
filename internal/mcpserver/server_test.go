package mcpserver_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/mcpserver"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func TestHealthTool(t *testing.T) {
	ctx := context.Background()
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
