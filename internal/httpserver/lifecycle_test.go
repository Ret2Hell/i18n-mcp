package httpserver

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func TestRunRejectsInvalidConfiguration(t *testing.T) {
	err := Run(t.Context(), Config{Addr: "0.0.0.0:0"}, staticServerProvider{}, nil)
	require.ErrorContains(t, err, "auth")
}

func TestRunReturnsListenFailure(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, listener.Close()) })
	provider := staticServerProvider{mcp.NewServer(&mcp.Implementation{Name: "test", Version: "test"}, nil)}
	err = Run(t.Context(), Config{Addr: listener.Addr().String()}, provider, nil)
	var opErr *net.OpError
	require.ErrorAs(t, err, &opErr)
	require.Equal(t, "listen", opErr.Op)
}

func TestRunShutsDownOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	provider := staticServerProvider{mcp.NewServer(&mcp.Implementation{Name: "test", Version: "test"}, nil)}
	done := make(chan error, 1)
	go func() { done <- Run(ctx, Config{Addr: "127.0.0.1:0"}, provider, nil) }()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("server did not shut down")
	}
}
