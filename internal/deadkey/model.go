package deadkey

import (
	"time"

	"github.com/Ret2Hell/i18n-mcp/internal/scanner"
)

type Status string

const (
	StatusUsed           Status = "used"
	StatusProbablyUnused Status = "probably_unused"
	StatusMaybeDynamic   Status = "maybe_dynamic"
	StatusIgnored        Status = "ignored"
	StatusKept           Status = "kept"
)

type ReportInput struct {
	RefreshUsage bool     `json:"refreshUsage,omitempty" jsonschema:"rerun usage scan before classifying keys"`
	Namespaces   []string `json:"namespaces,omitempty" jsonschema:"namespaces to include"`
	Keys         []string `json:"keys,omitempty" jsonschema:"keys to include"`
	IncludeUsed  bool     `json:"includeUsed,omitempty" jsonschema:"include keys classified as used"`
}

type Report struct {
	SourceLocale string         `json:"sourceLocale"`
	Items        []Item         `json:"items"`
	Summary      Summary        `json:"summary"`
	Usage        scanner.Report `json:"usage"`
	Warnings     []string       `json:"warnings,omitempty"`
	GeneratedAt  time.Time      `json:"generatedAt"`
}

type Item struct {
	Namespace      string                `json:"namespace"`
	Key            string                `json:"key"`
	FullKey        string                `json:"fullKey"`
	SourceValue    string                `json:"sourceValue,omitempty"`
	SourceFilePath string                `json:"sourceFilePath,omitempty"`
	Status         Status                `json:"status"`
	Confidence     scanner.Confidence    `json:"confidence"`
	Evidence       []scanner.Evidence    `json:"evidence,omitempty"`
	DynamicHints   []scanner.DynamicHint `json:"dynamicHints,omitempty"`
	Reasons        []string              `json:"reasons,omitempty"`
}

type Summary struct {
	Total          int `json:"total"`
	Used           int `json:"used"`
	ProbablyUnused int `json:"probablyUnused"`
	MaybeDynamic   int `json:"maybeDynamic"`
	Ignored        int `json:"ignored"`
	Kept           int `json:"kept"`
}
