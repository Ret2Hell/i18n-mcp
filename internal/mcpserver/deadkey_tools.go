package mcpserver

import (
	"context"
	"fmt"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/deadkey"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	pruneConfirmationRequestID    = "prune_confirmation"
	pruneConfirmationRequestState = "awaiting_prune_confirmation"
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

func keysPruneTool(a *app.App) func(context.Context, *mcp.CallToolRequest, deadkey.PruneInput) (*mcp.CallToolResult, *deadkey.PruneOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, in deadkey.PruneInput) (*mcp.CallToolResult, *deadkey.PruneOutput, error) {
		if in.Apply && in.ConfirmWithClient {
			edits, rejected, err := a.DeadKeys.BuildPruneEdits(ctx, in)
			if err != nil {
				return nil, nil, err
			}
			if len(rejected) > 0 {
				out := &deadkey.PruneOutput{DryRun: in.DryRunValue(), Rejected: rejected}
				return &mcp.CallToolResult{IsError: true}, out, nil
			}
			if !supportsFormElicitation(req) {
				return nil, nil, fmt.Errorf("client does not support form elicitation")
			}
			if len(req.Params.InputResponses) == 0 {
				return pruneConfirmationRequest(PrunePlan{KeyCount: len(edits)}), nil, nil
			}
			if err := validatePruneConfirmation(req); err != nil {
				return nil, nil, err
			}
		}

		out, err := a.DeadKeys.Prune(ctx, in)
		if err != nil {
			return nil, nil, err
		}
		if len(out.Rejected) > 0 {
			return &mcp.CallToolResult{IsError: true}, &out, nil
		}
		return nil, &out, nil
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

func pruneConfirmationRequest(planned PrunePlan) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		InputRequests: mcp.InputRequestMap{
			pruneConfirmationRequestID: &mcp.ElicitParams{
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
			},
		},
		RequestState: pruneConfirmationRequestState,
	}
}

func validatePruneConfirmation(req *mcp.CallToolRequest) error {
	if req.Params.RequestState != pruneConfirmationRequestState {
		return fmt.Errorf("prune confirmation request state is invalid")
	}
	response, ok := req.Params.InputResponses[pruneConfirmationRequestID]
	if !ok {
		return fmt.Errorf("prune confirmation response is missing")
	}
	result, ok := response.(*mcp.ElicitResult)
	if !ok {
		return fmt.Errorf("prune confirmation returned unexpected response type %T", response)
	}
	if result.Action != "accept" {
		return fmt.Errorf("prune confirmation %s", result.Action)
	}
	confirmed, ok := result.Content["confirm"].(bool)
	if !ok || !confirmed {
		return fmt.Errorf("prune confirmation declined")
	}
	return nil
}
