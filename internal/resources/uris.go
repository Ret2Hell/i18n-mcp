package resources

import "net/url"

// LocaleURI returns the resource URI for a locale namespace.
func LocaleURI(locale string, namespace string) string {
	return "i18n://locales/" + url.PathEscape(locale) + "/" + url.PathEscape(namespace)
}

const DiffURI = "i18n://analysis/diff"
const UsageURI = "i18n://analysis/usage"
const DeadKeysURI = "i18n://analysis/dead-keys"
const LatestReportURI = "i18n://reports/latest"
