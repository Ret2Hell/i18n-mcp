package mcpserver_test

import (
	"encoding/json"
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/locale"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func TestLocalesResource(t *testing.T) {
	ctx, clientSession := newTestClientSession(t, fixturePath(t, "next-intl-basic"))

	res, err := clientSession.ReadResource(ctx, &mcp.ReadResourceParams{URI: "i18n://locales"})
	require.NoError(t, err)
	require.Len(t, res.Contents, 1)
	require.Equal(t, "application/json", res.Contents[0].MIMEType)

	var inv locale.Inventory
	require.NoError(t, json.Unmarshal([]byte(res.Contents[0].Text), &inv))
	require.Equal(t, []string{"en", "fr"}, inv.Locales)
	require.Empty(t, inv.Units)
}

func TestLocaleNamespaceResourceTemplate(t *testing.T) {
	ctx, clientSession := newTestClientSession(t, fixturePath(t, "i18next-namespaces"))

	res, err := clientSession.ReadResource(ctx, &mcp.ReadResourceParams{URI: "i18n://locales/en/common"})
	require.NoError(t, err)
	require.Len(t, res.Contents, 1)
	require.Equal(t, "application/json", res.Contents[0].MIMEType)

	var content locale.NamespaceContent
	require.NoError(t, json.Unmarshal([]byte(res.Contents[0].Text), &content))
	require.Equal(t, "en", content.Locale)
	require.Equal(t, "common", content.Namespace)
	require.Len(t, content.Files, 1)
	require.Len(t, content.RawFiles, 1)
	require.Len(t, content.Units, 2)
}
