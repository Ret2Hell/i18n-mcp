package config

import (
	"cmp"
	"slices"
	"strings"
)

// ValidationOverride selects an explicit policy for a namespace, exact key or
// literal key prefix. Printf is optional so explicit false overrides inheritance.
type ValidationOverride struct {
	Namespace     string `json:"namespace,omitzero"`
	Key           string `json:"key,omitzero"`
	KeyPrefix     string `json:"keyPrefix,omitzero"`
	MessageFormat string `json:"messageFormat,omitzero" jsonschema:"auto, icu, i18next, interpolation, or raw; empty inherits"`
	Printf        *bool  `json:"printf,omitzero"`
}

// ForKey composes matching rules from least to most specific. Exact keys win;
// then the longest literal prefix; namespace-specific wins a same-scope tie.
func (v ValidationConfig) ForKey(namespace, key string) ValidationConfig {
	var matches []ValidationOverride
	for _, r := range v.Overrides {
		if r.Namespace != "" && r.Namespace != namespace {
			continue
		}
		if r.Key != "" && r.Key != key {
			continue
		}
		if r.KeyPrefix != "" && !strings.HasPrefix(key, r.KeyPrefix) {
			continue
		}
		matches = append(matches, r)
	}
	rank := func(r ValidationOverride) int {
		if r.Key != "" {
			return 2
		}
		if r.KeyPrefix != "" {
			return 1
		}
		return 0
	}
	specificity := func(r ValidationOverride) int {
		if r.Namespace != "" {
			return 1
		}
		return 0
	}
	slices.SortFunc(matches, func(a, b ValidationOverride) int {
		return cmp.Or(cmp.Compare(rank(a), rank(b)), cmp.Compare(len(a.KeyPrefix), len(b.KeyPrefix)), cmp.Compare(specificity(a), specificity(b)))
	})
	result := ValidationConfig{MessageFormat: v.MessageFormat, Printf: v.Printf}
	for _, r := range matches {
		if r.MessageFormat != "" {
			result.MessageFormat = r.MessageFormat
		}
		if r.Printf != nil {
			result.Printf = *r.Printf
		}
	}
	return result
}
