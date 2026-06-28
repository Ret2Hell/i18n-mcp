package scanner

import (
	"cmp"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

var sourceExtensions = map[string]bool{
	".ts":  true,
	".tsx": true,
	".js":  true,
	".jsx": true,
	".mjs": true,
	".cjs": true,
}

var ignoredSourceDirs = map[string]bool{
	".git":         true,
	".next":        true,
	".turbo":       true,
	"build":        true,
	"coverage":     true,
	"dist":         true,
	"node_modules": true,
	"out":          true,
}

func (s *Service) DiscoverSourceFiles(ctx context.Context, requested []string) ([]SourceFile, error) {
	if len(requested) > 0 {
		return s.discoverRequested(ctx, requested)
	}
	return s.discoverAll(ctx)
}

func (s *Service) discoverAll(ctx context.Context) ([]SourceFile, error) {
	var files []SourceFile
	root := s.guard.Root()
	err := filepath.WalkDir(root, func(absPath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		name := entry.Name()
		if entry.IsDir() {
			if ignoredSourceDirs[name] {
				return filepath.SkipDir
			}
			return nil
		}
		if !sourceExtensions[strings.ToLower(filepath.Ext(name))] {
			return nil
		}
		relPath, err := s.guard.Rel(absPath)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		files = append(files, SourceFile{Path: filepath.ToSlash(relPath), Bytes: int(info.Size())})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sortSourceFiles(files)
	return files, nil
}

func (s *Service) discoverRequested(ctx context.Context, requested []string) ([]SourceFile, error) {
	_ = ctx
	files := make([]SourceFile, 0, len(requested))
	for _, relPath := range requested {
		if !sourceExtensions[strings.ToLower(filepath.Ext(relPath))] {
			continue
		}
		absPath, err := s.guard.ResolveExisting(relPath)
		if err != nil {
			return nil, err
		}
		info, err := os.Lstat(absPath)
		if err != nil {
			return nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 || info.IsDir() {
			continue
		}
		guardedRel, err := s.guard.Rel(absPath)
		if err != nil {
			return nil, err
		}
		files = append(files, SourceFile{Path: filepath.ToSlash(guardedRel), Bytes: int(info.Size())})
	}
	sortSourceFiles(files)
	return files, nil
}

func sortSourceFiles(files []SourceFile) {
	slices.SortFunc(files, func(a, b SourceFile) int {
		return cmp.Compare(a.Path, b.Path)
	})
}
