package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Rugved7/termai/internal/config"
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
		model = "gemini-2.5-flash"
	default:
		return fmt.Errorf("invalid selection %q: choose 1 or 2", strings.TrimSpace(choice))
	}

	if err := config.Set(path, "provider", provider); err != nil {
		return fmt.Errorf("save provider configuration: %w", err)
	}

	if err := config.Set(path, "model", model); err != nil {
		return fmt.Errorf("save model configuration: %w", err)
	}

	fmt.Fprintf(output, "\nTermAI configured to use %s.\n", provider)

	if provider == "ollama" {
		fmt.Fprintln(output, "Make sure Ollama is installed and running.")
		fmt.Fprintln(output, "Pull the default model with: ollama pull "+model)
	} else {
		if strings.TrimSpace(os.Getenv("GEMINI_API_KEY")) == "" {
			fmt.Fprintln(output, "\nGemini API key not detected.")
			fmt.Fprintln(output, "Set your key using the GEMINI_API_KEY environment variable.")
		}
	}

	fmt.Fprintln(output, "\nConfiguration saved:", path)

	return nil
}
