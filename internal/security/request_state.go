package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	requestStateVersion = 1
	maxRequestStateSize = 64 << 10
)

// RequestStateClaims binds an MCP multi-round-trip retry to the operation that requested input.
type RequestStateClaims struct {
	Version     int             `json:"version"`
	Subject     string          `json:"subject"`
	Operation   string          `json:"operation"`
	InputDigest string          `json:"inputDigest"`
	PlanDigest  string          `json:"planDigest"`
	ExpiresAt   int64           `json:"expiresAt"`
	Data        json.RawMessage `json:"data,omitempty"`
}

// RequestStateSigner signs and verifies opaque MCP multi-round-trip request state.
type RequestStateSigner struct {
	key []byte
	now func() time.Time
}

// NewRequestStateSigner constructs a signer from key. The key must contain at least 32 bytes.
func NewRequestStateSigner(key []byte) (*RequestStateSigner, error) {
	if len(key) < sha256.Size {
		return nil, fmt.Errorf("request state signing key must be at least %d bytes", sha256.Size)
	}
	return &RequestStateSigner{key: append([]byte(nil), key...), now: time.Now}, nil
}

// NewRandomRequestStateSigner constructs a signer with a process-local random key.
func NewRandomRequestStateSigner() (*RequestStateSigner, error) {
	key := make([]byte, sha256.Size)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generate request state signing key: %w", err)
	}
	return NewRequestStateSigner(key)
}

// Sign returns integrity-protected opaque request state.
func (s *RequestStateSigner) Sign(claims RequestStateClaims) (string, error) {
	if s == nil {
		return "", errors.New("request state signer is not configured")
	}
	claims.Version = requestStateVersion
	if err := validateRequestStateClaims(claims, s.now()); err != nil {
		return "", err
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshal request state: %w", err)
	}
	signature := s.signature(payload)
	token := base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(signature)
	if len(token) > maxRequestStateSize {
		return "", errors.New("request state exceeds size limit")
	}
	return token, nil
}

// Verify authenticates and decodes opaque request state.
func (s *RequestStateSigner) Verify(token string) (RequestStateClaims, error) {
	if s == nil {
		return RequestStateClaims{}, errors.New("request state signer is not configured")
	}
	if token == "" || len(token) > maxRequestStateSize {
		return RequestStateClaims{}, errors.New("request state is invalid")
	}
	payloadPart, signaturePart, ok := strings.Cut(token, ".")
	if !ok || strings.Contains(signaturePart, ".") {
		return RequestStateClaims{}, errors.New("request state is invalid")
	}
	payload, err := base64.RawURLEncoding.DecodeString(payloadPart)
	if err != nil {
		return RequestStateClaims{}, errors.New("request state is invalid")
	}
	signature, err := base64.RawURLEncoding.DecodeString(signaturePart)
	if err != nil || !hmac.Equal(signature, s.signature(payload)) {
		return RequestStateClaims{}, errors.New("request state signature is invalid")
	}
	var claims RequestStateClaims
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&claims); err != nil {
		return RequestStateClaims{}, errors.New("request state payload is invalid")
	}
	if err := validateRequestStateClaims(claims, s.now()); err != nil {
		return RequestStateClaims{}, err
	}
	return claims, nil
}

func (s *RequestStateSigner) signature(payload []byte) []byte {
	mac := hmac.New(sha256.New, s.key)
	_, _ = mac.Write(payload)
	return mac.Sum(nil)
}

func validateRequestStateClaims(claims RequestStateClaims, now time.Time) error {
	if claims.Version != requestStateVersion {
		return errors.New("request state version is invalid")
	}
	if claims.Subject == "" || claims.Operation == "" || claims.InputDigest == "" || claims.PlanDigest == "" {
		return errors.New("request state claims are incomplete")
	}
	if claims.ExpiresAt <= now.Unix() {
		return errors.New("request state has expired")
	}
	return nil
}
