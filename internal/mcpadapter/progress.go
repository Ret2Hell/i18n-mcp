package mcpadapter

import (
	"context"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ProgressReporter reports coarse-grained progress for long-running operations.
type ProgressReporter interface {
	Step(ctx context.Context, message string, current int, total int)
}

// MCPProgressReporter sends MCP progress notifications when the request carries a progress token.
type MCPProgressReporter struct {
	req    *mcp.CallToolRequest
	logger *slog.Logger
}

// NewMCPProgressReporter returns a reporter bound to an MCP tool request.
func NewMCPProgressReporter(req *mcp.CallToolRequest, logger *slog.Logger) *MCPProgressReporter {
	return &MCPProgressReporter{req: req, logger: logger}
}

// Step emits an MCP progress notification if the client supplied a progress token.
func (r *MCPProgressReporter) Step(ctx context.Context, message string, current int, total int) {
	if r == nil || r.req == nil || r.req.Params == nil || r.req.Session == nil {
		return
	}
	token := r.req.Params.GetProgressToken()
	if token == nil {
		return
	}
	current = max(current, 0)
	total = max(total, 0)
	err := r.req.Session.NotifyProgress(ctx, &mcp.ProgressNotificationParams{
		ProgressToken: token,
		Message:       message,
		Progress:      float64(current),
		Total:         float64(total),
	})
	if err != nil && r.logger != nil {
		r.logger.Debug("sending MCP progress", "error", err)
	}
}

// NoopProgressReporter discards progress updates.
type NoopProgressReporter struct{}

// Step discards a progress update.
func (NoopProgressReporter) Step(context.Context, string, int, int) {}
