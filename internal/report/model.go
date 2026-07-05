package report

import (
	"context"
	"time"

	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/Ret2Hell/i18n-mcp/internal/deadkey"
	"github.com/Ret2Hell/i18n-mcp/internal/diff"
	"github.com/Ret2Hell/i18n-mcp/internal/locale"
	"github.com/Ret2Hell/i18n-mcp/internal/scanner"
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
	Format       Format           `json:"format,omitempty" jsonschema:"report format: json or markdown"`
	RefreshUsage bool             `json:"refreshUsage,omitempty" jsonschema:"rerun usage scan before report generation"`
	Progress     ProgressReporter `json:"-"`
}

// GenerateOutput contains a generated report and rendered text.
type GenerateOutput struct {
	Report Report `json:"report"`
	Format Format `json:"format"`
	Text   string `json:"text,omitempty"`
}

// Report combines inventory, diff, usage, and dead-key analyses.
type Report struct {
	GeneratedAt time.Time        `json:"generatedAt"`
	ProjectRoot string           `json:"projectRoot"`
	Config      config.Resolved  `json:"config"`
	Inventory   locale.Inventory `json:"inventory"`
	Diff        diff.Report      `json:"diff"`
	Usage       scanner.Report   `json:"usage"`
	DeadKeys    deadkey.Report   `json:"deadKeys"`
	Summary     Summary          `json:"summary"`
	Warnings    []string         `json:"warnings,omitzero"`
}

// Summary contains aggregate report counts.
type Summary struct {
	Locales        int `json:"locales"`
	Namespaces     int `json:"namespaces"`
	SourceKeys     int `json:"sourceKeys"`
	TargetKeys     int `json:"targetKeys"`
	Missing        int `json:"missing"`
	Stale          int `json:"stale"`
	Invalid        int `json:"invalid"`
	Extra          int `json:"extra"`
	Unknown        int `json:"unknown"`
	ProbablyUnused int `json:"probablyUnused"`
	MaybeDynamic   int `json:"maybeDynamic"`
	Warnings       int `json:"warnings"`
}
