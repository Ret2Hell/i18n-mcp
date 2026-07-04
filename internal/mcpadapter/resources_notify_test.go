package mcpadapter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUniqueResourceURIsCollapsesDuplicatesAndEmpty(t *testing.T) {
	got := uniqueResourceURIs([]string{"", "i18n://analysis/diff", "i18n://analysis/diff", "i18n://analysis/usage"})

	require.Equal(t, []string{"i18n://analysis/diff", "i18n://analysis/usage"}, got)
}
