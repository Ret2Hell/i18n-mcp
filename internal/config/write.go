package config

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
)

// WriteInput describes a requested configuration write.
type WriteInput struct {
	Config File  `json:"config" jsonschema:"complete .i18n-mcp.json config to write"`
	DryRun *bool `json:"dryRun,omitzero" jsonschema:"when true, preview changes without writing"`
	Apply  bool  `json:"apply,omitzero" jsonschema:"must be true to write .i18n-mcp.json"`
}

// WriteOutput describes the outcome of a configuration write.
type WriteOutput struct {
	DryRun      bool             `json:"dryRun"`
	Applied     bool             `json:"applied"`
	ConfigPath  string           `json:"configPath"`
	Validation  ValidationResult `json:"validation"`
	ChangedFile ChangedFile      `json:"changedFile"`
	Warnings    []string         `json:"warnings,omitzero"`
}

// ChangedFile describes the diff and write status for a file.
type ChangedFile struct {
	Path    string `json:"path"`
	Diff    string `json:"diff,omitzero"`
	Changed bool   `json:"changed"`
	Written bool   `json:"written,omitzero"`
}

// DryRunValue returns the effective dry-run setting.
func (in WriteInput) DryRunValue() bool {
	if in.DryRun != nil {
		return *in.DryRun || !in.Apply
	}
	return !in.Apply
}

func (s *Service) Write(ctx context.Context, in WriteInput) (WriteOutput, error) {
	dryRun := in.DryRunValue()
	path := s.writePath()
	displayPath, err := s.displayPath(path)
	if err != nil {
		return WriteOutput{}, err
	}
	after, err := renderConfig(in.Config)
	if err != nil {
		return WriteOutput{}, err
	}
	before, err := s.readExistingConfig(path)
	if err != nil {
		return WriteOutput{}, err
	}

	resolvedPath, err := s.guard.Resolve(path)
	if err != nil {
		return WriteOutput{}, err
	}
	proposed := Resolved{File: in.Config, ProjectRoot: s.guard.Root(), ConfigPath: resolvedPath, Exists: true}
	validation := s.Validate(ctx, proposed)
	out := WriteOutput{
		DryRun:     dryRun,
		ConfigPath: displayPath,
		Validation: validation,
		ChangedFile: ChangedFile{
			Path:    displayPath,
			Diff:    fsutil.UnifiedDiff(displayPath, before, after),
			Changed: !bytes.Equal(before, after),
		},
	}
	if !validation.Valid {
		return out, nil
	}
	if dryRun {
		return out, nil
	}
	if !out.ChangedFile.Changed {
		return out, nil
	}
	perm := os.FileMode(0o600)
	if resolvedInfo, err := os.Stat(resolvedPath); err == nil {
		perm = resolvedInfo.Mode().Perm()
	}
	if err := fsutil.AtomicWriteFile(s.guard, path, after, perm); err != nil {
		return out, err
	}
	out.Applied = true
	out.ChangedFile.Written = true
	return out, nil
}

func (s *Service) writePath() string {
	return cmp.Or(s.configPath, DefaultConfigFile)
}

func (s *Service) displayPath(path string) (string, error) {
	resolved, err := s.guard.Resolve(path)
	if err != nil {
		return "", err
	}
	rel, err := s.guard.Rel(resolved)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(rel), nil
}

func renderConfig(cfg File) ([]byte, error) {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func (s *Service) readExistingConfig(path string) ([]byte, error) {
	resolved, err := s.guard.Resolve(path)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(resolved)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return data, nil
}
