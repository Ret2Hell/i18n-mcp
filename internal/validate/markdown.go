package validate

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
)

type MarkdownLink struct {
	Raw         string `json:"raw"`
	Text        string `json:"text"`
	Destination string `json:"destination"`
}

var markdownLinkPattern = regexp.MustCompile(`\[([^\]\n]+)\]\(([^)\n]+)\)`)

func ExtractMarkdownLinks(s string) []MarkdownLink {
	matches := markdownLinkPattern.FindAllStringSubmatch(s, -1)
	links := make([]MarkdownLink, 0, len(matches))
	for _, match := range matches {
		if len(match) != 3 {
			continue
		}
		links = append(links, MarkdownLink{Raw: match[0], Text: match[1], Destination: match[2]})
	}
	return links
}

func compareMarkdownLinks(source string, target string) []Issue {
	sourceLinks := ExtractMarkdownLinks(source)
	targetLinks := ExtractMarkdownLinks(target)
	var issues []Issue
	if len(sourceLinks) != len(targetLinks) {
		issues = append(issues, Issue{
			Code:     "markdown_link_count_changed",
			Message:  fmt.Sprintf("target changes Markdown link count from %d to %d", len(sourceLinks), len(targetLinks)),
			Severity: SeverityWarning,
		})
	}

	sourceDestinations := markdownDestinationCounts(sourceLinks)
	targetDestinations := markdownDestinationCounts(targetLinks)
	issues = append(issues, compareTokenCounts(
		sourceDestinations,
		targetDestinations,
		"Markdown link destination",
		"markdown_link_destination_missing",
		"markdown_link_destination_extra",
		"markdown_link_destination_count_changed",
		SeverityWarning,
	)...)
	return issues
}

func markdownDestinationCounts(links []MarkdownLink) map[string]int {
	counts := make(map[string]int, len(links))
	for _, link := range links {
		counts[link.Destination]++
	}
	return counts
}

func MarkdownLinkDestinations(s string) []string {
	links := ExtractMarkdownLinks(s)
	seen := make(map[string]struct{}, len(links))
	for _, link := range links {
		seen[link.Destination] = struct{}{}
	}
	return slices.Sorted(maps.Keys(seen))
}
