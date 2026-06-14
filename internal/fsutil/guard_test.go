package fsutil

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGuardResolveInsideRoot(t *testing.T) {
	root := t.TempDir()
	guard, err := NewGuard(root)
	require.NoError(t, err)

	got, err := guard.Resolve("messages/en.json")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(guard.Root(), "messages", "en.json"), got)
}

func TestGuardRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	guard, err := NewGuard(root)
	require.NoError(t, err)

	_, err = guard.Resolve("../outside.json")
	require.Error(t, err)
}

func TestGuardRejectsAbsoluteOutsideRoot(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.json")
	guard, err := NewGuard(root)
	require.NoError(t, err)

	_, err = guard.Resolve(outside)
	require.Error(t, err)
}

func TestGuardRejectsSiblingWithSamePrefix(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "project")
	sibling := filepath.Join(parent, "project-evil")
	require.NoError(t, os.Mkdir(root, 0o700))
	require.NoError(t, os.Mkdir(sibling, 0o700))

	guard, err := NewGuard(root)
	require.NoError(t, err)

	_, err = guard.Resolve(filepath.Join(sibling, "secret.json"))
	require.Error(t, err)
}

func TestGuardAllowsAbsoluteInsideRoot(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, "messages", "en.json")
	guard, err := NewGuard(root)
	require.NoError(t, err)

	got, err := guard.Resolve(inside)
	require.NoError(t, err)
	require.Equal(t, inside, got)
}

func TestGuardResolveEmptyPathReturnsRoot(t *testing.T) {
	root := t.TempDir()
	guard, err := NewGuard(root)
	require.NoError(t, err)

	got, err := guard.Resolve("")
	require.NoError(t, err)
	require.Equal(t, guard.Root(), got)
}

func TestGuardRelRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	guard, err := NewGuard(root)
	require.NoError(t, err)

	_, err = guard.Rel("../outside.json")
	require.Error(t, err)
}

func TestGuardResolveExistingAllowsSymlinkInsideRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privileges on some Windows systems")
	}
	root := t.TempDir()
	target := filepath.Join(root, "messages", "en.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(target), 0o700))
	require.NoError(t, os.WriteFile(target, []byte(`{}`), 0o600))
	require.NoError(t, os.Symlink(target, filepath.Join(root, "link.json")))

	guard, err := NewGuard(root)
	require.NoError(t, err)

	got, err := guard.ResolveExisting("link.json")
	require.NoError(t, err)
	require.Equal(t, target, got)
}

func TestGuardResolveExistingRejectsSymlinkEscapes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privileges on some Windows systems")
	}
	root := t.TempDir()
	outside := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outside, "secret.json"), []byte(`{}`), 0o600))
	require.NoError(t, os.Symlink(filepath.Join(outside, "secret.json"), filepath.Join(root, "link.json")))

	guard, err := NewGuard(root)
	require.NoError(t, err)

	_, err = guard.ResolveExisting("link.json")
	require.Error(t, err)
}

func TestNewGuardResolvesRootSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privileges on some Windows systems")
	}
	parent := t.TempDir()
	realRoot := filepath.Join(parent, "real-project")
	linkRoot := filepath.Join(parent, "project-link")
	require.NoError(t, os.Mkdir(realRoot, 0o700))
	require.NoError(t, os.Symlink(realRoot, linkRoot))

	guard, err := NewGuard(linkRoot)
	require.NoError(t, err)
	require.Equal(t, realRoot, guard.Root())
}

func TestNewGuardRejectsFileRoot(t *testing.T) {
	rootFile := filepath.Join(t.TempDir(), "project.json")
	require.NoError(t, os.WriteFile(rootFile, []byte(`{}`), 0o600))

	_, err := NewGuard(rootFile)
	require.Error(t, err)
}
