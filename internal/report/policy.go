package report

import "github.com/Ret2Hell/i18n-mcp/internal/config"

// Failure describes a CI policy failure from an audit report.
type Failure struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Count   int    `json:"count"`
}

// EvaluatePolicy returns failures for report counts that violate policy.
func EvaluatePolicy(r Report, policy config.CIConfig) []Failure {
	var failures []Failure
	if policy.FailOnMissing && r.Summary.Missing > 0 {
		failures = append(failures, Failure{Code: "missing", Message: "missing translations detected", Count: r.Summary.Missing})
	}
	if policy.FailOnStale && r.Summary.Stale > 0 {
		failures = append(failures, Failure{Code: "stale", Message: "stale translations detected", Count: r.Summary.Stale})
	}
	if policy.FailOnInvalid && r.Summary.Invalid > 0 {
		failures = append(failures, Failure{Code: "invalid", Message: "invalid translations detected", Count: r.Summary.Invalid})
	}
	if policy.FailOnInvalid && r.Summary.SourceInvalid > 0 {
		failures = append(failures, Failure{Code: "source_invalid", Message: "source message validation failed; review source syntax or explicit policy overrides", Count: r.Summary.SourceInvalid})
	}
	// FailOnDeadKeys is retained in config for compatibility but is not
	// evaluated while dead-key analysis is disabled.
	return failures
}
