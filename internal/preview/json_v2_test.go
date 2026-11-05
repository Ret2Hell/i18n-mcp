package preview

import (
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/security"
	"github.com/stretchr/testify/require"
)

func TestPreviewDigestsAreDeterministic(t *testing.T) {
	first := map[string]any{"z": map[string]string{"b": "last", "a": "first"}, "a": "<safe> & text"}
	second := map[string]any{"a": "<safe> & text", "z": map[string]string{"a": "first", "b": "last"}}
	want, err := digestValue(first)
	require.NoError(t, err)
	for range 20 {
		got, err := digestValue(second)
		require.NoError(t, err)
		require.Equal(t, want, got)
	}
	signer, err := security.NewRandomRequestStateSigner()
	require.NoError(t, err)
	service := NewService("project", signer)
	view, err := service.Prepare(t.Context(), "op", first, nil, first, "")
	require.NoError(t, err)
	require.NoError(t, service.Verify(t.Context(), "op", second, nil, second, view.Metadata.Token))
}

func TestPreviewDoesNotPublishPartialDigests(t *testing.T) {
	signer, err := security.NewRandomRequestStateSigner()
	require.NoError(t, err)
	service := NewService("project", signer)
	for name, input := range map[string]any{
		"invalid_utf8": string([]byte{0xff}),
		"unsupported":  make(chan int),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := service.Prepare(t.Context(), "op", input, nil, nil, "")
			require.Error(t, err)
			_, err = service.Prepare(t.Context(), "op", nil, nil, input, "")
			require.Error(t, err)
			require.Empty(t, service.records)
		})
	}
	_, err = service.Prepare(t.Context(), "op", nil, []File{{Path: string([]byte{0xff})}}, nil, "")
	require.Error(t, err)
	require.Empty(t, service.records)
}
