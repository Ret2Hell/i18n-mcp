package diff

import "github.com/Ret2Hell/i18n-mcp/internal/validate"

// KeyStatus classifies the relationship between source and target keys.
type KeyStatus string

// Key diff statuses.
const (
	Current KeyStatus = "current"
	Missing KeyStatus = "missing"
	Stale   KeyStatus = "stale"
	Extra   KeyStatus = "extra"
	Invalid KeyStatus = "invalid"
	Unknown KeyStatus = "unknown"
)

// KeyDiff describes one source/target key comparison.
type KeyDiff struct {
	Locale             string            `json:"locale"`
	Namespace          string            `json:"namespace"`
	Key                string            `json:"key"`
	Status             KeyStatus         `json:"status"`
	SourceValue        string            `json:"sourceValue,omitzero"`
	TargetValue        string            `json:"targetValue,omitzero"`
	SourceHash         string            `json:"sourceHash,omitzero"`
	TranslatedFromHash string            `json:"translatedFromHash,omitzero"`
	TargetHash         string            `json:"targetHash,omitzero"`
	SourceFilePath     string            `json:"sourceFilePath,omitzero"`
	TargetFilePath     string            `json:"targetFilePath,omitzero"`
	Validation         []validate.Issue  `json:"validation,omitzero"`
	Metadata           map[string]string `json:"metadata,omitzero"`
}

// Report contains locale diff analysis results.
type Report struct {
	SourceLocale  string    `json:"sourceLocale"`
	TargetLocales []string  `json:"targetLocales"`
	Summary       Summary   `json:"summary"`
	Items         []KeyDiff `json:"items"`
	Warnings      []string  `json:"warnings,omitzero"`
}

// Summary counts key diff statuses.
type Summary struct {
	Total    int                     `json:"total"`
	Current  int                     `json:"current"`
	Missing  int                     `json:"missing"`
	Stale    int                     `json:"stale"`
	Extra    int                     `json:"extra"`
	Invalid  int                     `json:"invalid"`
	Unknown  int                     `json:"unknown"`
	ByLocale map[string]StatusCounts `json:"byLocale,omitzero"`
}

// StatusCounts counts key statuses for one locale.
type StatusCounts struct {
	Current int `json:"current"`
	Missing int `json:"missing"`
	Stale   int `json:"stale"`
	Extra   int `json:"extra"`
	Invalid int `json:"invalid"`
	Unknown int `json:"unknown"`
}
