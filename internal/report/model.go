package report

import (
	"context"
	"time"

	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/Ret2Hell/i18n-mcp/internal/diff"
	"github.com/Ret2Hell/i18n-mcp/internal/locale"
)

// Format identifies a supported report output format.
type Format string

// Supported report formats.
const (
	FormatJSON     Format = "json"
	FormatMarkdown Format = "markdown"
)

// ProgressReporter reports coarse-grained progress for report generation.
type ProgressReporter interface {
	Step(ctx context.Context, message string, current int, total int)
}

// GenerateInput configures report generation.
type GenerateInput struct {
	Format Format `json:"format,omitempty" jsonschema:"report format: json or markdown"`
	// Full is reserved for explicit local CLI exports, never MCP input.
	Full bool `json:"-"`
	// RefreshUsage is retained for Go callers but ignored while scanning is disabled.
	RefreshUsage bool             `json:"-"`
	Progress     ProgressReporter `json:"-"`
}

// GenerateOutput contains a generated report and rendered text.
type GenerateOutput struct {
	Report Report `json:"report"`
	Format Format `json:"format"`
	Text   string `json:"-"`
}

// Report combines locale inventory and translation diff analysis.
type Report struct {
	DetailsOmitted bool             `json:"detailsOmitted,omitzero"`
	DetailResource string           `json:"detailResource,omitzero"`
	WarningsTotal  int              `json:"warningsTotal,omitzero"`
	GeneratedAt    time.Time        `json:"generatedAt"`
	ProjectRoot    string           `json:"projectRoot"`
	Config         config.Resolved  `json:"config"`
	Inventory      locale.Inventory `json:"inventory"`
	Diff           diff.Report      `json:"diff"`
	Summary        Summary          `json:"summary"`
	Warnings       []string         `json:"warnings,omitzero"`
}

// Summary contains aggregate report counts.
type Summary struct {
	Locales    int `json:"locales"`
	Namespaces int `json:"namespaces"`
	SourceKeys int `json:"sourceKeys"`
	TargetKeys int `json:"targetKeys"`
	Missing    int `json:"missing"`
	Stale      int `json:"stale"`
	Invalid    int `json:"invalid"`
	Extra      int `json:"extra"`
	Unknown    int `json:"unknown"`
	Warnings   int `json:"warnings"`
}
