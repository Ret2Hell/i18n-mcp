package validate

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
)

func compareICUMessages(ctx context.Context, source, target string) []Issue {
	sourceNodes, sourceErr := parseICU(ctx, source)
	targetNodes, targetErr := parseICU(ctx, target)
	var issues []Issue
	if sourceErr != nil {
		issues = append(issues, Issue{Code: "icu_source_invalid", Message: sourceErr.Error(), Severity: SeverityError})
	}
	if targetErr != nil {
		issues = append(issues, Issue{Code: "icu_target_invalid", Message: targetErr.Error(), Severity: SeverityError})
	}
	if sourceErr != nil || targetErr != nil {
		return issues
	}
	sources, targets := icuArguments(sourceNodes), icuArguments(targetNodes)
	for _, name := range slices.Sorted(maps.Keys(sources)) {
		targetArgs, ok := targets[name]
		if !ok {
			issues = append(issues, icuIssue("icu_argument_missing", name, "target is missing argument"))
			continue
		}
		for _, sourceArg := range sources[name] {
			matching := false
			compatible := false
			for _, targetArg := range targetArgs {
				if sourceArg.kind != targetArg.kind {
					continue
				}
				compatible = true
				if sourceArg.offset != targetArg.offset {
					continue
				}
				selectors := true
				for selector := range sourceArg.branches {
					if sourceArg.kind != "select" && !strings.HasPrefix(selector, "=") && selector != "other" {
						continue
					}
					if _, ok := targetArg.branches[selector]; !ok {
						selectors = false
						break
					}
				}
				if selectors {
					matching = true
					break
				}
			}
			if !compatible {
				issues = append(issues, icuIssue("icu_argument_type_changed", name, "target changes argument type "+sourceArg.kind))
				break
			}
			if !matching {
				issues = append(issues, icuIssue("icu_selector_changed", name, "target must preserve plural offset, explicit numeric selectors and select choices"))
				break
			}
		}
	}
	for _, name := range slices.Sorted(maps.Keys(targets)) {
		if _, ok := sources[name]; !ok {
			issues = append(issues, icuIssue("icu_argument_extra", name, "target introduces an argument absent from source"))
			continue
		}
		for _, targetArg := range targets[name] {
			if !slices.ContainsFunc(sources[name], func(sourceArg icuNode) bool { return sourceArg.kind == targetArg.kind }) {
				issues = append(issues, icuIssue("icu_argument_type_changed", name, "target introduces an incompatible argument type"))
				break
			}
		}
	}
	return append(issues, compareICUMessageCounts(sourceNodes, targetNodes)...)
}

// Counts are compared within corresponding message bodies, never summed over
// mutually exclusive plural categories. Added locale-specific branches are valid.
func compareICUMessageCounts(source, target []icuNode) []Issue {
	counts := func(nodes []icuNode) map[string]int {
		out := map[string]int{}
		for _, node := range nodes {
			if node.name != "" {
				out[node.name+" ("+node.kind+")"]++
			}
		}
		return out
	}
	a, b := counts(source), counts(target)
	var issues []Issue
	for _, key := range slices.Sorted(maps.Keys(a)) {
		if b[key] > 0 && a[key] != b[key] {
			issues = append(issues, icuIssue("icu_argument_count_changed", key, "target changes argument count within a message body"))
		}
	}
	for _, node := range source {
		if len(node.branches) == 0 {
			continue
		}
		candidates := []icuNode{}
		for _, other := range target {
			if other.name == node.name && other.kind == node.kind {
				candidates = append(candidates, other)
			}
		}
		if len(candidates) != 1 {
			continue
		}
		for _, selector := range slices.Sorted(maps.Keys(node.branches)) {
			if body, ok := candidates[0].branches[selector]; ok {
				issues = append(issues, compareICUMessageCounts(node.branches[selector], body)...)
			}
		}
	}
	return issues
}

func icuIssue(code, name, message string) Issue {
	return Issue{Code: code, Message: fmt.Sprintf("%s %q", message, name), Severity: SeverityError}
}
