package translate

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildSamplingPromptRejectsMissingPlan(t *testing.T) {
	prompt, err := buildSamplingPrompt(SamplingRequest{})

	require.Error(t, err)
	require.Empty(t, prompt)
	require.Contains(t, err.Error(), "translation plan")
}

func TestBuildSamplingPromptIncludesPlanContextAndConstraints(t *testing.T) {
	prompt, err := buildSamplingPrompt(SamplingRequest{
		Plan: &Batch{
			BatchID:       "batch-123",
			SourceLocale:  "en",
			TargetLocales: []string{"fr", "de"},
			Items: []Item{
				{
					Locale:       "fr",
					Namespace:    "common",
					Key:          "welcome",
					SourceValue:  "Hello {name}\n<strong>friend</strong>",
					OldValue:     "Salut",
					Placeholders: []string{"{name}"},
					Tags:         []string{"strong"},
				},
			},
		},
		StyleGuide: "Use concise product copy.",
		Glossary:   "friend = ami",
	})

	require.NoError(t, err)
	require.Contains(t, prompt, "Source locale: en")
	require.Contains(t, prompt, "Target locales: fr, de")
	require.Contains(t, prompt, "Batch ID: batch-123")
	require.Contains(t, prompt, "Target locale: fr")
	require.Contains(t, prompt, "Namespace: common")
	require.Contains(t, prompt, "Key: welcome")
	require.Contains(t, prompt, `Current source value: "Hello {name}\n<strong>friend</strong>"`)
	require.Contains(t, prompt, `Previous target value: "Salut"`)
	require.Contains(t, prompt, "Placeholders to preserve: {name}")
	require.Contains(t, prompt, "HTML-like tags to preserve: strong")
	require.Contains(t, prompt, "Style guide:\nUse concise product copy.")
	require.Contains(t, prompt, "Glossary:\nfriend = ami")
	require.Contains(t, prompt, "Return JSON only")
	require.Contains(t, prompt, "Exact JSON response schema")
	require.False(t, strings.Contains(prompt, "```"), "prompt should not invite fenced output")
}

func TestBuildSamplingPromptOmitsBlankOptionalContext(t *testing.T) {
	prompt, err := buildSamplingPrompt(SamplingRequest{
		Plan: &Batch{
			BatchID:       "batch-123",
			SourceLocale:  "en",
			TargetLocales: []string{"fr"},
			Items: []Item{{
				Locale:      "fr",
				Namespace:   "common",
				Key:         "title",
				SourceValue: "Title",
			}},
		},
		StyleGuide: "  \n\t",
		Glossary:   "",
	})

	require.NoError(t, err)
	require.NotContains(t, prompt, "Style guide:")
	require.NotContains(t, prompt, "Glossary:")
	require.NotContains(t, prompt, "Previous target value:")
	require.NotContains(t, prompt, "Placeholders to preserve:")
	require.NotContains(t, prompt, "HTML-like tags to preserve:")
}
