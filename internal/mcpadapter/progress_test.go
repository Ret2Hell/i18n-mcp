package mcpadapter_test

import (
	"context"
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/mcpadapter"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func TestMCPProgressReporterIgnoresNotificationError(t *testing.T) {
	ctx := t.Context()
	server := mcp.NewServer(&mcp.Implementation{Name: "test-server", Version: "v0.0.0"}, nil)
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.0"}, nil)
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	serverSession, err := server.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, serverSession.Close()) })

	clientSession, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, clientSession.Close()) })

	params := &mcp.CallToolParamsRaw{Name: "i18n.project.detect"}
	params.SetProgressToken("detect-progress")
	req := &mcp.CallToolRequest{Session: serverSession, Params: params}

	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()

	require.NotPanics(t, func() {
		mcpadapter.NewMCPProgressReporter(req, nil).Step(canceledCtx, "checking project files", 1, 4)
	})
}
