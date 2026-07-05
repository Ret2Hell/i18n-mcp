package deadkey

import (
	"time"

	"github.com/Ret2Hell/i18n-mcp/internal/scanner"
)

// Status classifies whether a source translation key is still used.
type Status string

// Dead-key statuses.
const (
	StatusUsed           Status = "used"
	StatusProbablyUnused Status = "probably_unused"
	StatusMaybeDynamic   Status = "maybe_dynamic"
	StatusIgnored        Status = "ignored"
	StatusKept           Status = "kept"
)

// ReportInput configures dead-key classification.
type ReportInput struct {
	RefreshUsage bool     `json:"refreshUsage,omitempty" jsonschema:"rerun usage scan before classifying keys"`
	Namespaces   []string `json:"namespaces,omitzero" jsonschema:"namespaces to include"`
	Keys         []string `json:"keys,omitzero" jsonschema:"keys to include"`
	IncludeUsed  bool     `json:"includeUsed,omitempty" jsonschema:"include keys classified as used"`
}

// Report contains dead-key classification results.
type Report struct {
	SourceLocale string         `json:"sourceLocale"`
	Items        []Item         `json:"items"`
	Summary      Summary        `json:"summary"`
	Usage        scanner.Report `json:"usage"`
	Warnings     []string       `json:"warnings,omitzero"`
	GeneratedAt  time.Time      `json:"generatedAt"`
}

// Item describes one source key's dead-key classification.
type Item struct {
	Namespace      string                `json:"namespace"`
	Key            string                `json:"key"`
	FullKey        string                `json:"fullKey"`
	SourceValue    string                `json:"sourceValue,omitempty"`
	SourceFilePath string                `json:"sourceFilePath,omitempty"`
	Status         Status                `json:"status"`
	Confidence     scanner.Confidence    `json:"confidence"`
	Evidence       []scanner.Evidence    `json:"evidence,omitzero"`
	DynamicHints   []scanner.DynamicHint `json:"dynamicHints,omitzero"`
	Reasons        []string              `json:"reasons,omitzero"`
}

// Summary counts dead-key classification statuses.
type Summary struct {
	Total          int `json:"total"`
	Used           int `json:"used"`
	ProbablyUnused int `json:"probablyUnused"`
	MaybeDynamic   int `json:"maybeDynamic"`
	Ignored        int `json:"ignored"`
	Kept           int `json:"kept"`
}
