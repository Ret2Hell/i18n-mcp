package translate

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/Ret2Hell/i18n-mcp/internal/locale"
	"github.com/Ret2Hell/i18n-mcp/internal/state"
	"github.com/Ret2Hell/i18n-mcp/internal/validate"
)

// Validate validates proposed translations against source and rules.
func (s *Service) Validate(ctx context.Context, in ValidationInput) (ValidationOutput, error) {
	inv, err := s.locales.Inventory(ctx)
	if err != nil {
		return ValidationOutput{}, err
	}

	latest, hasLatest := s.LatestPlan()
	if in.BatchID != "" && (!hasLatest || latest.BatchID != in.BatchID) {
		return ValidationOutput{}, fmt.Errorf("unknown translation batch id %q", in.BatchID)
	}
	allowed := map[string]Item{}
	if hasLatest && (in.BatchID == "" || in.BatchID == latest.BatchID) {
		for _, item := range latest.Items {
			allowed[proposalIdentity(item.Locale, item.Namespace, item.Key)] = item
		}
	}

	sources := sourceUnits(inv)
	targetFiles := targetFileMap(inv)
	out := ValidationOutput{BatchID: in.BatchID}
	for _, proposal := range in.Translations {
		accepted, rejected := s.validateProposal(inv.SourceLocale, sources, targetFiles, allowed, in, proposal)
		if len(rejected.Issues) > 0 {
			out.Rejected = append(out.Rejected, rejected)
			continue
		}
		out.Accepted = append(out.Accepted, accepted)
	}
	sortValidationOutput(&out)
	out.Summary = ValidationSummary{Total: len(in.Translations), Accepted: len(out.Accepted), Rejected: len(out.Rejected)}
	return out, nil
}

func (s *Service) validateProposal(sourceLocale string, sources map[string]locale.Unit, targetFiles map[string]string, allowed map[string]Item, in ValidationInput, proposal ProposedTranslation) (ValidatedTranslation, RejectedTranslation) {
	rejected := RejectedTranslation{ProposedTranslation: proposal}
	identity := proposalIdentity(proposal.Locale, proposal.Namespace, proposal.Key)

	if strings.TrimSpace(proposal.Locale) == "" || strings.TrimSpace(proposal.Namespace) == "" || strings.TrimSpace(proposal.Key) == "" {
		rejected.Issues = append(rejected.Issues, proposalIssue(proposal, "proposal_incomplete", "locale, namespace, and key are required"))
		return ValidatedTranslation{}, rejected
	}
	if len(allowed) > 0 {
		if _, ok := allowed[identity]; !ok {
			rejected.Issues = append(rejected.Issues, proposalIssue(proposal, "proposal_not_in_batch", "proposal is not part of the current translation batch"))
			return ValidatedTranslation{}, rejected
		}
	}

	sourceUnit, ok := sources[sourceIdentity(proposal.Namespace, proposal.Key)]
	if !ok {
		rejected.Issues = append(rejected.Issues, proposalIssue(proposal, "source_missing", "source key no longer exists"))
		return ValidatedTranslation{}, rejected
	}
	if proposal.SourceValue != "" && proposal.SourceValue != sourceUnit.Value && !in.AllowSourceDrift {
		rejected.Issues = append(rejected.Issues, proposalIssue(proposal, "source_drift", "proposal source value does not match the current source value"))
		return ValidatedTranslation{}, rejected
	}

	result := s.validator.ValidatePair(validate.Pair{
		SourceLocale: sourceLocale,
		Locale:       proposal.Locale,
		Namespace:    proposal.Namespace,
		Key:          proposal.Key,
		Source:       sourceUnit.Value,
		Target:       proposal.Value,
	})
	if !result.OK {
		rejected.Issues = slices.Clone(result.Issues)
		return ValidatedTranslation{}, rejected
	}

	accepted := ValidatedTranslation{
		ProposedTranslation: proposal,
		SourceHash:          state.SourceHash(sourceUnit.Value),
		TargetHash:          state.TargetHash(proposal.Value),
		TargetFilePath:      targetFiles[targetFileIdentity(proposal.Locale, proposal.Namespace)],
		Warnings:            result.Warnings,
	}
	accepted.SourceValue = sourceUnit.Value
	return accepted, RejectedTranslation{}
}

func sourceUnits(inv locale.Inventory) map[string]locale.Unit {
	out := map[string]locale.Unit{}
	for _, unit := range inv.Units {
		if unit.Locale == inv.SourceLocale {
			out[sourceIdentity(unit.Namespace, unit.Key)] = unit
		}
	}
	return out
}

func targetFileMap(inv locale.Inventory) map[string]string {
	out := map[string]string{}
	for _, file := range inv.Files {
		if file.Locale == inv.SourceLocale {
			continue
		}
		out[targetFileIdentity(file.Locale, file.Namespace)] = file.Path
	}
	return out
}

func proposalIssue(proposal ProposedTranslation, code string, message string) validate.Issue {
	return validate.Issue{
		Code:      code,
		Message:   message,
		Severity:  validate.SeverityError,
		Locale:    proposal.Locale,
		Namespace: proposal.Namespace,
		Key:       proposal.Key,
	}
}

func sourceIdentity(namespace string, key string) string {
	return namespace + "\x00" + key
}

func proposalIdentity(localeCode string, namespace string, key string) string {
	return localeCode + "\x00" + namespace + "\x00" + key
}

func targetFileIdentity(localeCode string, namespace string) string {
	return localeCode + "\x00" + namespace
}

func sortValidationOutput(out *ValidationOutput) {
	slices.SortFunc(out.Accepted, func(a, b ValidatedTranslation) int {
		return cmp.Compare(
			proposalIdentity(a.Locale, a.Namespace, a.Key),
			proposalIdentity(b.Locale, b.Namespace, b.Key),
		)
	})
	slices.SortFunc(out.Rejected, func(a, b RejectedTranslation) int {
		return cmp.Compare(
			proposalIdentity(a.Locale, a.Namespace, a.Key),
			proposalIdentity(b.Locale, b.Namespace, b.Key),
		)
	})
}
