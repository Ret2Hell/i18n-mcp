package translate

import "github.com/Ret2Hell/i18n-mcp/internal/fsutil"

// PreviewEdits returns changed-file previews for file edits.
func PreviewEdits(edits []FileEdit) []ChangedFile {
	out := make([]ChangedFile, 0, len(edits))
	for _, edit := range edits {
		diffText := fsutil.UnifiedDiff(edit.Path, edit.Before, edit.After)
		out = append(out, ChangedFile{
			Path:    edit.Path,
			Diff:    diffText,
			Changed: diffText != "",
		})
	}
	return out
}
