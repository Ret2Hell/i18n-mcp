package translate

import (
	"encoding/json"
	"os"
	"sort"
	"strings"

	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/Ret2Hell/i18n-mcp/internal/state"
)

type batchIDPayload struct {
	ProjectRoot   string   `json:"projectRoot"`
	SourceLocale  string   `json:"sourceLocale"`
	TargetLocales []string `json:"targetLocales"`
	Items         []Item   `json:"items"`
}

func buildBatchID(projectRoot string, batch Batch) string {
	payload := batchIDPayload{
		ProjectRoot:   projectRoot,
		SourceLocale:  batch.SourceLocale,
		TargetLocales: append([]string(nil), batch.TargetLocales...),
		Items:         append([]Item(nil), batch.Items...),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	sum := state.SourceHash(string(data))
	return "batch_" + strings.TrimPrefix(sum, "sha256:")[:16]
}

func defaultValidationRules() []string {
	return []string{
		"preserve placeholders exactly",
		"preserve HTML-like tag structure",
		"preserve ICU argument names",
		"do not return empty target text for non-empty source text",
		"call i18n.translation.validate before applying translations",
	}
}

func (s *Service) loadPlanContext(cfg config.Resolved) (string, []string, []ContextFileRef, []string) {
	var styleGuide string
	var glossaryRefs []string
	var contextFiles []ContextFileRef
	var warnings []string

	if cfg.Translation.StyleGuidePath != "" {
		text, err := s.readOptionalText(cfg.Translation.StyleGuidePath)
		if err != nil {
			warnings = append(warnings, err.Error())
		} else {
			styleGuide = text
			contextFiles = append(contextFiles, ContextFileRef{Kind: "styleGuide", Path: cfg.Translation.StyleGuidePath})
		}
	}
	if cfg.Translation.GlossaryPath != "" {
		glossaryRefs = append(glossaryRefs, cfg.Translation.GlossaryPath)
		contextFiles = append(contextFiles, ContextFileRef{Kind: "glossary", Path: cfg.Translation.GlossaryPath})
	}
	sort.Strings(glossaryRefs)
	sort.Slice(contextFiles, func(i, j int) bool {
		if contextFiles[i].Kind != contextFiles[j].Kind {
			return contextFiles[i].Kind < contextFiles[j].Kind
		}
		return contextFiles[i].Path < contextFiles[j].Path
	})
	sort.Strings(warnings)
	return styleGuide, glossaryRefs, contextFiles, warnings
}

func (s *Service) readOptionalText(relPath string) (string, error) {
	resolved, err := s.guard.ResolveExisting(relPath)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func stringSet(values []string) map[string]bool {
	set := map[string]bool{}
	for _, value := range values {
		if value != "" {
			set[value] = true
		}
	}
	return set
}
