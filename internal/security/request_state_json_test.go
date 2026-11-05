package security

import (
	"encoding/base64"
	"encoding/json/jsontext"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRequestStatePreservesEmptyRawValues(t *testing.T) {
	signer, err := NewRequestStateSigner([]byte(strings.Repeat("k", 32)))
	require.NoError(t, err)
	for _, data := range []jsontext.Value{nil, jsontext.Value(`{}`), jsontext.Value(`[]`), jsontext.Value(`null`)} {
		claims := RequestStateClaims{
			Subject: "subject", Operation: "op", InputDigest: "input", PlanDigest: "plan",
			ExpiresAt: time.Now().Add(time.Minute).Unix(), Data: data,
		}
		token, err := signer.Sign(claims)
		require.NoError(t, err)
		got, err := signer.Verify(token)
		require.NoError(t, err)
		require.Equal(t, data, got.Data, "empty plan input must not disappear through v2 omitempty semantics")
	}
}

func TestRequestStateRejectsMalformedAuthenticatedPayloads(t *testing.T) {
	signer, err := NewRequestStateSigner([]byte(strings.Repeat("k", 32)))
	require.NoError(t, err)
	signer.now = func() time.Time { return time.Unix(1_800_000_000, 0) }
	valid := `{"version":1,"subject":"subject","operation":"op","inputDigest":"input","planDigest":"plan","expiresAt":1800000060}`
	for name, payload := range map[string]string{
		"trailing_value":   valid + ` {}`,
		"trailing_junk":    valid + ` junk`,
		"duplicate":        strings.Replace(valid, `"subject":"subject"`, `"subject":"subject","subject":"other"`, 1),
		"nested_duplicate": strings.TrimSuffix(valid, "}") + `,"data":{"a":1,"a":2}}`,
		"unknown":          strings.TrimSuffix(valid, "}") + `,"typo":true}`,
		"wrong_case":       strings.Replace(valid, `"version"`, `"Version"`, 1),
		"invalid_utf8":     strings.Replace(valid, `"operation":"op"`, "\"operation\":\"\xff\"", 1),
	} {
		t.Run(name, func(t *testing.T) {
			// A valid MAC must not bypass strict parsing of the complete payload.
			token := base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + base64.RawURLEncoding.EncodeToString(signer.signature([]byte(payload)))
			_, err := signer.Verify(token)
			require.ErrorContains(t, err, "payload is invalid")
		})
	}
}
