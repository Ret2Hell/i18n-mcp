package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultsFormat(t *testing.T) {
	got := Defaults()

	require.False(t, got.Format.SortKeys)
	require.Equal(t, 2, got.Format.Indent)
	require.True(t, got.Format.TrailingNewline)
}
