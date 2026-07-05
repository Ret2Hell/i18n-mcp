package state

import (
	"crypto/sha256"
	"encoding/hex"
)

const (
	sourceHashVersion = "i18n-mcp-source-v1"
	targetHashVersion = "i18n-mcp-target-v1"
)

// SourceHash returns a stable hash for source text.
func SourceHash(value string) string {
	return textHash(sourceHashVersion, value)
}

// TargetHash returns a stable hash for translated target text.
func TargetHash(value string) string {
	return textHash(targetHashVersion, value)
}

func textHash(version string, value string) string {
	sum := sha256.Sum256([]byte(version + "\x1e" + value))
	return "sha256:" + hex.EncodeToString(sum[:])
}
