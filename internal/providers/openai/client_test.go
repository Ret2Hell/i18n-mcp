package openai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Ret2Hell/i18n-mcp/internal/translate"
)

func TestGenerateReturnsProposalsAndUsage(t *testing.T) {
	const secret = "sk-test-secret"
	var gotAuth string
	var gotReq chatRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotReq); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices":[{"message":{"role":"assistant","content":"[{\"locale\":\"fr\",\"namespace\":\"common\",\"key\":\"hello\",\"sourceValue\":\"Hello {name}\",\"value\":\"Bonjour {name}\"}]"}}],
			"usage":{"prompt_tokens":11,"completion_tokens":7,"total_tokens":18}
		}`))
	}))
	defer server.Close()

	client := NewClient(Options{Credentials: new(Credentials{APIKey: secret, BaseURL: server.URL + "/v1/", Model: "test-model"})})
	res, err := client.Generate(t.Context(), providerRequest())
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if gotAuth != "Bearer "+secret {
		t.Fatal("authorization header was not sent as bearer token")
	}
	if gotReq.Model != "test-model" || len(gotReq.Messages) != 2 {
		t.Fatalf("unexpected chat request: %#v", gotReq)
	}
	prompt := gotReq.Messages[1].Content
	for _, want := range []string{"en", "fr", "Hello {name}", "common", "hello"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q in %q", want, prompt)
		}
	}
	if len(res.Proposals) != 1 || res.Proposals[0].Value != "Bonjour {name}" {
		t.Fatalf("proposals = %#v", res.Proposals)
	}
	if res.Usage.InputTokens != 11 || res.Usage.OutputTokens != 7 || res.Usage.TotalTokens != 18 {
		t.Fatalf("usage = %#v", res.Usage)
	}
}

func TestGenerateNon2xxDoesNotLeakResponseSecrets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream-secret-token", http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewClient(Options{Credentials: new(Credentials{APIKey: "sk-test-secret", BaseURL: server.URL, Model: "test-model"})})
	_, err := client.Generate(t.Context(), providerRequest())
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "upstream-secret-token") || strings.Contains(err.Error(), "sk-test-secret") {
		t.Fatalf("error leaked secret: %v", err)
	}
	if !strings.Contains(err.Error(), "status 401") {
		t.Fatalf("error = %v", err)
	}
}

func TestGenerateContextCancellationStopsHTTPRequest(t *testing.T) {
	started := make(chan struct{})
	serverDone := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		select {
		case <-r.Context().Done():
		case <-serverDone:
		}
	}))
	defer func() {
		close(serverDone)
		server.Close()
	}()

	ctx, cancel := context.WithCancel(t.Context())
	client := NewClient(Options{Credentials: new(Credentials{APIKey: "sk-test-secret", BaseURL: server.URL, Model: "test-model"})})
	errCh := make(chan error, 1)
	go func() {
		_, err := client.Generate(ctx, providerRequest())
		errCh <- err
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("request did not start")
	}
	cancel()
	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Generate() error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("request did not stop after cancellation")
	}
}

func TestGenerateRequiresCredentials(t *testing.T) {
	_, err := NewClient(Options{}).Generate(t.Context(), providerRequest())
	if err == nil {
		t.Fatal("expected missing credentials error")
	}
}

func providerRequest() translate.ProviderRequest {
	return translate.ProviderRequest{
		SourceLocale: "en",
		TargetLocale: "fr",
		Items: []translate.ProviderItem{{
			ID:          "fr:common:hello",
			Locale:      "fr",
			Namespace:   "common",
			Key:         "hello",
			SourceValue: "Hello {name}",
			Status:      "missing",
			SourceHash:  "abc123",
		}},
		StyleGuide: "friendly",
		Glossary:   "Hello=Bonjour",
	}
}
