package mcpserver

import (
	"context"
	"sync"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func TestWorkLimitRejectsExcessAndReleasesCapacity(t *testing.T) {
	var once sync.Once
	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan struct{})
	next := func(_ context.Context, _ string, req mcp.Request) (mcp.Result, error) {
		if req.GetParams().(*mcp.CallToolParamsRaw).Name == "i18n.health" {
			return &mcp.CallToolResult{}, nil
		}
		once.Do(func() { close(started) })
		<-release
		return &mcp.CallToolResult{}, nil
	}
	handler := workLimit()(next)
	request := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Name: "i18n.keys.diff"}}
	go func() { defer close(done); _, _ = handler(t.Context(), "tools/call", request) }()
	<-started
	result, err := handler(t.Context(), "tools/call", request)
	require.NoError(t, err)
	require.True(t, result.(*mcp.CallToolResult).IsError)
	health := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Name: "i18n.health"}}
	result, err = handler(t.Context(), "tools/call", health)
	require.NoError(t, err)
	require.False(t, result.(*mcp.CallToolResult).IsError)
	close(release)
	<-done
	result, err = handler(t.Context(), "tools/call", request)
	require.NoError(t, err)
	require.False(t, result.(*mcp.CallToolResult).IsError)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = handler(ctx, "tools/call", request)
	require.ErrorIs(t, err, context.Canceled)
}
