package mcpserver

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func TestValidatePruneConfirmation(t *testing.T) {
	tests := []struct {
		name        string
		state       string
		responses   mcp.InputResponseMap
		errorString string
	}{
		{
			name:  "accepted",
			state: pruneConfirmationRequestState,
			responses: mcp.InputResponseMap{
				pruneConfirmationRequestID: &mcp.ElicitResult{Action: "accept", Content: map[string]any{"confirm": true}},
			},
		},
		{
			name:        "invalid request state",
			responses:   mcp.InputResponseMap{pruneConfirmationRequestID: &mcp.ElicitResult{Action: "accept", Content: map[string]any{"confirm": true}}},
			errorString: "request state is invalid",
		},
		{
			name:        "missing response",
			state:       pruneConfirmationRequestState,
			errorString: "response is missing",
		},
		{
			name:        "declined",
			state:       pruneConfirmationRequestState,
			responses:   mcp.InputResponseMap{pruneConfirmationRequestID: &mcp.ElicitResult{Action: "decline"}},
			errorString: "confirmation decline",
		},
		{
			name:        "missing confirmation value",
			state:       pruneConfirmationRequestState,
			responses:   mcp.InputResponseMap{pruneConfirmationRequestID: &mcp.ElicitResult{Action: "accept"}},
			errorString: "confirmation declined",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{RequestState: tt.state, InputResponses: tt.responses}}

			err := validatePruneConfirmation(req)

			if tt.errorString == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tt.errorString)
			}
		})
	}
}
