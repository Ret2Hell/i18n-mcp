package validate

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

type Issue struct {
	Code      string   `json:"code"`
	Message   string   `json:"message"`
	Severity  Severity `json:"severity"`
	Locale    string   `json:"locale,omitzero"`
	Namespace string   `json:"namespace,omitzero"`
	Key       string   `json:"key,omitzero"`
}

type Result struct {
	OK       bool    `json:"ok"`
	Issues   []Issue `json:"issues,omitzero"`
	Warnings []Issue `json:"warnings,omitzero"`
}

type Service struct{}

func NewService() *Service {
	return &Service{}
}

type Pair struct {
	SourceLocale string `json:"sourceLocale,omitzero"`
	Locale       string `json:"locale,omitzero"`
	Namespace    string `json:"namespace,omitzero"`
	Key          string `json:"key,omitzero"`
	Source       string `json:"source"`
	Target       string `json:"target"`
}
