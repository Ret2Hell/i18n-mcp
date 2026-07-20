// Package security provides authenticated subject isolation helpers.
package security

import (
	"context"
	"errors"

	"github.com/modelcontextprotocol/go-sdk/auth"
)

// LocalSubject is the shared subject for unauthenticated local transports.
const LocalSubject = "local-stdio"

// SubjectFromContext returns the authenticated subject or the local transport subject.
func SubjectFromContext(ctx context.Context) string {
	ti := auth.TokenInfoFromContext(ctx)
	if ti == nil || ti.UserID == "" {
		return LocalSubject
	}
	return ti.UserID
}

// RequireSubject verifies ownership without revealing cross-subject objects.
func RequireSubject(ctx context.Context, owner string) error {
	if owner == "" {
		return errors.New("owned object has no subject")
	}
	if SubjectFromContext(ctx) != owner {
		return errors.New("not found")
	}
	return nil
}
