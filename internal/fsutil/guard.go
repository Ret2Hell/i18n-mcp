package fsutil

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Guard confines filesystem operations to a project root.
type Guard struct {
	root string
}

// NewGuard creates a Guard for root. The root is made absolute, symlinks in the
// root itself are resolved, and the final path must exist and be a directory.
// Resolving the root up front makes later containment checks compare against the
// real project directory rather than a symlink alias.
func NewGuard(root string) (*Guard, error) {
	root = cmp.Or(root, ".")
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	realRoot, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, fmt.Errorf("resolve project root: %w", err)
	}
	info, err := os.Stat(realRoot)
	if err != nil {
		return nil, fmt.Errorf("stat project root: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("project root is not a directory: %s", realRoot)
	}
	return &Guard{root: filepath.Clean(realRoot)}, nil
}

// Root returns the guarded project root.
func (g *Guard) Root() string {
	return g.root
}

// Resolve returns an absolute, cleaned path that is guaranteed to be inside the
// guarded root. Relative paths are interpreted from the root; absolute paths are
// allowed only when they already point inside the root.
//
// Resolve does not require the target to exist and does not resolve symlinks in
// the final path. That makes it suitable for create/write paths. For existing
// files that may be symlinks, use ResolveExisting instead.
func (g *Guard) Resolve(path string) (string, error) {
	if path == "" {
		return g.root, nil
	}

	var candidate string
	if filepath.IsAbs(path) {
		candidate = filepath.Clean(path)
	} else {
		candidate = filepath.Join(g.root, path)
	}

	abs, err := filepath.Abs(candidate)
	if err != nil {
		return "", err
	}
	clean := filepath.Clean(abs)
	if err := g.ensureInside(clean); err != nil {
		return "", err
	}
	return clean, nil
}

// ResolveExisting resolves path like Resolve, then evaluates symlinks in the
// final target and checks containment again. This catches symlink escapes such
// as a file inside the project that points to /etc/passwd.
//
// The target must exist because filepath.EvalSymlinks requires an existing path.
func (g *Guard) ResolveExisting(path string) (string, error) {
	resolved, err := g.Resolve(path)
	if err != nil {
		return "", err
	}
	real, err := filepath.EvalSymlinks(resolved)
	if err != nil {
		return "", err
	}
	clean := filepath.Clean(real)
	if err := g.ensureInside(clean); err != nil {
		return "", err
	}
	return clean, nil
}

// Rel resolves path safely and returns it relative to the guarded root.
func (g *Guard) Rel(path string) (string, error) {
	resolved, err := g.Resolve(path)
	if err != nil {
		return "", err
	}
	return filepath.Rel(g.root, resolved)
}

// ensureInside rejects paths whose relative form climbs out of the root. It is
// based on filepath.Rel rather than string prefix checks so similarly named
// directories, such as /tmp/app and /tmp/app-evil, are not confused.
func (g *Guard) ensureInside(path string) error {
	rel, err := filepath.Rel(g.root, path)
	if err != nil {
		return err
	}
	_, climbsOut := strings.CutPrefix(rel, ".."+string(filepath.Separator))
	if rel == ".." || climbsOut || filepath.IsAbs(rel) {
		return fmt.Errorf("path escapes project root: %s", path)
	}
	return nil
}
