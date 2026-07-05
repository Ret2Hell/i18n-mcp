package openai

import "testing"

func TestLoadCredentialsRequiresAPIKey(t *testing.T) {
	_, err := LoadCredentials(func(key string) (string, bool) {
		values := map[string]string{"I18N_MCP_OPENAI_MODEL": "gpt-4o-mini"}
		value, ok := values[key]
		return value, ok
	})
	if err == nil {
		t.Fatal("expected missing API key error")
	}
}

func TestLoadCredentialsRequiresModel(t *testing.T) {
	_, err := LoadCredentials(func(key string) (string, bool) {
		values := map[string]string{"I18N_MCP_OPENAI_API_KEY": "secret"}
		value, ok := values[key]
		return value, ok
	})
	if err == nil {
		t.Fatal("expected missing model error")
	}
}

func TestLoadCredentialsDefaultsBaseURL(t *testing.T) {
	creds, err := LoadCredentials(func(key string) (string, bool) {
		values := map[string]string{
			"I18N_MCP_OPENAI_API_KEY": "secret",
			"I18N_MCP_OPENAI_MODEL":   "gpt-4o-mini",
		}
		value, ok := values[key]
		return value, ok
	})
	if err != nil {
		t.Fatalf("LoadCredentials() error = %v", err)
	}
	if creds.BaseURL != "https://api.openai.com/v1" {
		t.Fatalf("BaseURL = %q", creds.BaseURL)
	}
}

func TestRedactNeverReturnsSecret(t *testing.T) {
	secret := "sk-test-secret"
	if got := Redact(secret); got == secret {
		t.Fatal("Redact returned original secret")
	}
	if got := Redact(""); got != "" {
		t.Fatalf("Redact(empty) = %q", got)
	}
}
