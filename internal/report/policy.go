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
	if policy.FailOnDeadKeys && r.Summary.ProbablyUnused > 0 {
		failures = append(failures, Failure{Code: "dead_keys", Message: "probably unused keys detected", Count: r.Summary.ProbablyUnused})
	}
	return failures
}
