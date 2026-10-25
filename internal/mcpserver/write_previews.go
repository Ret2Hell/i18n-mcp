package mcpserver

import (
	"encoding/json"
	"fmt"
	"unicode/utf8"

	"github.com/Ret2Hell/i18n-mcp/internal/pagination"
	"github.com/Ret2Hell/i18n-mcp/internal/preview"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func writePreviewError(err error) *mcp.CallToolResult {
	message := err.Error()
	if len(message) > 4096 {
		message = message[:4096] + " [error text truncated]"
	}
	return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: message}}}
}

// Reserve headroom for final write flags and bounded filesystem error messages.
// This check occurs before writes, not after a successful operation.
func checkWritePreview(out any) error {
	raw, err := json.Marshal(out)
	if err != nil {
		return err
	}
	if len(raw) > pagination.ResponseBudget-(32<<10) {
		return fmt.Errorf("preview metadata/diagnostics exceed the response budget; narrow the operation before applying; no files were written")
	}
	return nil
}

func anyWritten(files []preview.File) bool {
	for _, file := range files {
		if file.Written {
			return true
		}
	}
	return false
}

func reviewedFiles(actual []preview.File, view preview.View) []preview.File {
	written := map[string]bool{}
	for _, file := range actual {
		written[file.Path] = file.Written
	}
	files := append([]preview.File(nil), view.Files...)
	for i := range files {
		files[i].Written = written[files[i].Path]
	}
	return files
}

func boundWriteWarnings(warnings []string, view preview.View) []string {
	count := min(len(warnings), 4)
	result := append([]string(nil), warnings[:count]...)
	if count < len(warnings) {
		view.Metadata.DiagnosticsTruncated = true
	}
	for i := range result {
		if len(result[i]) > 2048 {
			n := 2048
			for n > 0 && !utf8.ValidString(result[i][:n]) {
				n--
			}
			result[i] = result[i][:n] + " [warning truncated]"
			view.Metadata.DiagnosticsTruncated = true
		}
	}
	return result
}

func requireReview(view preview.View, token string) error {
	if view.Metadata.Truncated && token == "" {
		return fmt.Errorf("preview is truncated; retrieve the complete patches from preview.manifestUri/patchUri, review them, and repeat apply with previewToken=preview.token; no files were written")
	}
	return nil
}
