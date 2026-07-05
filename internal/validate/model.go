package validate

// Severity classifies a validation issue as an error or warning.
type Severity string

// Validation severities.
const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// Issue describes one validation finding.
type Issue struct {
	Code      string   `json:"code"`
	Message   string   `json:"message"`
	Severity  Severity `json:"severity"`
	Locale    string   `json:"locale,omitzero"`
	Namespace string   `json:"namespace,omitzero"`
	Key       string   `json:"key,omitzero"`
}

// Result contains validation issues and warnings.
type Result struct {
	OK       bool    `json:"ok"`
	Issues   []Issue `json:"issues,omitzero"`
	Warnings []Issue `json:"warnings,omitzero"`
}

// Service validates translated strings.
type Service struct{}

// NewService creates a translation validation service.
func NewService() *Service {
	return &Service{}
}

// Pair contains a source string and target translation to validate.
type Pair struct {
	SourceLocale string `json:"sourceLocale,omitzero"`
	Locale       string `json:"locale,omitzero"`
	Namespace    string `json:"namespace,omitzero"`
	Key          string `json:"key,omitzero"`
	Source       string `json:"source"`
	Target       string `json:"target"`
}
