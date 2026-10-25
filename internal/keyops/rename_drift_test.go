package keyops_test

import (
	"context"
	"github.com/Ret2Hell/i18n-mcp/internal/keyops"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func TestRenameDetectsFileDriftAfterApprovalBeforeAnyWrite(t *testing.T) {
	a := newRenameFixtureApp(t)
	before := readRenameFile(t, a.ProjectRoot, "messages/en/auth.json")
	external := []byte("{\"login\":{\"title\":\"External change\"}}")
	out, err := a.KeyOps.Rename(t.Context(), keyops.RenameInput{Apply: true, Namespace: "auth", FromKey: "login.title", ToKey: "login.heading", BeforeWrite: func(context.Context, keyops.RenameOutput) error {
		return os.WriteFile(filepath.Join(a.ProjectRoot, "messages/fr/auth.json"), external, 0600)
	}})
	require.ErrorContains(t, err, "changed since preview")
	require.Zero(t, out.Renamed)
	require.Equal(t, before, readRenameFile(t, a.ProjectRoot, "messages/en/auth.json"))
	require.Equal(t, string(external), readRenameFile(t, a.ProjectRoot, "messages/fr/auth.json"))
}
