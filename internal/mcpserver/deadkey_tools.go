package mcpserver

import (
	"context"
	"fmt"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/deadkey"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DeadReportOutput is the output for the dead-key report tool.
type DeadReportOutput struct {
	Report deadkey.Report `json:"report" jsonschema:"dead-key classification report with evidence and confidence"`
}

func deadReportTool(a *app.App) func(context.Context, *mcp.CallToolRequest, deadkey.ReportInput) (*mcp.CallToolResult, DeadReportOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, in deadkey.ReportInput) (*mcp.CallToolResult, DeadReportOutput, error) {
		_ = req
		report, err := a.DeadKeys.Report(ctx, in)
		if err != nil {
			return nil, DeadReportOutput{}, err
		}
		return nil, DeadReportOutput{Report: report}, nil
	}
}

func keysPruneTool(a *app.App) func(context.Context, *mcp.CallToolRequest, deadkey.PruneInput) (*mcp.CallToolResult, deadkey.PruneOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, in deadkey.PruneInput) (*mcp.CallToolResult, deadkey.PruneOutput, error) {
		if in.Apply && in.ConfirmWithClient {
			edits, rejected, err := a.DeadKeys.BuildPruneEdits(ctx, in)
			if err != nil {
				return nil, deadkey.PruneOutput{}, err
			}
			if len(rejected) > 0 {
				out := deadkey.PruneOutput{DryRun: in.DryRunValue(), Rejected: rejected}
				return &mcp.CallToolResult{IsError: true}, out, nil
			}
			if err := confirmPrune(ctx, req, PrunePlan{KeyCount: len(edits)}); err != nil {
				return nil, deadkey.PruneOutput{}, err
			}
		}

		out, err := a.DeadKeys.Prune(ctx, in)
		if err != nil {
			return nil, deadkey.PruneOutput{}, err
		}
		if len(out.Rejected) > 0 {
			return &mcp.CallToolResult{IsError: true}, out, nil
		}
		return nil, out, nil
	}
}

// PrunePlan summarizes a pending prune confirmation.
type PrunePlan struct {
	KeyCount int
}

func supportsFormElicitation(req *mcp.CallToolRequest) bool {
	if req == nil || req.Session == nil || req.Session.InitializeParams() == nil {
		return false
	}
	caps := req.Session.InitializeParams().Capabilities
	return caps != nil && caps.Elicitation != nil && caps.Elicitation.Form != nil
}

func confirmPrune(ctx context.Context, req *mcp.CallToolRequest, planned PrunePlan) error {
	if !supportsFormElicitation(req) {
		return fmt.Errorf("client does not support form elicitation")
	}
	res, err := req.Session.Elicit(ctx, &mcp.ElicitParams{
		Mode:    "form",
		Message: fmt.Sprintf("Confirm pruning %d i18n keys from locale files.", planned.KeyCount),
		RequestedSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"confirm": map[string]any{"type": "boolean", "title": "Apply prune"},
				"note":    map[string]any{"type": "string", "title": "Review note"},
			},
			"required": []string{"confirm"},
		},
	})
	if err != nil {
		return err
	}
	if res.Action != "accept" {
		return fmt.Errorf("prune confirmation %s", res.Action)
	}
	confirmed, ok := res.Content["confirm"].(bool)
	if !ok || !confirmed {
		return fmt.Errorf("prune confirmation declined")
	}
	return nil
}
