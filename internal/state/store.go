package state

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
)

// Store reads and writes the translation state file.
type Store struct {
	guard *fsutil.Guard
	path  string
}

// NewStore creates a Store at the default state path.
func NewStore(guard *fsutil.Guard) *Store {
	return NewStoreAt(guard, DefaultStatePath)
}

// NewStoreAt creates a Store at path.
func NewStoreAt(guard *fsutil.Guard, path string) *Store {
	return &Store{guard: guard, path: cmp.Or(path, DefaultStatePath)}
}

// Path returns the configured state file path.
func (s *Store) Path() string {
	return s.path
}

// Load reads the state file from disk.
func (s *Store) Load(ctx context.Context) (File, error) {
	_ = ctx
	resolved, err := s.guard.Resolve(s.path)
	if err != nil {
		return File{}, err
	}

	info, err := os.Lstat(resolved)
	if errors.Is(err, os.ErrNotExist) {
		return EmptyFile(), nil
	}
	if err != nil {
		return File{}, fmt.Errorf("stat state file: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return File{}, fmt.Errorf("state file symlink is not supported: %s", s.path)
	}

	data, err := os.ReadFile(resolved)
	if err != nil {
		return File{}, fmt.Errorf("read state file: %w", err)
	}

	var file File
	if err := json.Unmarshal(data, &file); err != nil {
		return File{}, fmt.Errorf("parse state file %s: %w", s.path, err)
	}
	if file.Version != CurrentVersion {
		return File{}, fmt.Errorf("unsupported state version %d", file.Version)
	}
	file.Normalize()
	return file, nil
}

// Save writes the state file to disk.
func (s *Store) Save(ctx context.Context, file File) error {
	_ = ctx
	file.Normalize()
	file.Version = CurrentVersion
	file.UpdatedAt = time.Now().UTC()

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state file: %w", err)
	}
	data = append(data, '\n')
	return fsutil.AtomicWriteFile(s.guard, s.path, data, 0o600)
}
