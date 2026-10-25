package fsutil

import (
	"bytes"
	"errors"
	"fmt"
	"os"
)

// VerifyFileVersion rejects known filesystem drift before a planned write. It is
// an optimistic check, not an OS-level cross-process transaction or file lock.
func VerifyFileVersion(guard *Guard, path string, before []byte, missing bool) error {
	resolved, err := guard.Resolve(path)
	if err != nil {
		return err
	}
	info, err := os.Lstat(resolved)
	if errors.Is(err, os.ErrNotExist) {
		if missing {
			return nil
		}
		return fmt.Errorf("file changed since preview: %s was removed", path)
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refuse to write through symlink: %s", path)
	}
	if missing {
		return fmt.Errorf("file changed since preview: %s was created", path)
	}
	current, err := os.ReadFile(resolved)
	if err != nil {
		return err
	}
	if !bytes.Equal(current, before) {
		return fmt.Errorf("file changed since preview: %s; review a new preview", path)
	}
	return nil
}
