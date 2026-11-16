package config

import (
	"os"
	"path/filepath"
	"strings"
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

func TestLoadRejectsUnknownTopLevelField(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")

	data := []byte(`
provider: ollama
modle: llama3.2:3b
data_dir: /tmp/termai
ollama:
  base_url: http://localhost:11434
`)

	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := Load(path)

	if err == nil {
		t.Fatal("expected unknown field error")
	}

	if !strings.Contains(err.Error(), "modle") {
		t.Errorf("error should mention unknown field: %v", err)
	}
}

func TestLoadRejectsUnknownNestedField(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")

	data := []byte(`
provider: ollama
model: llama3.2:3b
data_dir: /tmp/termai
ollama:
  base_url: http://localhost:11434
  base_urll: http://localhost:11434
`)

	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := Load(path)

	if err == nil {
		t.Fatal("expected unknown nested field error")
	}

	if !strings.Contains(err.Error(), "base_urll") {
		t.Errorf("error should mention unknown field: %v", err)
	}
}

func TestLoadRejectsEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")

	if err := os.WriteFile(path, []byte("   \n"), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := Load(path)

	if err == nil {
		t.Fatal("expected empty configuration error")
	}

	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoadRejectsMultipleYAMLDocuments(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")

	data := []byte(`
provider: ollama
model: llama3.2:3b
data_dir: /tmp/termai
ollama:
  base_url: http://localhost:11434
---
provider: ollama
model: another-model
`)

	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := Load(path)

	if err == nil {
		t.Fatal("expected multiple YAML documents error")
	}

	if !strings.Contains(err.Error(), "exactly one YAML document") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoadRejectsDuplicateYAMLKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")

	data := []byte(`
provider: ollama
model: llama3.2:3b
model: codellama:7b
data_dir: /tmp/termai
ollama:
  base_url: http://localhost:11434
`)

	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := Load(path)

	if err == nil {
		t.Fatal("expected duplicate YAML key error")
	}
}

func TestValidateAcceptsSupportedProviders(t *testing.T) {
	providers := []string{
		"ollama",
		"gemini",
	}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			cfg, err := Default()
			if err != nil {
				t.Fatal(err)
			}

			cfg.Provider = provider

			if err := cfg.Validate(); err != nil {
				t.Fatalf(
					"expected provider %q to be valid: %v",
					provider,
					err,
				)
			}
		})
	}
}

func TestValidateCloudProviderDoesNotRequireOllamaURL(t *testing.T) {
	cfg, err := Default()
	if err != nil {
		t.Fatal(err)
	}

	cfg.Provider = "gemini"
	cfg.Ollama.BaseURL = ""

	if err := cfg.Validate(); err != nil {
		t.Fatalf("cloud configuration should not require Ollama URL: %v", err)
	}
}

func TestValidateRejectsEmptyModel(t *testing.T) {
	cfg, err := Default()
	if err != nil {
		t.Fatal(err)
	}

	cfg.Model = " "

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected empty model error")
	}
}

func TestValidateRejectsUnimplementedProviders(t *testing.T) {
	providers := []string{
		"openai",
		"anthropic",
		"groq",
		"unknown",
	}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			cfg, err := Default()
			if err != nil {
				t.Fatal(err)
			}

			cfg.Provider = provider

			if err := cfg.Validate(); err == nil {
				t.Fatalf(
					"expected provider %q to be rejected",
					provider,
				)
			}
		})
	}
}
