package mcpserver

import (
	"context"
	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/keyops"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func keysRenameTool(a *app.App) func(context.Context, *mcp.CallToolRequest, keyops.RenameInput) (*mcp.CallToolResult, keyops.RenameOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in keyops.RenameInput) (*mcp.CallToolResult, keyops.RenameOutput, error) {
		canonical := in
		canonical.Apply = false
		canonical.DryRun = nil
		canonical.PreviewToken = ""
		canonical.BeforeWrite = nil
		out, err := a.KeyOps.Rename(ctx, canonical)
		if err != nil {
			return nil, keyops.RenameOutput{}, err
		}
		return runReviewedWrite(ctx, a.Previews, reviewedWriteOperation[keyops.RenameOutput]{
			name:        "i18n.keys.rename",
			canonical:   canonical,
			dryRun:      in.DryRunValue(),
			token:       in.PreviewToken,
			invalidated: "rename now conflicts; request a new preview",
			outcome: func(out *keyops.RenameOutput) writeOutcome {
				return writeOutcome{files: &out.ChangedFiles, metadata: &out.Preview, warnings: &out.Warnings, state: out.StateUpdates, blocked: len(out.Conflicts) > 0}
			},
			apply: func(ctx context.Context, verify func(context.Context, keyops.RenameOutput) error) (keyops.RenameOutput, error) {
				in.BeforeWrite = verify
				return a.KeyOps.Rename(ctx, in)
			},
		}, out)
	}
}
