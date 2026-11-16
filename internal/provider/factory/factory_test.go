package factory

import (
	"testing"

	"github.com/Rugved7/termai/internal/config"
	"github.com/Rugved7/termai/internal/provider/gemini"
	"github.com/Rugved7/termai/internal/provider/ollama"
)

func TestNewOllamaProvider(t *testing.T) {
	cfg, err := config.Default()
	if err != nil {
		t.Fatal(err)
	}

	provider, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	if provider == nil {
		t.Fatal("New() returned nil provider")
	}

	if _, ok := provider.(*ollama.Client); !ok {
		t.Fatalf("expected *ollama.Client, got %T", provider)
	}
}

func TestNewUnsupportedProvider(t *testing.T) {
	cfg, err := config.Default()
	if err != nil {
		t.Fatal(err)
	}

	cfg.Provider = "unsupported"

	provider, err := New(cfg)

	if err == nil {
		t.Fatal("expected error for unsupported provider")
	}

	if provider != nil {
		t.Fatal("expected nil provider for unsupported provider")
	}
}

func TestNewGeminiProvider(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "test-api-key")

	cfg, err := config.Default()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Provider = "gemini"
	cfg.Model = "gemini-2.5-flash"

	provider, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := provider.(*gemini.Client); !ok {
		t.Fatalf("expected *gemini.Client, got %T", provider)
	}
}

func TestNewGeminiProviderWithoutAPIKey(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "")

	cfg, err := config.Default()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Provider = "gemini"

	provider, err := New(cfg)
	if err == nil {
		t.Fatal("expected missing API key error")
	}

	if provider != nil {
		t.Fatal("expected nil provider on initialization failure")
	}
}
