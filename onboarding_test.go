package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Rugved7/termai/internal/config"
)

func TestRunOnboardingOllama(t *testing.T) {
	path := t.TempDir() + "/config.yaml"

	var output bytes.Buffer

	err := runOnboarding(
		path,
		strings.NewReader("1\n"),
		&output,
	)
	if err != nil {
		t.Fatalf("runOnboarding() error = %v", err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}

	if cfg.Provider != "ollama" {
		t.Errorf("provider = %q, want ollama", cfg.Provider)
	}

	if cfg.Model != "llama3.2:3b" {
		t.Errorf("model = %q, want llama3.2:3b", cfg.Model)
	}
}

func TestRunOnboardingGemini(t *testing.T) {
	path := t.TempDir() + "/config.yaml"
	t.Setenv("GEMINI_API_KEY", "")

	var output bytes.Buffer

	// Skip API key entry by just pressing enter (empty key)
	err := runOnboarding(
		path,
		strings.NewReader("2\n\n"),
		&output,
	)
	if err != nil {
		t.Fatalf("runOnboarding() error = %v", err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}

	if cfg.Provider != "gemini" {
		t.Errorf("provider = %q, want gemini", cfg.Provider)
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "aistudio.google.com") {
		t.Error("expected Gemini API key instructions in output")
	}

	// Should mention auth login command when key is skipped
	if !strings.Contains(outputStr, "auth login") {
		t.Error("expected auth login suggestion in output")
	}
}

func TestRunOnboardingInvalidSelection(t *testing.T) {
	path := t.TempDir() + "/config.yaml"

	var output bytes.Buffer

	err := runOnboarding(
		path,
		strings.NewReader("3\n"),
		&output,
	)
	if err == nil {
		t.Fatal("expected error for invalid selection")
	}
}

func TestRunOnboardingEmptyInput(t *testing.T) {
	path := t.TempDir() + "/config.yaml"

	var output bytes.Buffer

	err := runOnboarding(
		path,
		strings.NewReader(""),
		&output,
	)
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}
