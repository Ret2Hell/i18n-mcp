package scanner

import "strings"

func FullKey(namespace string, key string) string {
	if namespace == "" {
		return key
	}
	if key == "" {
		return namespace
	}
	return namespace + "." + key
}

func UsageIdentity(namespace string, key string) string {
	return namespace + "\x00" + key
}

func CleanLiteralKey(key string) string {
	return strings.TrimSpace(key)
}
