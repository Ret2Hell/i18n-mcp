package mcpadapter

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func TestSubscriptionValidatorsAcceptRegisteredResources(t *testing.T) {
	for _, uri := range []string{
		"i18n://locales",
		"i18n://locales/fr/common",
		"i18n://analysis/diff",
		"i18n://translation/plan/latest",
		"i18n://reports/latest",
	} {
		t.Run(uri, func(t *testing.T) {
			require.NoError(t, ValidateSubscribe(t.Context(), &mcp.SubscribeRequest{Params: &mcp.SubscribeParams{URI: uri}}))
			require.NoError(t, ValidateUnsubscribe(t.Context(), &mcp.UnsubscribeRequest{Params: &mcp.UnsubscribeParams{URI: uri}}))
		})
	}
}

func TestSubscriptionValidatorsRejectUnknownOrMalformedResources(t *testing.T) {
	for _, uri := range []string{
		"file:///tmp/messages.json",
		"i18n://unknown",
		"i18n://analysis/unknown",
		"i18n://locales/fr",
		"i18n://locales/fr/common/extra",
		"i18n://locales/fr/common?private=true",
		"i18n://locales/fr%2Fca/common",
	} {
		t.Run(uri, func(t *testing.T) {
			require.Error(t, ValidateSubscribe(t.Context(), &mcp.SubscribeRequest{Params: &mcp.SubscribeParams{URI: uri}}))
			require.Error(t, ValidateUnsubscribe(t.Context(), &mcp.UnsubscribeRequest{Params: &mcp.UnsubscribeParams{URI: uri}}))
		})
	}
}
