package mcpserver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/deadkey"
	"github.com/Ret2Hell/i18n-mcp/internal/security"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	pruneConfirmationRequestID = "prune_confirmation"
	pruneOperation             = "i18n.keys.prune"
	pruneConfirmationTTL       = 5 * time.Minute
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
			plan, rejected, err := a.DeadKeys.PlanPrune(ctx, in)
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
			if !hasPruneConfirmationResponse(req) {
				result, err := pruneConfirmationRequest(ctx, a, in, plan)
				return result, nil, err
			}
			confirmed, err := validatePruneConfirmation(req)
			if err != nil {
				return nil, nil, err
			}
			if !confirmed {
				out := &deadkey.PruneOutput{DryRun: false, Warnings: []string{"prune confirmation declined; no files were changed"}}
				return nil, out, nil
			}
			claims, err := validatePruneRequestState(ctx, a, req, in, plan)
			if err != nil {
				return nil, nil, err
			}
			out, err := a.DeadKeys.PruneConfirmed(ctx, in, claims.PlanDigest)
			if err != nil {
				return nil, nil, err
			}
			return pruneToolOutput(out)
		}

		out, err := a.DeadKeys.Prune(ctx, in)
		if err != nil {
			return nil, nil, err
		}
		return pruneToolOutput(out)
	}
}

func pruneToolOutput(out deadkey.PruneOutput) (*mcp.CallToolResult, *deadkey.PruneOutput, error) {
	if len(out.Rejected) > 0 {
		return &mcp.CallToolResult{IsError: true}, &out, nil
	}
	return nil, &out, nil
}

func supportsFormElicitation(req *mcp.CallToolRequest) bool {
	if req == nil {
		return false
	}
	caps := req.ClientCapabilities()
	return caps != nil && caps.Elicitation != nil && caps.Elicitation.Form != nil
}

func hasPruneConfirmationResponse(req *mcp.CallToolRequest) bool {
	if req == nil || req.Params == nil {
		return false
	}
	_, ok := req.Params.InputResponses[pruneConfirmationRequestID]
	return ok
}

func pruneConfirmationRequest(ctx context.Context, a *app.App, in deadkey.PruneInput, plan deadkey.PrunePlan) (*mcp.CallToolResult, error) {
	inputDigest, err := pruneInputDigest(in)
	if err != nil {
		return nil, err
	}
	requestState, err := a.RequestStateSigner.Sign(security.RequestStateClaims{
		Subject:     security.SubjectFromContext(ctx),
		Operation:   pruneOperation,
		InputDigest: inputDigest,
		PlanDigest:  plan.Digest,
		ExpiresAt:   time.Now().Add(pruneConfirmationTTL).Unix(),
	})
	if err != nil {
		return nil, err
	}
	return &mcp.CallToolResult{
		InputRequests: mcp.InputRequestMap{
			pruneConfirmationRequestID: &mcp.ElicitParams{
				Mode:    "form",
				Message: fmt.Sprintf("Confirm pruning %d i18n keys from locale files.", plan.KeyCount),
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
		RequestState: requestState,
	}, nil
}

func validatePruneConfirmation(req *mcp.CallToolRequest) (bool, error) {
	response, ok := req.Params.InputResponses[pruneConfirmationRequestID]
	if !ok {
		return false, fmt.Errorf("prune confirmation response is missing")
	}
	result, ok := response.(*mcp.ElicitResult)
	if !ok {
		return false, fmt.Errorf("prune confirmation returned unexpected response type %T", response)
	}
	if result.Action != "accept" {
		return false, nil
	}
	confirmed, ok := result.Content["confirm"].(bool)
	if !ok {
		return false, fmt.Errorf("prune confirmation response is missing confirm")
	}
	return confirmed, nil
}

func validatePruneRequestState(ctx context.Context, a *app.App, req *mcp.CallToolRequest, in deadkey.PruneInput, plan deadkey.PrunePlan) (security.RequestStateClaims, error) {
	claims, err := a.RequestStateSigner.Verify(req.Params.RequestState)
	if err != nil {
		return security.RequestStateClaims{}, err
	}
	inputDigest, err := pruneInputDigest(in)
	if err != nil {
		return security.RequestStateClaims{}, err
	}
	if claims.Subject != security.SubjectFromContext(ctx) || claims.Operation != pruneOperation || claims.InputDigest != inputDigest || claims.PlanDigest != plan.Digest {
		return security.RequestStateClaims{}, fmt.Errorf("prune confirmation request state does not match this request")
	}
	return claims, nil
}

func pruneInputDigest(in deadkey.PruneInput) (string, error) {
	payload, err := json.Marshal(in)
	if err != nil {
		return "", fmt.Errorf("marshal prune confirmation input: %w", err)
	}
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:]), nil
}
