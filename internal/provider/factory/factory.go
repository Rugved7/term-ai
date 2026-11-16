package factory

import (
	"fmt"

	"github.com/Rugved7/termai/internal/config"
	"github.com/Rugved7/termai/internal/llm"
	"github.com/Rugved7/termai/internal/provider/gemini"
	"github.com/Rugved7/termai/internal/provider/ollama"
)

func New(cfg config.Config) (llm.Provider, error) {
	switch cfg.Provider {
	case "ollama":
		client, err := ollama.NewClient(cfg.Ollama.BaseURL)
		if err != nil {
			return nil, fmt.Errorf("initialize Ollama provider: %w", err)
		}
		return client, nil

	case "gemini":
		apiKey, err := config.ResolveAPIKey("gemini")
		if err != nil {
			return nil, fmt.Errorf("initialize Gemini provider: %w", err)
		}

		client, err := gemini.NewClient(apiKey)
		if err != nil {
			return nil, fmt.Errorf("initialize Gemini provider: %w", err)
		}
		return client, nil

	default:
		return nil, fmt.Errorf("unsupported provider %q", cfg.Provider)
	}
}
