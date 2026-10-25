package report_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type exportWriterFunc func([]byte) (int, error)

func (f exportWriterFunc) Write(p []byte) (int, error) { return f(p) }

func TestFullJSONExportPropagatesEveryWriteFailure(t *testing.T) {
	a := newReportFixtureApp(t)
	writes := 0
	_, err := a.Reports.WriteFullJSON(t.Context(), exportWriterFunc(func(p []byte) (int, error) { writes++; return len(p), nil }))
	require.NoError(t, err)
	want := errors.New("export storage failed")
	for failAt := 1; failAt <= writes; failAt++ {
		t.Run(fmt.Sprintf("write-%d", failAt), func(t *testing.T) {
			calls := 0
			var partial bytes.Buffer
			_, err := a.Reports.WriteFullJSON(t.Context(), exportWriterFunc(func(p []byte) (int, error) {
				calls++
				if calls == failAt {
					// A writer can make partial progress and still fail.
					n := len(p) / 2
					_, _ = partial.Write(p[:n])
					return n, want
				}
				return partial.Write(p)
			}))
			require.ErrorIs(t, err, want)
			require.Equal(t, failAt, calls, "must stop at the first failed write")
			_, ok, err := a.Reports.Latest(t.Context())
			require.NoError(t, err)
			require.False(t, ok, "failed exports must not populate the report cache")
		})
	}
}

func TestFullJSONExportCancelsDuringStreaming(t *testing.T) {
	a := newReportFixtureApp(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var partial bytes.Buffer
	rows := 0
	_, err := a.Reports.WriteFullJSON(ctx, exportWriterFunc(func(p []byte) (int, error) {
		if strings.Contains(string(p), `"status":`) {
			rows++
			cancel()
		}
		return partial.Write(p)
	}))
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 1, rows, "must stop before streaming the second row")
	require.NotEmpty(t, partial.Bytes())
	require.False(t, json.Valid(partial.Bytes()), "canceled output is incomplete")
	_, ok, err := a.Reports.Latest(t.Context())
	require.NoError(t, err)
	require.False(t, ok)
}

func TestFullJSONExportRejectsInvalidInputsBeforeWriting(t *testing.T) {
	for _, path := range []string{".i18n-mcp.json", "messages/en/common.json"} {
		t.Run(path, func(t *testing.T) {
			a := newReportFixtureApp(t)
			writeReportFile(t, a.Guard.Root(), path, `{`)
			var output bytes.Buffer
			_, err := a.Reports.WriteFullJSON(t.Context(), &output)
			require.Error(t, err)
			require.Empty(t, output.Bytes())
		})
	}
}
