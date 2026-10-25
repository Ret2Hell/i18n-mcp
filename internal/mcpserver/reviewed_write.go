package mcpserver

import (
	"context"
	"errors"

	"github.com/Ret2Hell/i18n-mcp/internal/preview"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// writeOutcome adapts operation-specific outputs without moving their domain
// validation into the shared preview/review/write workflow. Pointers refer to
// fields of the output supplied to outcome; the adapter must not retain them.
type writeOutcome struct {
	files    *[]preview.File
	metadata **preview.Metadata
	warnings *[]string
	state    any
	blocked  bool
}

type reviewedWriteOperation[O any] struct {
	name        string
	canonical   any
	dryRun      bool
	token       string
	invalidated string
	outcome     func(*O) writeOutcome
	// apply must install verify as the service's immediately-before-write check.
	apply func(context.Context, func(context.Context, O) error) (O, error)
}

// runReviewedWrite receives an already-computed canonical dry-run result. Review
// tokens, response budgets and pre-write verification are identical for every
// operation; conflicts/rejections remain the responsibility of its adapter.
func runReviewedWrite[O any](ctx context.Context, previews *preview.Service, op reviewedWriteOperation[O], out O) (*mcp.CallToolResult, O, error) {
	var zero O
	result := op.outcome(&out)
	token := ""
	if !op.dryRun {
		token = op.token
	}
	view, err := previews.Prepare(ctx, op.name, op.canonical, *result.files, result.state, token)
	if err != nil {
		return nil, zero, err
	}
	*result.files = view.Files
	*result.metadata = view.Metadata
	if err := checkWritePreview(out); err != nil {
		return nil, zero, err
	}
	if result.blocked {
		return &mcp.CallToolResult{IsError: true}, out, nil
	}
	if op.dryRun {
		return nil, out, nil
	}
	if err := requireReview(view, op.token); err != nil {
		return writePreviewError(err), out, nil
	}
	verify := func(ctx context.Context, actual O) error {
		result := op.outcome(&actual)
		return previews.Verify(ctx, op.name, op.canonical, *result.files, result.state, view.Metadata.Token)
	}
	previous := out
	out, err = op.apply(ctx, verify)
	result = op.outcome(&out)
	*result.files = reviewedFiles(*result.files, view)
	*result.metadata = view.Metadata
	*result.warnings = boundWriteWarnings(*result.warnings, view)
	if result.blocked && err == nil {
		err = errors.New(op.invalidated)
	}
	if err != nil && !anyWritten(*result.files) && checkWritePreview(out) != nil {
		out = previous
		view.Metadata.DiagnosticsTruncated = true
	}
	if err != nil {
		return writePreviewError(err), out, nil
	}
	return nil, out, nil
}
