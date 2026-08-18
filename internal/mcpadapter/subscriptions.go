package mcpadapter

import (
	"context"
	"net/url"

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
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if u.Scheme != "i18n" {
		return mcp.ResourceNotFoundError(raw)
	}
	return nil
}
