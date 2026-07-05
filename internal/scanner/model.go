package scanner

import (
	"context"
	"time"
)

// Confidence indicates how reliable a scanned usage signal is.
type Confidence string

// Confidence levels for scanner evidence.
const (
	ConfidenceExact  Confidence = "exact"
	ConfidenceHigh   Confidence = "high"
	ConfidenceMedium Confidence = "medium"
	ConfidenceLow    Confidence = "low"
)

// ProgressReporter receives scanner progress events.
type ProgressReporter interface {
	Step(ctx context.Context, message string, current int, total int)
}

// ScanInput configures a source scan.
type ScanInput struct {
	Files     []string         `json:"files,omitzero" jsonschema:"optional project-relative source files to scan"`
	Progress  ProgressReporter `json:"-"`
	BatchSize int              `json:"batchSize,omitzero" jsonschema:"optional progress notification batch size; defaults to 25"`
}

// Report contains source scan results.
type Report struct {
	FilesScanned int           `json:"filesScanned"`
	Files        []SourceFile  `json:"files"`
	Usages       []Usage       `json:"usages"`
	DynamicHints []DynamicHint `json:"dynamicHints,omitzero"`
	Warnings     []string      `json:"warnings,omitzero"`
	GeneratedAt  time.Time     `json:"generatedAt"`
}

// SourceFile describes a scanned source file.
type SourceFile struct {
	Path  string `json:"path"`
	Bytes int    `json:"bytes"`
}

// Usage describes a detected translation key usage.
type Usage struct {
	Namespace string     `json:"namespace,omitempty"`
	Key       string     `json:"key"`
	FullKey   string     `json:"fullKey"`
	Kind      string     `json:"kind"`
	Evidence  []Evidence `json:"evidence"`
}

// Evidence describes where a translation usage was found.
type Evidence struct {
	Namespace  string     `json:"namespace,omitempty"`
	Key        string     `json:"key"`
	FullKey    string     `json:"fullKey"`
	FilePath   string     `json:"file"`
	Line       int        `json:"line"`
	Column     int        `json:"column"`
	Snippet    string     `json:"snippet"`
	Pattern    string     `json:"pattern"`
	Confidence Confidence `json:"confidence"`
}

// DynamicHint describes a likely dynamic translation key usage.
type DynamicHint struct {
	Namespace  string     `json:"namespace,omitempty"`
	KeyPattern string     `json:"keyPattern,omitempty"`
	FilePath   string     `json:"file"`
	Line       int        `json:"line"`
	Column     int        `json:"column"`
	Snippet    string     `json:"snippet"`
	Pattern    string     `json:"pattern"`
	Confidence Confidence `json:"confidence"`
	Message    string     `json:"message"`
}
