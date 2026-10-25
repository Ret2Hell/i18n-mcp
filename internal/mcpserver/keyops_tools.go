package mcpserver

import (
	"context"
	"fmt"
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
		token := ""
		if !in.DryRunValue() {
			token = in.PreviewToken
		}
		view, err := a.Previews.Prepare(ctx, "i18n.keys.rename", canonical, out.ChangedFiles, out.StateUpdates, token)
		if err != nil {
			return nil, keyops.RenameOutput{}, err
		}
		out.ChangedFiles = view.Files
		out.Preview = view.Metadata
		if err := checkWritePreview(out); err != nil {
			return nil, keyops.RenameOutput{}, err
		}
		if len(out.Conflicts) > 0 {
			return &mcp.CallToolResult{IsError: true}, out, nil
		}
		if in.DryRunValue() {
			return nil, out, nil
		}
		if err := requireReview(view, in.PreviewToken); err != nil {
			return writePreviewError(err), out, nil
		}
		in.BeforeWrite = func(ctx context.Context, actual keyops.RenameOutput) error {
			return a.Previews.Verify(ctx, "i18n.keys.rename", canonical, actual.ChangedFiles, actual.StateUpdates, view.Metadata.Token)
		}
		previous := out
		out, err = a.KeyOps.Rename(ctx, in)
		out.ChangedFiles = reviewedFiles(out.ChangedFiles, view)
		out.Preview = view.Metadata
		out.Warnings = boundWriteWarnings(out.Warnings, view)
		if len(out.Conflicts) > 0 && err == nil {
			err = fmt.Errorf("rename now conflicts; request a new preview")
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
