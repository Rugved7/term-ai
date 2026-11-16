package config

import (
	"strings"
	"testing"
)

func TestAPIKeyEnvVar(t *testing.T) {
	tests := []struct {
		provider string
		expected string
	}{
		{"openai", "OPENAI_API_KEY"},
		{"gemini", "GEMINI_API_KEY"},
		{"anthropic", "ANTHROPIC_API_KEY"},
		{"groq", "GROQ_API_KEY"},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			got, err := APIKeyEnvVar(tt.provider)
			if err != nil {
				t.Fatal(err)
			}

			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestResolveAPIKey(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "test-api-key")

	got, err := ResolveAPIKey("gemini")
	if err != nil {
		t.Fatal(err)
	}

	if got != "test-api-key" {
		t.Errorf("unexpected API key result")
	}
}

func TestResolveAPIKeyRejectsMissingKey(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "")

	_, err := ResolveAPIKey("gemini")
	if err == nil {
		t.Fatal("expected missing API key error")
	}

	if !strings.Contains(err.Error(), "GEMINI_API_KEY") {
		t.Errorf("error should identify environment variable: %v", err)
	}
}

func TestResolveAPIKeyRejectsUnsupportedProvider(t *testing.T) {
	_, err := ResolveAPIKey("unknown")

	if err == nil {
		t.Fatal("expected unsupported provider error")
	}
}
