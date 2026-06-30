package mcpserver

import (
	"context"
	"slices"
	"strings"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const completionLimit = 100

func complete(a *app.App) func(context.Context, *mcp.CompleteRequest) (*mcp.CompleteResult, error) {
	return func(ctx context.Context, req *mcp.CompleteRequest) (*mcp.CompleteResult, error) {
		if req == nil || req.Params == nil {
			return completionResult(nil), nil
		}
		arg := req.Params.Argument.Name
		prefix := req.Params.Argument.Value
		var contextArgs map[string]string
		if req.Params.Context != nil && req.Params.Context.Arguments != nil {
			contextArgs = req.Params.Context.Arguments
		}

		var values []string
		var err error
		switch arg {
		case "locale", "locales":
			values, err = completeLocales(ctx, a, prefix)
		case "namespace", "namespaces":
			values, err = completeNamespaces(ctx, a, prefix)
		case "key", "keys", "fromKey", "toKey":
			values, err = completeKeys(ctx, a, prefix, contextArgs)
		case "batchId":
			values = a.Translation.CompleteBatchIDs(prefix)
		default:
			values = nil
		}
		if err != nil {
			return nil, err
		}
		return completionResult(values), nil
	}
}

func completeLocales(ctx context.Context, a *app.App, prefix string) ([]string, error) {
	inv, err := a.Locales.Inventory(ctx)
	if err != nil {
		return nil, err
	}
	return filterPrefix(inv.Locales, prefix), nil
}

func completeNamespaces(ctx context.Context, a *app.App, prefix string) ([]string, error) {
	inv, err := a.Locales.Inventory(ctx)
	if err != nil {
		return nil, err
	}
	return filterPrefix(inv.Namespaces, prefix), nil
}

func completeKeys(ctx context.Context, a *app.App, prefix string, contextArgs map[string]string) ([]string, error) {
	inv, err := a.Locales.Inventory(ctx)
	if err != nil {
		return nil, err
	}

	namespace := strings.TrimSpace(contextArgs["namespace"])
	values := make([]string, 0, len(inv.Units))
	for _, unit := range inv.Units {
		if unit.Locale != inv.SourceLocale {
			continue
		}
		if namespace != "" && unit.Namespace != namespace {
			continue
		}
		values = append(values, unit.Key)
	}
	return filterPrefix(values, prefix), nil
}

func completionResult(values []string) *mcp.CompleteResult {
	values = uniqueSorted(values)
	total := len(values)
	hasMore := len(values) > completionLimit
	if hasMore {
		values = values[:completionLimit]
	}
	return &mcp.CompleteResult{
		Completion: mcp.CompletionResultDetails{
			Values:  values,
			Total:   total,
			HasMore: hasMore,
		},
	}
}

func filterPrefix(values []string, prefix string) []string {
	prefix = strings.TrimSpace(prefix)
	var out []string
	for _, value := range values {
		if prefix == "" || strings.HasPrefix(value, prefix) {
			out = append(out, value)
		}
	}
	return out
}

func uniqueSorted(values []string) []string {
	slices.Sort(values)
	values = slices.DeleteFunc(values, func(value string) bool {
		return value == ""
	})
	return slices.Compact(values)
}
