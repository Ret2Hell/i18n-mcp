package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
	"github.com/stretchr/testify/require"
)

func TestResolveRoot(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "apps", "web")
	require.NoError(t, os.MkdirAll(child, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"private":true}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "not-a-dir"), []byte("file"), 0o644))

	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	service := NewService(guard)

	tests := []struct {
		name        string
		projectRoot string
		want        RootInfo
		wantErr     string
	}{
		{
			name:        "default root",
			projectRoot: "",
			want: RootInfo{
				ProjectRoot: root,
				Name:        filepath.Base(root),
			},
		},
		{
			name:        "explicit current root",
			projectRoot: ".",
			want: RootInfo{
				ProjectRoot: root,
				Name:        filepath.Base(root),
				Relative:    "",
			},
		},
		{
			name:        "nested root",
			projectRoot: filepath.Join("apps", "web"),
			want: RootInfo{
				ProjectRoot: child,
				Name:        "web",
				Relative:    filepath.Join("apps", "web"),
			},
		},
		{
			name:        "missing root",
			projectRoot: "missing",
			wantErr:     "no such file or directory",
		},
		{
			name:        "file root",
			projectRoot: "not-a-dir",
			wantErr:     "project root is not a directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := service.ResolveRoot(t.Context(), tt.projectRoot)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestRootUsesDefaultProjectRoot(t *testing.T) {
	root := t.TempDir()
	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)

	got, err := NewService(guard).Root(t.Context())
	require.NoError(t, err)
	require.Equal(t, RootInfo{ProjectRoot: root, Name: filepath.Base(root)}, got)
}
