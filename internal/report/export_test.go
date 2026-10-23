package report_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/report"
	"github.com/stretchr/testify/require"
)

type failedExportWriter struct{}

func (failedExportWriter) Write([]byte) (int, error) { return 0, errors.New("export disk full") }

func TestFullJSONExportStreamsAllRowsAndDoesNotCache(t *testing.T) {
	a := newReportFixtureApp(t)
	var buf bytes.Buffer
	metadata, err := a.Reports.WriteFullJSON(t.Context(), &buf)
	require.NoError(t, err)
	var exported report.Report
	require.NoError(t, json.Unmarshal(buf.Bytes(), &exported))
	require.Len(t, exported.Diff.Items, 2)
	require.Equal(t, metadata.Summary, exported.Summary)
	require.Empty(t, metadata.Diff.Items)
	_, ok, err := a.Reports.Latest(t.Context())
	require.NoError(t, err)
	require.False(t, ok)
	_, err = a.Reports.WriteFullJSON(t.Context(), failedExportWriter{})
	require.ErrorContains(t, err, "disk full")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	var canceled bytes.Buffer
	_, err = a.Reports.WriteFullJSON(ctx, &canceled)
	require.ErrorIs(t, err, context.Canceled)
	require.Empty(t, canceled.String())
}
