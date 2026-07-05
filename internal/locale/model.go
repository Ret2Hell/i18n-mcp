package locale

import "encoding/json"

// FileRef identifies a locale file matched by a configured pattern.
type FileRef struct {
	Locale    string `json:"locale"`
	Namespace string `json:"namespace,omitzero"`
	Path      string `json:"path"`
	Pattern   string `json:"pattern"`
}

// Unit is one string translation unit from a locale file.
type Unit struct {
	Locale    string   `json:"locale"`
	Namespace string   `json:"namespace"`
	Key       string   `json:"key"`
	Value     string   `json:"value"`
	FilePath  string   `json:"filePath"`
	JSONPath  []string `json:"jsonPath"`
}

// Warning describes a non-fatal locale inventory issue.
type Warning struct {
	Code      string   `json:"code"`
	Message   string   `json:"message"`
	Locale    string   `json:"locale,omitzero"`
	Namespace string   `json:"namespace,omitzero"`
	FilePath  string   `json:"filePath,omitzero"`
	Key       string   `json:"key,omitzero"`
	JSONPath  []string `json:"jsonPath,omitzero"`
}

// FlattenResult contains flattened translation units and warnings.
type FlattenResult struct {
	Units    []Unit    `json:"units"`
	Warnings []Warning `json:"warnings,omitzero"`
}

// Inventory summarizes discovered locale files and translation units.
type Inventory struct {
	SourceLocale      string                    `json:"sourceLocale"`
	TargetLocales     []string                  `json:"targetLocales"`
	Locales           []string                  `json:"locales"`
	Namespaces        []string                  `json:"namespaces"`
	Files             []FileSummary             `json:"files"`
	Units             []Unit                    `json:"units,omitzero"`
	CountsByLocale    map[string]int            `json:"countsByLocale"`
	CountsByNamespace map[string]int            `json:"countsByNamespace"`
	Warnings          []Warning                 `json:"warnings,omitzero"`
	Duplicates        []DuplicateNamespaceIssue `json:"duplicates,omitzero"`
}

// FileSummary summarizes a discovered locale file.
type FileSummary struct {
	FileRef
	StringKeys int `json:"stringKeys"`
	Bytes      int `json:"bytes"`
}

// DuplicateNamespaceIssue reports files that map to the same locale namespace.
type DuplicateNamespaceIssue struct {
	Locale    string   `json:"locale"`
	Namespace string   `json:"namespace"`
	Paths     []string `json:"paths"`
}

// NamespaceContent contains raw files and units for one locale namespace.
type NamespaceContent struct {
	Locale    string           `json:"locale"`
	Namespace string           `json:"namespace"`
	Files     []FileSummary    `json:"files"`
	RawFiles  []RawFileContent `json:"rawFiles"`
	Units     []Unit           `json:"units"`
	Warnings  []Warning        `json:"warnings,omitzero"`
}

// RawFileContent contains raw JSON for a locale file.
type RawFileContent struct {
	Path string          `json:"path"`
	JSON json.RawMessage `json:"json"`
}
