package validate

import (
	"fmt"
	"regexp"
	"sort"
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
	counts := map[string]int{}
	for _, link := range links {
		counts[link.Destination]++
	}
	return counts
}

func MarkdownLinkDestinations(s string) []string {
	seen := map[string]bool{}
	var out []string
	for _, link := range ExtractMarkdownLinks(s) {
		if seen[link.Destination] {
			continue
		}
		seen[link.Destination] = true
		out = append(out, link.Destination)
	}
	sort.Strings(out)
	return out
}
