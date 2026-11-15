package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg, err := Default()
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Provider != "ollama" {
		t.Errorf("provider = %q, want ollama", cfg.Provider)
	}

	if cfg.Model != "llama3.2:3b" {
		t.Errorf("model = %q, want llama3.2:3b", cfg.Model)
	}

	if cfg.Ollama.BaseURL != "http://localhost:11434" {
		t.Errorf("unexpected Ollama URL: %q", cfg.Ollama.BaseURL)
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("default configuration is invalid: %v", err)
	}
}

func TestLoadMissingConfigUsesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.yaml")

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Model != "llama3.2:3b" {
		t.Errorf("model = %q, want default model", cfg.Model)
	}
}

func TestCreateAndLoadConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "termai", "config.yaml")

	if err := CreateDefault(path); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Provider != "ollama" {
		t.Errorf("provider = %q, want ollama", cfg.Provider)
	}

	// Creating again must not overwrite the existing configuration.
	if err := CreateDefault(path); err == nil {
		t.Fatal("expected error when configuration already exists")
	}
}

func TestLoadCustomConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")

	content := []byte(`
provider: ollama
model: llama3.2:3b
data_dir: /tmp/termai-test
ollama:
  base_url: http://127.0.0.1:11434
`)

	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.DataDir != "/tmp/termai-test" {
		t.Errorf("data directory = %q", cfg.DataDir)
	}

	if cfg.Ollama.BaseURL != "http://127.0.0.1:11434" {
		t.Errorf("Ollama URL = %q", cfg.Ollama.BaseURL)
	}
}

func TestValidateRejectsInvalidProvider(t *testing.T) {
	cfg, err := Default()
	if err != nil {
		t.Fatal(err)
	}

	cfg.Provider = "unknown"

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected unsupported provider error")
	}
}
