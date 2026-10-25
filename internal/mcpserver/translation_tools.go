package mcpserver

import (
	"context"
	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/translate"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func translationApplyTool(a *app.App) func(context.Context, *mcp.CallToolRequest, translate.ApplyInput) (*mcp.CallToolResult, translate.ApplyOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in translate.ApplyInput) (*mcp.CallToolResult, translate.ApplyOutput, error) {
		canonical := in
		canonical.Apply = false
		canonical.DryRun = nil
		canonical.PreviewToken = ""
		canonical.BeforeWrite = nil
		out, err := a.Translation.Apply(ctx, canonical)
		if err != nil {
			return nil, translate.ApplyOutput{}, err
		}
		return runReviewedWrite(ctx, a.Previews, reviewedWriteOperation[translate.ApplyOutput]{
			name:        "i18n.translation.apply",
			canonical:   canonical,
			dryRun:      in.DryRunValue(),
			token:       in.PreviewToken,
			invalidated: "translations no longer validate; request a new preview",
			outcome: func(out *translate.ApplyOutput) writeOutcome {
				return writeOutcome{files: &out.ChangedFiles, metadata: &out.Preview, warnings: &out.Warnings, state: out.PlannedStateUpdates, blocked: len(out.Rejected) > 0}
			},
			apply: func(ctx context.Context, verify func(context.Context, translate.ApplyOutput) error) (translate.ApplyOutput, error) {
				in.BeforeWrite = verify
				return a.Translation.Apply(ctx, in)
			},
		}, out)
	}
}
