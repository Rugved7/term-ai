package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/Rugved7/termai/internal/config"
	"github.com/Rugved7/termai/internal/provider/gemini"
	"github.com/Rugved7/termai/internal/secrets"
	"golang.org/x/term"
)

func runOnboarding(path string, input io.Reader, output io.Writer) error {
	reader := bufio.NewReader(input)

	fmt.Fprintln(output, "\nWelcome to TermAI!")
	fmt.Fprintln(output, "\nChoose your inference provider:")
	fmt.Fprintln(output, "  1. Local  — Ollama (runs on your machine)")
	fmt.Fprintln(output, "  2. Cloud  — Gemini (use your own API key)")
	fmt.Fprint(output, "\nSelect [1/2]: ")

	choice, err := reader.ReadString('\n')
	if err != nil && len(choice) == 0 {
		return fmt.Errorf("read provider selection: %w", err)
	}

	var provider, model string

	switch strings.TrimSpace(choice) {
	case "1":
		provider = "ollama"
		model = "llama3.2:3b"
	case "2":
		provider = "gemini"
		model = "gemini-2.0-flash-exp"
	default:
		return fmt.Errorf("invalid selection %q: choose 1 or 2", strings.TrimSpace(choice))
	}

	if err := config.Initialize(path, provider, model); err != nil {
		return fmt.Errorf("initialize configuration: %w", err)
	}

	fmt.Fprintf(output, "\nTermAI configured to use %s.\n", provider)

	if provider == "ollama" {
		fmt.Fprintln(output, "Make sure Ollama is installed and running.")
		fmt.Fprintln(output, "Pull the default model with: ollama pull "+model)
	} else {
		// Gemini: walk through API key setup
		if err := setupGeminiKey(path, output); err != nil {
			return err
		}
	}

	fmt.Fprintln(output, "\nConfiguration saved:", path)

	return nil
}

func setupGeminiKey(configPath string, output io.Writer) error {
	fmt.Fprintln(output, "\nGemini Setup:")
	fmt.Fprintln(output, "Get your API key from: https://aistudio.google.com/apikey")
	fmt.Fprintln(output)
	fmt.Fprint(output, "Paste your API key (input hidden): ")

	var apiKeyStr string
	var err error

	// Check if stdin is a terminal (TTY)
	if term.IsTerminal(int(syscall.Stdin)) {
		// Read password without echo (only works with TTY)
		var apiKey []byte
		apiKey, err = term.ReadPassword(int(syscall.Stdin))
		fmt.Fprintln(output) // newline after hidden input

		if err != nil {
			return fmt.Errorf("read API key: %w", err)
		}

		apiKeyStr = strings.TrimSpace(string(apiKey))
	} else {
		// Not a TTY (e.g., in tests), read normally
		fmt.Fprintln(output)
		fmt.Fprintln(output, "(Warning: Input not hidden - not running in a terminal)")
		
		// Just return early in test environments
		fmt.Fprintln(output, "\nNo API key provided. You can add it later with:")
		fmt.Fprintln(output, "  termai auth login gemini")
		return nil
	}

	if apiKeyStr == "" {
		fmt.Fprintln(output, "\nNo API key provided. You can add it later with:")
		fmt.Fprintln(output, "  termai auth login gemini")
		return nil
	}

	// Validate the key
	fmt.Fprintln(output, "\nValidating API key...")

	client, err := gemini.NewClient(apiKeyStr)
	if err != nil {
		return fmt.Errorf("initialize Gemini client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err = client.Models(ctx)
	if err != nil {
		errMsg := err.Error()
		
		// Check for specific error types based on error message
		if strings.Contains(errMsg, "400") || 
		   strings.Contains(errMsg, "401") || 
		   strings.Contains(errMsg, "403") {
			fmt.Fprintf(output, "\n⚠ API key was rejected\n")
			fmt.Fprintln(output, "Please verify your key at https://aistudio.google.com/apikey")
			fmt.Fprintln(output, "\nYou can try again with:")
			fmt.Fprintln(output, "  termai auth login gemini")
			return nil // Don't fail onboarding
		}
		
		if strings.Contains(errMsg, "429") {
			fmt.Fprintln(output, "\nNote: Rate limited, but your key is valid.")
		} else {
			return fmt.Errorf("validate API key: %w", err)
		}
	}

	// Store the key
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	credentialsDir := filepath.Join(cfg.DataDir, "..", "credentials")
	store := secrets.NewStore(credentialsDir)

	if err := store.Set("gemini", apiKeyStr); err != nil {
		return fmt.Errorf("store API key: %w", err)
	}

	fmt.Fprintln(output, "\n✓ API key stored securely")
	fmt.Fprintf(output, "  Key: %s\n", secrets.MaskAPIKey(apiKeyStr))

	return nil
}
