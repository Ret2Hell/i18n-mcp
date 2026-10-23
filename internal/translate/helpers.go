package translate

import (
	"cmp"
	"fmt"
	json "github.com/Ret2Hell/i18n-mcp/internal/jsonutil"
	"io"
	"os"
	"slices"
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

func batchDigest(projectRoot string, batch Batch) (string, error) {
	payload := batchIDPayload{
		ProjectRoot:   projectRoot,
		SourceLocale:  batch.SourceLocale,
		TargetLocales: slices.Clone(batch.TargetLocales),
		Items:         slices.Clone(batch.Items),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sum := state.SourceHash(string(data))
	hash, _ := strings.CutPrefix(sum, "sha256:")
	return hash, nil
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

func (s *Service) loadPlanContext(cfg config.Resolved) (string, string, []string, []ContextFileRef, []string) {
	var styleGuide string
	var glossaryText string
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
		text, err := s.readOptionalText(cfg.Translation.GlossaryPath)
		if err != nil {
			warnings = append(warnings, err.Error())
		} else {
			glossaryText = text
			glossaryRefs = append(glossaryRefs, cfg.Translation.GlossaryPath)
			contextFiles = append(contextFiles, ContextFileRef{Kind: "glossary", Path: cfg.Translation.GlossaryPath})
		}
	}
	slices.Sort(glossaryRefs)
	slices.SortFunc(contextFiles, func(a, b ContextFileRef) int {
		return cmp.Or(
			cmp.Compare(a.Kind, b.Kind),
			cmp.Compare(a.Path, b.Path),
		)
	})
	slices.Sort(warnings)
	return styleGuide, glossaryText, glossaryRefs, contextFiles, warnings
}

func (s *Service) readOptionalText(relPath string) (string, error) {
	resolved, err := s.guard.ResolveExisting(relPath)
	if err != nil {
		return "", err
	}
	file, err := os.Open(resolved)
	if err != nil {
		return "", err
	}
	defer file.Close()
	const maxContextBytes = 16 << 10
	data, err := io.ReadAll(io.LimitReader(file, maxContextBytes+1))
	if err != nil {
		return "", err
	}
	if len(data) > maxContextBytes {
		return "", fmt.Errorf("translation context %s exceeds %d bytes; shorten the file", relPath, maxContextBytes)
	}
	return string(data), nil
}
