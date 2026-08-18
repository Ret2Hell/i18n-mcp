package mcpadapter

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func TestSubscriptionValidatorsAcceptI18nURIs(t *testing.T) {
	require.NoError(t, ValidateSubscribe(t.Context(), &mcp.SubscribeRequest{Params: &mcp.SubscribeParams{URI: "i18n://analysis/diff"}}))
	require.NoError(t, ValidateUnsubscribe(t.Context(), &mcp.UnsubscribeRequest{Params: &mcp.UnsubscribeParams{URI: "i18n://analysis/diff"}}))
}

func TestSubscriptionValidatorsRejectOtherSchemes(t *testing.T) {
	require.Error(t, ValidateSubscribe(t.Context(), &mcp.SubscribeRequest{Params: &mcp.SubscribeParams{URI: "file:///tmp/messages.json"}}))
	require.Error(t, ValidateUnsubscribe(t.Context(), &mcp.UnsubscribeRequest{Params: &mcp.UnsubscribeParams{URI: "file:///tmp/messages.json"}}))
}
