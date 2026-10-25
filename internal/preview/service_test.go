package preview

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Ret2Hell/i18n-mcp/internal/security"
	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/stretchr/testify/require"
)

func subject(t *testing.T, name string) context.Context {
	t.Helper()
	var ctx context.Context
	handler := auth.RequireBearerToken(func(context.Context, string, *http.Request) (*auth.TokenInfo, error) {
		return &auth.TokenInfo{UserID: name, Expiration: time.Now().Add(time.Hour)}, nil
	}, nil)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { ctx = r.Context() }))
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer test")
	handler.ServeHTTP(httptest.NewRecorder(), req)
	require.NotNil(t, ctx)
	return ctx
}

func TestPatchChunksAreExactImmutableOwnedAndBounded(t *testing.T) {
	signer, err := security.NewRandomRequestStateSigner()
	require.NoError(t, err)
	service := NewService("project", signer)
	ctx := subject(t, "a")
	original := strings.Repeat("é\n", 20000)
	files := []File{{Path: "locales/fr.json", Diff: original, Changed: true}}
	view, err := service.Prepare(ctx, "op", map[string]any{"key": "value"}, files, nil, "")
	require.NoError(t, err)
	require.True(t, view.Metadata.Truncated)
	require.LessOrEqual(t, len(view.Files[0].Diff), InlineBytes)
	require.True(t, view.Files[0].DiffTruncated)
	id := strings.Split(view.Metadata.ManifestURI, "/")[3]
	_, err = service.Page(subject(t, "b"), id, "file", 0, 0)
	require.Error(t, err)
	var recovered strings.Builder
	offset := 0
	for {
		page, err := service.Page(ctx, id, "file", 0, offset)
		require.NoError(t, err)
		raw, err := json.Marshal(page)
		require.NoError(t, err)
		require.Less(t, len(raw), 32<<10)
		var chunk struct {
			Data         string
			TotalBytes   int
			NextResource string
			SHA256       string
		}
		require.NoError(t, json.Unmarshal(raw, &chunk))
		bytes, err := base64.StdEncoding.DecodeString(chunk.Data)
		require.NoError(t, err)
		recovered.Write(bytes)
		offset += len(bytes)
		require.Equal(t, sum(original), chunk.SHA256)
		if chunk.NextResource == "" {
			break
		}
	}
	require.Equal(t, original, recovered.String())
	require.NoError(t, service.Verify(ctx, "op", map[string]any{"key": "value"}, files, nil, view.Metadata.Token))
	files[0].Diff = "changed"
	require.Error(t, service.Verify(ctx, "op", map[string]any{"key": "value"}, files, nil, view.Metadata.Token))
	// Mutating the caller's file slice cannot alter an already-published snapshot.
	page, err := service.Page(ctx, id, "file", 0, 0)
	require.NoError(t, err)
	require.NotNil(t, page)
	require.Error(t, NewService("other", signer).Verify(ctx, "op", nil, files, nil, view.Metadata.Token))
	service.now = func() time.Time { return time.Now().Add(ttl + time.Second) }
	_, err = service.Page(ctx, id, "file", 0, 0)
	require.Error(t, err)
}

func TestPreviewRetentionAndManifestLimits(t *testing.T) {
	signer, err := security.NewRandomRequestStateSigner()
	require.NoError(t, err)
	service := NewService("project", signer)
	files := make([]File, MaxInlineFiles+5)
	for i := range files {
		files[i] = File{Path: "file", Diff: "diff", Changed: true}
	}
	first, err := service.Prepare(t.Context(), "op", nil, files, nil, "")
	require.NoError(t, err)
	require.Len(t, first.Files, MaxInlineFiles)
	require.True(t, first.Metadata.Truncated)
	for range maxSnapshots {
		_, err = service.Prepare(t.Context(), "op", nil, nil, nil, "")
		require.NoError(t, err)
	}
	require.LessOrEqual(t, len(service.records), maxSnapshots)
	require.Error(t, service.Verify(t.Context(), "op", nil, files, nil, first.Metadata.Token))
	_, err = service.Prepare(t.Context(), "op", nil, []File{{Diff: strings.Repeat("x", MaxBytes+1)}}, nil, "")
	require.Error(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = service.Prepare(ctx, "op", nil, nil, nil, "")
	require.ErrorIs(t, err, context.Canceled)
}
