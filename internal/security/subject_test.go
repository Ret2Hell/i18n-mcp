package security

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/stretchr/testify/require"
)

func TestStoreSubjectIsolation(t *testing.T) {
	ctxA := authenticatedContext(t, "subject-a")
	ctxB := authenticatedContext(t, "subject-b")
	var store Store[string]
	store.Put(ctxA, "job-1", "result")

	value, ok, err := store.Get(ctxA, "job-1")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "result", value)

	_, ok, err = store.Get(ctxB, "job-1")
	require.EqualError(t, err, "not found")
	require.False(t, ok)
}

func TestStoreLocalSubject(t *testing.T) {
	var store Store[string]
	store.Put(t.Context(), "latest", "report")
	value, ok, err := store.Get(t.Context(), "latest")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "report", value)
	require.Equal(t, LocalSubject, SubjectFromContext(t.Context()))
}

func TestMissingUserIDUsesLocalSubject(t *testing.T) {
	require.Equal(t, LocalSubject, SubjectFromContext(authenticatedContext(t, "")))
}

func authenticatedContext(t *testing.T, userID string) context.Context {
	t.Helper()
	var got context.Context
	verifier := func(context.Context, string, *http.Request) (*auth.TokenInfo, error) {
		return &auth.TokenInfo{UserID: userID, Expiration: time.Now().Add(time.Hour)}, nil
	}
	handler := auth.RequireBearerToken(verifier, nil)(http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
		got = req.Context()
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer token")
	handler.ServeHTTP(httptest.NewRecorder(), req)
	require.NotNil(t, got)
	return got
}
