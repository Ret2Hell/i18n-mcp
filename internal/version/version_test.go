package version

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {
	got := Get()
	require.Equal(t, AppName, got.Name)
	require.Equal(t, Version, got.Version)
	require.Equal(t, Commit, got.Commit)
	require.Equal(t, Date, got.Built)
}
