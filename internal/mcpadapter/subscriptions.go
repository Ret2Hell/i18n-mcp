package mcpadapter

import (
	"context"

	"github.com/Ret2Hell/i18n-mcp/internal/resources"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ValidateSubscribe accepts subscriptions only for i18n resources.
func ValidateSubscribe(_ context.Context, req *mcp.SubscribeRequest) error {
	return validateI18nURI(req.Params.URI)
}

// ValidateUnsubscribe accepts unsubscriptions only for i18n resources.
func ValidateUnsubscribe(_ context.Context, req *mcp.UnsubscribeRequest) error {
	return validateI18nURI(req.Params.URI)
}

func validateI18nURI(raw string) error {
	if !resources.IsSubscribableURI(raw) {
		return mcp.ResourceNotFoundError(raw)
	}
	return nil
}
