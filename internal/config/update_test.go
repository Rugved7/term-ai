package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetProvider(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")

	if err := CreateDefault(path); err != nil {
		t.Fatal(err)
	}

	if err := Set(path, "provider", "gemini"); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Provider != "gemini" {
		t.Errorf("provider = %q, want gemini", cfg.Provider)
	}

	if cfg.Model != "llama3.2:3b" {
		t.Errorf("model changed unexpectedly: %q", cfg.Model)
	}

	if cfg.Ollama.BaseURL != "http://localhost:11434" {
		t.Errorf("Ollama URL changed unexpectedly")
	}
}

func TestSetModel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")

	if err := CreateDefault(path); err != nil {
		t.Fatal(err)
	}

	if err := Set(path, "model", "gemini-2.5-flash"); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Model != "gemini-2.5-flash" {
		t.Errorf("model = %q, want gemini-2.5-flash", cfg.Model)
	}
}

func TestSetOllamaBaseURL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")

	if err := CreateDefault(path); err != nil {
		t.Fatal(err)
	}

	want := "http://127.0.0.1:11434"

	if err := Set(path, "ollama.base_url", want); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Ollama.BaseURL != want {
		t.Errorf("Ollama URL = %q, want %q", cfg.Ollama.BaseURL, want)
	}
}

func TestSetCreatesMissingConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "termai", "config.yaml")

	if err := Set(path, "provider", "gemini"); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Provider != "gemini" {
		t.Errorf("provider = %q, want gemini", cfg.Provider)
	}
}

func TestSetRejectsUnsupportedKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")

	if err := CreateDefault(path); err != nil {
		t.Fatal(err)
	}

	err := Set(path, "api_key", "secret")

	if err == nil {
		t.Fatal("expected unsupported key error")
	}
}

func TestSetRejectsInvalidProvider(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")

	if err := CreateDefault(path); err != nil {
		t.Fatal(err)
	}

	err := Set(path, "provider", "openai")

	if err == nil {
		t.Fatal("expected invalid provider error")
	}

	cfg, loadErr := Load(path)
	if loadErr != nil {
		t.Fatal(loadErr)
	}

	if cfg.Provider != "ollama" {
		t.Errorf("configuration changed after failed update: %q", cfg.Provider)
	}
}

func TestSetRejectsInvalidOllamaURL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")

	if err := CreateDefault(path); err != nil {
		t.Fatal(err)
	}

	err := Set(path, "ollama.base_url", "invalid-url")

	if err == nil {
		t.Fatal("expected invalid URL error")
	}
}

func TestSetRejectsEmptyValue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")

	err := Set(path, "model", " ")

	if err == nil {
		t.Fatal("expected empty value error")
	}
}

func TestSetPreservesFilePermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")

	if err := CreateDefault(path); err != nil {
		t.Fatal(err)
	}

	if err := Set(path, "model", "llama3.2:3b"); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	if info.Mode().Perm() != 0600 {
		t.Errorf("permissions = %o, want 600", info.Mode().Perm())
	}
}

func TestSetDoesNotModifyFileOnInvalidUpdate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")

	if err := CreateDefault(path); err != nil {
		t.Fatal(err)
	}

	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	err = Set(path, "provider", "unsupported")
	if err == nil {
		t.Fatal("expected validation error")
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if strings.TrimSpace(string(before)) != strings.TrimSpace(string(after)) {
		t.Fatal("configuration file changed after failed update")
	}
}
