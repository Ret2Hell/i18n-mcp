package config

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	json "github.com/Ret2Hell/i18n-mcp/internal/jsonutil"
	"os"
	"path/filepath"

	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
	"github.com/Ret2Hell/i18n-mcp/internal/preview"
)

// WriteInput describes a requested configuration write.
type WriteInput struct {
	PreviewToken string                                   `json:"previewToken,omitzero" jsonschema:"approval token from a reviewed preview; required for truncated previews"`
	BeforeWrite  func(context.Context, WriteOutput) error `json:"-"`
	Config       FileInput                                `json:"config" jsonschema:"replacement config; sourceLocale and localeFiles required; omitted optional fields use defaults"`
	DryRun       *bool                                    `json:"dryRun,omitzero" jsonschema:"when true, preview changes without writing"`
	Apply        bool                                     `json:"apply,omitzero" jsonschema:"must be true to write .i18n-mcp.json"`
}

// WriteOutput describes the outcome of a configuration write.
type WriteOutput struct {
	Preview     *preview.Metadata `json:"preview,omitzero"`
	DryRun      bool              `json:"dryRun"`
	Applied     bool              `json:"applied"`
	ConfigPath  string            `json:"configPath"`
	Validation  ValidationResult  `json:"validation"`
	ChangedFile ChangedFile       `json:"changedFile"`
	Warnings    []string          `json:"warnings,omitzero"`
}

// ChangedFile describes the diff and write status for a file.
type ChangedFile = preview.File

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
	cfg := NormalizeInput(in.Config)
	after, err := renderConfig(cfg)
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
	proposed := Resolved{File: cfg, ProjectRoot: s.guard.Root(), ConfigPath: resolvedPath, Exists: true}
	validation := s.Validate(ctx, proposed)
	out := WriteOutput{
		DryRun:     dryRun,
		ConfigPath: displayPath,
		Validation: validation,
		ChangedFile: ChangedFile{
			BeforeMissing: before == nil, BeforeSHA256: preview.HashContent(before), AfterSHA256: preview.HashContent(after),
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
	if in.BeforeWrite != nil {
		if err := in.BeforeWrite(ctx, out); err != nil {
			return out, err
		}
	}
	if err := ctx.Err(); err != nil {
		return out, err
	}
	if err := fsutil.VerifyFileVersion(s.guard, path, before, before == nil); err != nil {
		return out, err
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
