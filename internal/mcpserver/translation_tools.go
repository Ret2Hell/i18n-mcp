package mcpserver

import (
	"context"
	"fmt"
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
		token := ""
		if !in.DryRunValue() {
			token = in.PreviewToken
		}
		view, err := a.Previews.Prepare(ctx, "i18n.translation.apply", canonical, out.ChangedFiles, out.PlannedStateUpdates, token)
		if err != nil {
			return nil, translate.ApplyOutput{}, err
		}
		out.ChangedFiles = view.Files
		out.Preview = view.Metadata
		if err := checkWritePreview(out); err != nil {
			return nil, translate.ApplyOutput{}, err
		}
		if len(out.Rejected) > 0 {
			return &mcp.CallToolResult{IsError: true}, out, nil
		}
		if in.DryRunValue() {
			return nil, out, nil
		}
		if err := requireReview(view, in.PreviewToken); err != nil {
			return writePreviewError(err), out, nil
		}
		in.BeforeWrite = func(ctx context.Context, actual translate.ApplyOutput) error {
			return a.Previews.Verify(ctx, "i18n.translation.apply", canonical, actual.ChangedFiles, actual.PlannedStateUpdates, view.Metadata.Token)
		}
		previous := out
		out, err = a.Translation.Apply(ctx, in)
		out.ChangedFiles = reviewedFiles(out.ChangedFiles, view)
		out.Preview = view.Metadata
		out.Warnings = boundWriteWarnings(out.Warnings, view)
		if len(out.Rejected) > 0 && err == nil {
			err = fmt.Errorf("translations no longer validate; request a new preview")
		}
		if err != nil && !anyWritten(out.ChangedFiles) && checkWritePreview(out) != nil {
			out = previous
			view.Metadata.DiagnosticsTruncated = true
		}
		if err != nil {
			return writePreviewError(err), out, nil
		}
		return nil, out, nil
	}
}
