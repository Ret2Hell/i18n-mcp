package fsutil

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func AtomicWriteFile(guard *Guard, relPath string, data []byte, perm os.FileMode) (err error) {
	target, err := guard.Resolve(relPath)
	if err != nil {
		return err
	}
	parent := filepath.Dir(target)

	if err := rejectSymlinkAncestors(guard.Root(), parent); err != nil {
		return err
	}
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return fmt.Errorf("create parent directory for %s: %w", relPath, err)
	}
	if err := rejectSymlinkAncestors(guard.Root(), parent); err != nil {
		return err
	}

	if info, err := os.Lstat(target); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refuse to write through symlink: %s", relPath)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat target %s: %w", relPath, err)
	}

	tmp, err := os.CreateTemp(parent, "."+filepath.Base(target)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file for %s: %w", relPath, err)
	}
	tmpName := tmp.Name()
	defer func() {
		if closeErr := tmp.Close(); closeErr != nil && !errors.Is(closeErr, os.ErrClosed) {
			err = errors.Join(err, fmt.Errorf("close temp file during cleanup for %s: %w", relPath, closeErr))
		}
		if removeErr := os.Remove(tmpName); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			err = errors.Join(err, fmt.Errorf("remove temp file during cleanup for %s: %w", relPath, removeErr))
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		return fmt.Errorf("write temp file for %s: %w", relPath, err)
	}
	if err := tmp.Chmod(perm); err != nil {
		return fmt.Errorf("chmod temp file for %s: %w", relPath, err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("sync temp file for %s: %w", relPath, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file for %s: %w", relPath, err)
	}
	if err := os.Rename(tmpName, target); err != nil {
		return fmt.Errorf("rename temp file for %s: %w", relPath, err)
	}
	return nil
}

func rejectSymlinkAncestors(root string, path string) error {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return err
	}
	if rel == "." {
		return nil
	}
	_, climbsOut := strings.CutPrefix(rel, ".."+string(filepath.Separator))
	if rel == ".." {
		return fmt.Errorf("path escapes project root: %s", path)
	}
	if climbsOut {
		return fmt.Errorf("path escapes project root: %s", path)
	}
	if filepath.IsAbs(rel) {
		return fmt.Errorf("path escapes project root: %s", path)
	}

	current := root
	for part := range strings.SplitSeq(rel, string(filepath.Separator)) {
		if part == "" || part == "." {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("stat path component %s: %w", current, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refuse to write through symlink path component: %s", current)
		}
	}
	return nil
}
