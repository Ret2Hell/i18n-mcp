package diff

import (
	"context"
	"maps"
	"slices"
	"strings"

	"github.com/Ret2Hell/i18n-mcp/internal/locale"
	"github.com/Ret2Hell/i18n-mcp/internal/state"
	"github.com/Ret2Hell/i18n-mcp/internal/validate"
)

// inventoryAnalysis holds only operation-local indexes and source validation.
// Validation is shared across target locales, never across inventory snapshots.
type inventoryAnalysis struct {
	service          *Service
	inventory        locale.Inventory
	state            state.File
	sources, targets map[string]locale.Unit
	checkedSources   map[string][]validate.Issue
}

func matchingIdentities(inv locale.Inventory, filter Filter) []string {
	identities := map[string]struct{}{}
	for _, unit := range inv.Units {
		if len(filter.Namespaces) > 0 && !slices.Contains(filter.Namespaces, unit.Namespace) {
			continue
		}
		if len(filter.Keys) > 0 && !slices.Contains(filter.Keys, unit.Key) {
			continue
		}
		if strings.HasPrefix(unit.Key, filter.KeyPrefix) {
			identities[sourceIdentity(unit.Namespace, unit.Key)] = struct{}{}
		}
	}
	return slices.Sorted(maps.Keys(identities))
}

func (a *inventoryAnalysis) classify(ctx context.Context, code, id string, report *Report) (KeyDiff, bool) {
	namespace, key, _ := strings.Cut(id, "\x00")
	source, hasSource := a.sources[id]
	target, hasTarget := a.targets[targetIdentity(code, namespace, key)]
	var sourceIssues []validate.Issue
	if hasSource {
		sourceIssues = a.sourceIssues(ctx, id, source, report)
	}
	switch {
	case len(sourceIssues) > 0:
		return KeyDiff{Locale: code, Namespace: namespace, Key: key, Status: SourceInvalid, SourceValue: source.Value, TargetValue: target.Value, SourceHash: state.SourceHash(source.Value), SourceFilePath: source.FilePath, TargetFilePath: target.FilePath, SourceValidation: sourceIssues}, true
	case hasSource && hasTarget:
		return a.service.classifyExisting(ctx, a.inventory, source, target, a.state), true
	case hasSource:
		return KeyDiff{Locale: code, Namespace: namespace, Key: key, Status: Missing, SourceValue: source.Value, SourceHash: state.SourceHash(source.Value), SourceFilePath: source.FilePath}, true
	case hasTarget:
		return KeyDiff{Locale: code, Namespace: namespace, Key: key, Status: Extra, TargetValue: target.Value, TargetFilePath: target.FilePath}, true
	default:
		return KeyDiff{}, false
	}
}

func (a *inventoryAnalysis) sourceIssues(ctx context.Context, id string, source locale.Unit, report *Report) []validate.Issue {
	if issues, checked := a.checkedSources[id]; checked {
		return issues
	}
	policy := a.inventory.Validation.ForKey(source.Namespace, source.Key)
	issues := validate.CheckSource(ctx, source.Value, validate.Options{MessageFormat: policy.MessageFormat, Printf: policy.Printf})
	for i := range issues {
		issues[i].Locale = a.inventory.SourceLocale
		issues[i].Namespace = source.Namespace
		issues[i].Key = source.Key
	}
	a.checkedSources[id] = issues
	if len(issues) > 0 {
		report.SourceIssueCount++
		if len(report.SourceIssues) < 20 {
			report.SourceIssues = append(report.SourceIssues, issues[0])
		}
	}
	return issues
}
