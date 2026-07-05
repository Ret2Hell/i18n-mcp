package resources

import "net/url"

// LocaleURI returns the resource URI for a locale namespace.
func LocaleURI(locale string, namespace string) string {
	return "i18n://locales/" + url.PathEscape(locale) + "/" + url.PathEscape(namespace)
}

const (
	// DiffURI is the resource URI for the latest diff analysis.
	DiffURI = "i18n://analysis/diff"
	// UsageURI is the resource URI for the latest usage analysis.
	UsageURI = "i18n://analysis/usage"
	// DeadKeysURI is the resource URI for the latest dead-key analysis.
	DeadKeysURI = "i18n://analysis/dead-keys"
	// LatestReportURI is the resource URI for the latest audit report.
	LatestReportURI = "i18n://reports/latest"
)
