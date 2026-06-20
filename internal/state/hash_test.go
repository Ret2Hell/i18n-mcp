package state

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSourceHashIsDeterministic(t *testing.T) {
	first := SourceHash("Hello {name}")
	second := SourceHash("Hello {name}")

	require.Equal(t, first, second)
	_, ok := strings.CutPrefix(first, "sha256:")
	require.True(t, ok)
}

func TestSourceHashChangesWhenValueChanges(t *testing.T) {
	require.NotEqual(t, SourceHash("Hello"), SourceHash("Hello!"))
}

func TestSourceAndTargetHashesUseDifferentVersions(t *testing.T) {
	require.NotEqual(t, SourceHash("Bonjour"), TargetHash("Bonjour"))
}
