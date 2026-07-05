package scanner

import "strings"

// FullKey combines namespace and key into a display key.
func FullKey(namespace string, key string) string {
	if namespace == "" {
		return key
	}
	if key == "" {
		return namespace
	}
	return namespace + "." + key
}

// UsageIdentity returns a stable identity for a namespace and key.
func UsageIdentity(namespace string, key string) string {
	return namespace + "\x00" + key
}

// CleanLiteralKey normalizes a scanned literal translation key.
func CleanLiteralKey(key string) string {
	return strings.TrimSpace(key)
}
