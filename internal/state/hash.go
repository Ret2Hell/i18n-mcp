package state

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

const (
	sourceHashVersion = "i18n-mcp-source-v1"
	targetHashVersion = "i18n-mcp-target-v1"
)

func SourceHash(value string) string {
	return textHash(sourceHashVersion, value)
}

func TargetHash(value string) string {
	return textHash(targetHashVersion, value)
}

func textHash(version string, value string) string {
	parts := []string{version, value}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x1e")))
	return "sha256:" + hex.EncodeToString(sum[:])
}
