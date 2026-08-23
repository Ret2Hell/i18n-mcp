package resources

import (
	"net/url"
	"strings"
)

const (
	// LocalesURI is the resource URI for the locale inventory.
	LocalesURI = "i18n://locales"
	// DiffURI is the resource URI for the latest diff analysis.
	DiffURI = "i18n://analysis/diff"
	// UsageURI is the resource URI for the latest usage analysis.
	UsageURI = "i18n://analysis/usage"
	// DeadKeysURI is the resource URI for the latest dead-key analysis.
	DeadKeysURI = "i18n://analysis/dead-keys"
	// LatestPlanURI is the resource URI for the latest translation plan.
	LatestPlanURI = "i18n://translation/plan/latest"
	// LatestReportURI is the resource URI for the latest audit report.
	LatestReportURI = "i18n://reports/latest"
)

// LocaleURI returns the resource URI for a locale namespace.
func LocaleURI(locale string, namespace string) string {
	return "i18n://locales/" + url.PathEscape(locale) + "/" + url.PathEscape(namespace)
}

// ParseLocaleURI parses an exact locale namespace resource URI.
func ParseLocaleURI(rawURI string) (locale string, namespace string, ok bool) {
	parsed, err := url.Parse(rawURI)
	if err != nil || parsed.Scheme != "i18n" || parsed.Host != "locales" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", "", false
	}
	path, ok := strings.CutPrefix(parsed.EscapedPath(), "/")
	if !ok {
		return "", "", false
	}
	localePart, namespacePart, ok := strings.Cut(path, "/")
	if !ok || strings.Contains(namespacePart, "/") {
		return "", "", false
	}
	locale, err = url.PathUnescape(localePart)
	if err != nil || locale == "" || strings.Contains(locale, "/") {
		return "", "", false
	}
	namespace, err = url.PathUnescape(namespacePart)
	if err != nil || namespace == "" || strings.Contains(namespace, "/") {
		return "", "", false
	}
	return locale, namespace, true
}

// IsSubscribableURI reports whether rawURI identifies a registered i18n resource.
func IsSubscribableURI(rawURI string) bool {
	switch rawURI {
	case LocalesURI, DiffURI, UsageURI, DeadKeysURI, LatestPlanURI, LatestReportURI:
		return true
	default:
		_, _, ok := ParseLocaleURI(rawURI)
		return ok
	}
}
