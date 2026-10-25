package mcpserver

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/pagination"
	"github.com/Ret2Hell/i18n-mcp/internal/preview"
	"github.com/Ret2Hell/i18n-mcp/internal/security"
	"github.com/stretchr/testify/require"
)

type reviewTestOutput struct {
	Files    []preview.File    `json:"files"`
	Preview  *preview.Metadata `json:"preview"`
	Warnings []string          `json:"warnings"`
	State    int               `json:"state"`
	Rejected []string          `json:"rejected"`
}

func reviewTestOperation() reviewedWriteOperation[reviewTestOutput] {
	return reviewedWriteOperation[reviewTestOutput]{
		name: "test.write", canonical: struct{ Key string }{"hello"}, invalidated: "plan no longer valid",
		outcome: func(out *reviewTestOutput) writeOutcome {
			return writeOutcome{files: &out.Files, metadata: &out.Preview, warnings: &out.Warnings, state: out.State, blocked: len(out.Rejected) > 0}
		},
	}
}

func reviewTestService(t *testing.T) *preview.Service {
	t.Helper()
	signer, err := security.NewRandomRequestStateSigner()
	require.NoError(t, err)
	return preview.NewService(t.TempDir(), signer)
}

func TestEmptyReviewedWriteSlicesRemainNil(t *testing.T) {
	for _, files := range [][]preview.File{nil, {}} {
		require.Nil(t, reviewedFiles(nil, preview.View{Files: files}))
	}
	for _, warnings := range [][]string{nil, {}} {
		view := preview.View{Metadata: &preview.Metadata{}}
		require.Nil(t, boundWriteWarnings(warnings, view))
		require.False(t, view.Metadata.DiagnosticsTruncated)
	}
}

func TestReviewedWriteRejectsBeforeApply(t *testing.T) {
	for _, tc := range []struct {
		name                       string
		dryRun, blocked, truncated bool
		wantError                  bool
	}{
		{name: "dry run", dryRun: true},
		{name: "domain rejection", blocked: true, wantError: true},
		{name: "truncated without approval", truncated: true, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			op := reviewTestOperation()
			op.dryRun = tc.dryRun
			if tc.dryRun {
				op.token = "ignored-on-dry-run"
			}
			op.apply = func(context.Context, func(context.Context, reviewTestOutput) error) (reviewTestOutput, error) {
				t.Fatal("unexpected apply")
				return reviewTestOutput{}, nil
			}
			out := reviewTestOutput{Files: []preview.File{{Path: "messages/fr.json", Diff: "patch", Changed: true}}, State: 1}
			if tc.blocked {
				out.Rejected = []string{"invalid"}
			}
			if tc.truncated {
				out.Files[0].Diff = strings.Repeat("x", preview.InlineBytes+1)
			}
			res, got, err := op.run(t.Context(), reviewTestService(t), out)
			require.NoError(t, err)
			require.NotNil(t, got.Preview)
			require.False(t, got.Files[0].Written)
			if tc.wantError {
				require.NotNil(t, res)
				require.True(t, res.IsError)
			} else {
				require.Nil(t, res)
			}
		})
	}
}

func TestReviewedWriteChecksActualPlanBeforeWrite(t *testing.T) {
	for _, drift := range []bool{false, true} {
		t.Run(map[bool]string{false: "approved", true: "drifted"}[drift], func(t *testing.T) {
			previews := reviewTestService(t)
			op := reviewTestOperation()
			out := reviewTestOutput{Files: []preview.File{{Path: "messages/fr.json", Diff: strings.Repeat("x", preview.InlineBytes+1), Changed: true}}, State: 1}
			view, err := previews.Prepare(t.Context(), op.name, op.canonical, out.Files, out.State, "")
			require.NoError(t, err)
			op.token = view.Metadata.Token
			wrote := false
			op.apply = func(ctx context.Context, verify func(context.Context, reviewTestOutput) error) (reviewTestOutput, error) {
				actual := out
				if drift {
					actual.State++
				}
				if err := verify(ctx, actual); err != nil {
					return actual, err
				}
				wrote = true
				actual.Files[0].Written = true
				return actual, nil
			}
			res, got, err := op.run(t.Context(), previews, out)
			require.NoError(t, err)
			require.Equal(t, !drift, wrote)
			require.Equal(t, !drift, got.Files[0].Written)
			require.Equal(t, op.token, got.Preview.Token)
			require.LessOrEqual(t, len(got.Files[0].Diff), preview.InlineBytes)
			if drift {
				require.True(t, res.IsError)
			} else {
				require.Nil(t, res)
			}
		})
	}
}

func TestReviewedWritePreservesFailureAndBoundsDiagnostics(t *testing.T) {
	for _, tc := range []struct {
		name                        string
		blocked, oversized, written bool
	}{
		{name: "new rejection", blocked: true},
		{name: "oversized failure", oversized: true},
		{name: "partial write", written: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			op := reviewTestOperation()
			out := reviewTestOutput{Files: []preview.File{{Path: "messages/fr.json", Diff: "patch", Changed: true}}}
			op.apply = func(ctx context.Context, verify func(context.Context, reviewTestOutput) error) (reviewTestOutput, error) {
				require.NoError(t, verify(ctx, out))
				actual := out
				actual.Files = slices.Clone(out.Files)
				actual.Files[0].Written = tc.written
				actual.Warnings = []string{strings.Repeat("w", 4096)}
				if tc.blocked {
					actual.Rejected = []string{"changed"}
					return actual, nil
				}
				if tc.oversized {
					actual.Rejected = []string{strings.Repeat("x", pagination.ResponseBudget)}
				}
				return actual, errors.New("write failed")
			}
			res, got, err := op.run(t.Context(), reviewTestService(t), out)
			require.NoError(t, err)
			require.True(t, res.IsError)
			require.True(t, got.Preview.DiagnosticsTruncated)
			require.Equal(t, tc.written, got.Files[0].Written)
			require.NoError(t, checkWritePreview(got))
			if tc.oversized {
				require.Empty(t, got.Rejected)
			}
		})
	}
}
