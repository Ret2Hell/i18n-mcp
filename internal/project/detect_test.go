package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
	"github.com/stretchr/testify/require"
)

func TestDetectNextJSUsesCurrentNextConventions(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "package.json", `{
		"scripts": {"dev": "next dev", "build": "next build --webpack", "test": "vitest"},
		"dependencies": {"next": "^16.2.0", "react": "^19.2.0", "react-dom": "^19.2.0"}
	}`)
	writeFile(t, root, "next.config.ts", "export default {}\n")
	writeFile(t, root, "next.config.cjs", "module.exports = {}\n")
	writeFile(t, root, "next.config.cts", "export default {}\n")
	writeFile(t, root, "next-env.d.ts", "/// <reference types=\"next\" />\n")
	writeFile(t, root, filepath.Join("src", "app", "page.tsx"), "export default function Page() { return null }\n")
	writeFile(t, root, filepath.Join("src", "proxy.ts"), "export function proxy() {}\n")
	writeFile(t, root, filepath.Join("src", "instrumentation.ts"), "export function register() {}\n")

	got := detectProject(t, root)

	require.True(t, got.NextJS.LooksLikeNextJS)
	require.True(t, got.NextJS.PackageJSON)
	require.Equal(t, filepath.Join(root, "package.json"), got.NextJS.PackageJSONPath)
	require.Equal(t, "^16.2.0", got.NextJS.NextDependency)
	require.Equal(t, "^19.2.0", got.NextJS.ReactDependency)
	require.Equal(t, "^19.2.0", got.NextJS.ReactDOMDependency)
	require.Equal(t, []string{"build", "dev"}, got.NextJS.NextScripts)
	require.Equal(t, []string{filepath.Join(root, "next.config.ts")}, got.NextJS.NextConfigFiles)
	require.ElementsMatch(t, []string{filepath.Join(root, "next.config.cjs"), filepath.Join(root, "next.config.cts")}, got.NextJS.UnsupportedNextConfigFiles)
	require.True(t, got.NextJS.NextEnvDTS)
	require.Equal(t, []string{filepath.Join(root, "src", "proxy.ts")}, got.NextJS.ProxyFiles)
	require.Equal(t, []string{filepath.Join(root, "src", "instrumentation.ts")}, got.NextJS.InstrumentationFiles)
	require.True(t, got.NextJS.SrcDir)
	require.True(t, got.NextJS.AppDir)
	require.True(t, got.NextJS.AppRouter)
	require.False(t, got.NextJS.PagesDir)
	require.False(t, got.NextJS.PagesRouter)
	require.Empty(t, got.Warnings)
}

func TestDetectNextJSRequiresRouterPathsToBeDirectories(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "app", "not a directory\n")
	writeFile(t, root, "pages", "not a directory\n")

	got := detectProject(t, root)

	require.False(t, got.NextJS.AppDir)
	require.False(t, got.NextJS.PagesDir)
	require.False(t, got.NextJS.LooksLikeNextJS)
	require.Contains(t, got.Warnings, "project root does not look like a Next.js app")
}

func detectProject(t *testing.T, root string) Detection {
	t.Helper()
	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	got, err := NewService(guard).Detect(t.Context(), DetectOptions{})
	require.NoError(t, err)
	return got
}

func writeFile(t *testing.T, root, name, data string) {
	t.Helper()
	path := filepath.Join(root, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(data), 0o644))
}
