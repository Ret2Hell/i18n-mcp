package security

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRequestStateSignerRoundTrip(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	signer, err := NewRequestStateSigner([]byte(strings.Repeat("k", 32)))
	require.NoError(t, err)
	signer.now = func() time.Time { return now }
	want := RequestStateClaims{
		Subject:     "subject-a",
		Operation:   "i18n.keys.prune",
		InputDigest: "input",
		PlanDigest:  "plan",
		ExpiresAt:   now.Add(time.Minute).Unix(),
	}

	token, err := signer.Sign(want)
	require.NoError(t, err)
	got, err := signer.Verify(token)
	require.NoError(t, err)
	want.Version = requestStateVersion
	require.Equal(t, want, got)
}

func TestRequestStateSignerRejectsInvalidState(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	signer, err := NewRequestStateSigner([]byte(strings.Repeat("k", 32)))
	require.NoError(t, err)
	signer.now = func() time.Time { return now }
	valid, err := signer.Sign(RequestStateClaims{
		Subject:     "subject-a",
		Operation:   "i18n.keys.prune",
		InputDigest: "input",
		PlanDigest:  "plan",
		ExpiresAt:   now.Add(time.Minute).Unix(),
	})
	require.NoError(t, err)

	tests := []struct {
		name        string
		token       string
		advance     time.Duration
		errorString string
	}{
		{name: "empty", errorString: "invalid"},
		{name: "malformed", token: "not-a-token", errorString: "invalid"},
		{name: "modified", token: "A" + valid[1:], errorString: "signature"},
		{name: "expired", token: valid, advance: 2 * time.Minute, errorString: "expired"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signer.now = func() time.Time { return now.Add(tt.advance) }
			_, err := signer.Verify(tt.token)
			require.ErrorContains(t, err, tt.errorString)
		})
	}
}

func TestRequestStateSignerRequiresStrongKey(t *testing.T) {
	_, err := NewRequestStateSigner([]byte("short"))
	require.ErrorContains(t, err, "at least 32 bytes")
}
