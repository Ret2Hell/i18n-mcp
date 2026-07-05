package config

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
)

// Service loads, validates, and writes i18n MCP configuration.
type Service struct {
	guard      *fsutil.Guard
	configPath string
}

// NewService creates a configuration service scoped by guard.
func NewService(guard *fsutil.Guard, configPath string) *Service {
	return &Service{guard: guard, configPath: configPath}
}

// Resolve loads configuration from disk and merges it with defaults.
func (s *Service) Resolve(ctx context.Context) (Resolved, error) {
	_ = ctx
	cfg := Defaults()
	path := cmp.Or(s.configPath, DefaultConfigFile)

	resolvedPath, err := s.guard.Resolve(path)
	if err != nil {
		return Resolved{}, err
	}

	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Resolved{File: cfg, ProjectRoot: s.guard.Root(), ConfigPath: resolvedPath, Exists: false}, nil
		}
		return Resolved{}, fmt.Errorf("read config: %w", err)
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return Resolved{}, fmt.Errorf("parse config %s: %w", resolvedPath, err)
	}

	return Resolved{File: cfg, ProjectRoot: s.guard.Root(), ConfigPath: resolvedPath, Exists: true}, nil
}
