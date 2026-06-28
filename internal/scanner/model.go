package scanner

import "time"

type Confidence string

const (
	ConfidenceExact  Confidence = "exact"
	ConfidenceHigh   Confidence = "high"
	ConfidenceMedium Confidence = "medium"
	ConfidenceLow    Confidence = "low"
)

type ScanInput struct {
	Files []string `json:"files,omitzero" jsonschema:"optional project-relative source files to scan"`
}

type Report struct {
	FilesScanned int           `json:"filesScanned"`
	Files        []SourceFile  `json:"files"`
	Usages       []Usage       `json:"usages"`
	DynamicHints []DynamicHint `json:"dynamicHints,omitzero"`
	Warnings     []string      `json:"warnings,omitzero"`
	GeneratedAt  time.Time     `json:"generatedAt"`
}

type SourceFile struct {
	Path  string `json:"path"`
	Bytes int    `json:"bytes"`
}

type Usage struct {
	Namespace string     `json:"namespace,omitempty"`
	Key       string     `json:"key"`
	FullKey   string     `json:"fullKey"`
	Kind      string     `json:"kind"`
	Evidence  []Evidence `json:"evidence"`
}

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
