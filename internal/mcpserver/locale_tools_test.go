package mcpserver_test

import (
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func TestLocalesListTool(t *testing.T) {
	ctx, clientSession := newTestClientSession(t, fixturePath(t, "next-intl-basic"))

	res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.locales.list"})
	require.NoError(t, err)
	require.False(t, res.IsError)

	data, err := json.Marshal(res.StructuredContent)
	require.NoError(t, err)

	var out struct {
		Inventory struct {
			Locales        []string       `json:"locales"`
			TargetLocales  []string       `json:"targetLocales"`
			CountsByLocale map[string]int `json:"countsByLocale"`
			Units          []any          `json:"units,omitzero"`
		} `json:"inventory"`
	}
	require.NoError(t, json.Unmarshal(data, &out))
	require.Equal(t, []string{"en", "fr"}, out.Inventory.Locales)
	require.Equal(t, []string{"fr"}, out.Inventory.TargetLocales)
	require.Equal(t, 2, out.Inventory.CountsByLocale["en"])
	require.Empty(t, out.Inventory.Units)
}

func TestLocalesListToolCanIncludeUnits(t *testing.T) {
	ctx, clientSession := newTestClientSession(t, fixturePath(t, "next-intl-basic"))

	res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "i18n.locales.list",
		Arguments: map[string]any{"includeUnits": true},
	})
	require.NoError(t, err)
	require.False(t, res.IsError)

	data, err := json.Marshal(res.StructuredContent)
	require.NoError(t, err)

	var out struct {
		Inventory struct {
			Units []struct {
				Locale string `json:"locale"`
				Key    string `json:"key"`
			} `json:"units"`
		} `json:"inventory"`
	}
	require.NoError(t, json.Unmarshal(data, &out))
	require.NotEmpty(t, out.Inventory.Units)
}
