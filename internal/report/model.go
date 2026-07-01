package report

import (
	"time"

	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/Ret2Hell/i18n-mcp/internal/deadkey"
	"github.com/Ret2Hell/i18n-mcp/internal/diff"
	"github.com/Ret2Hell/i18n-mcp/internal/locale"
	"github.com/Ret2Hell/i18n-mcp/internal/scanner"
)

type Format string

const (
	FormatJSON     Format = "json"
	FormatMarkdown Format = "markdown"
)

type GenerateInput struct {
	Format       Format `json:"format,omitempty" jsonschema:"report format: json or markdown"`
	RefreshUsage bool   `json:"refreshUsage,omitempty" jsonschema:"rerun usage scan before report generation"`
}

type GenerateOutput struct {
	Report Report `json:"report"`
	Format Format `json:"format"`
	Text   string `json:"text,omitempty"`
}

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
