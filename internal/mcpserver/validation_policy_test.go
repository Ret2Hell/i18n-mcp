package mcpserver_test

import (
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/diff"
	"github.com/Ret2Hell/i18n-mcp/internal/translate"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func TestDetectedICUPolicyFlowsThroughDiffPlanValidateAndApply(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "package.json", `{"dependencies":{"next-intl":"4"}}`)
	writeFile(t, root, "locales/en.json", `{"score":"{score}% score","notes":"{count, plural, =0 {Notes} one {# note} other {# notes}}"}`)
	writeFile(t, root, "locales/fr.json", `{"score":"Score : {score} %"}`)
	ctx, session := newTestClientSession(t, root)
	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.keys.diff"})
	require.NoError(t, err)
	require.False(t, res.IsError)
	var diffOut struct {
		Report diff.Report `json:"report"`
	}
	unmarshalStructuredContent(t, res.StructuredContent, &diffOut)
	require.Zero(t, diffOut.Report.Summary.Invalid)
	res, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.translation.plan"})
	require.NoError(t, err)
	require.False(t, res.IsError)
	var plan struct {
		Batch translate.Batch `json:"batch"`
	}
	unmarshalStructuredContent(t, res.StructuredContent, &plan)
	require.Len(t, plan.Batch.Items, 1)
	require.Equal(t, []string{"{count}"}, plan.Batch.Items[0].Placeholders)
	proposals := []map[string]any{{"locale": "fr", "namespace": "common", "key": "notes", "sourceValue": plan.Batch.Items[0].SourceValue, "value": "{count, plural, =0 {Notes françaises} one {# note française} other {# notes françaises}}"}}
	args := map[string]any{"batchId": plan.Batch.BatchID, "translations": proposals}
	for _, name := range []string{"i18n.translation.validate", "i18n.translation.apply"} {
		res, err = session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
		require.NoError(t, err)
		require.False(t, res.IsError)
	}
	args["apply"] = true
	res, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.translation.apply", Arguments: args})
	require.NoError(t, err)
	require.False(t, res.IsError)
	res, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.keys.diff"})
	require.NoError(t, err)
	unmarshalStructuredContent(t, res.StructuredContent, &diffOut)
	require.Zero(t, diffOut.Report.Summary.Invalid)
	require.Zero(t, diffOut.Report.Summary.Missing)
}
