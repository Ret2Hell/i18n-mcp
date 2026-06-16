package project

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
)

type RootInfo struct {
	ProjectRoot string `json:"projectRoot"`
	Name        string `json:"name"`
	Relative    string `json:"relative,omitzero"`
}

type Service struct {
	guard *fsutil.Guard
}

func NewService(guard *fsutil.Guard) *Service {
	return &Service{guard: guard}
}

func (s *Service) Root(ctx context.Context) (RootInfo, error) {
	return s.ResolveRoot(ctx, "")
}

func (s *Service) ResolveRoot(ctx context.Context, projectRoot string) (RootInfo, error) {
	_ = ctx
	_, info, err := s.resolveGuard(projectRoot)
	if err != nil {
		return RootInfo{}, err
	}
	return info, nil
}

func (s *Service) guardFor(ctx context.Context, projectRoot string) (*fsutil.Guard, RootInfo, error) {
	_ = ctx
	return s.resolveGuard(projectRoot)
}

func (s *Service) resolveGuard(projectRoot string) (*fsutil.Guard, RootInfo, error) {
	if projectRoot == "" {
		root := s.guard.Root()
		return s.guard, RootInfo{
			ProjectRoot: root,
			Name:        filepath.Base(root),
		}, nil
	}

	resolved, err := s.guard.ResolveExisting(projectRoot)
	if err != nil {
		return nil, RootInfo{}, err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return nil, RootInfo{}, fmt.Errorf("stat project root: %w", err)
	}
	if !info.IsDir() {
		return nil, RootInfo{}, fmt.Errorf("project root is not a directory: %s", resolved)
	}

	subGuard, err := fsutil.NewGuard(resolved)
	if err != nil {
		return nil, RootInfo{}, err
	}
	rel, err := filepath.Rel(s.guard.Root(), subGuard.Root())
	if err != nil {
		return nil, RootInfo{}, err
	}
	if rel == "." {
		rel = ""
	}

	return subGuard, RootInfo{
		ProjectRoot: subGuard.Root(),
		Name:        filepath.Base(subGuard.Root()),
		Relative:    rel,
	}, nil
}
