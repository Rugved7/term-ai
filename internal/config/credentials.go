package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rugved7/termai/internal/secrets"
)

// APIKeyEnvVar returns the environment variable used by a cloud provider.
func APIKeyEnvVar(provider string) (string, error) {
	switch provider {
	case "openai":
		return "OPENAI_API_KEY", nil
	case "gemini":
		return "GEMINI_API_KEY", nil
	case "anthropic":
		return "ANTHROPIC_API_KEY", nil
	case "groq":
		return "GROQ_API_KEY", nil
	default:
		return "", fmt.Errorf(
			"API key environment variable is not defined for provider %q",
			provider,
		)
	}
}

// ResolveAPIKey retrieves a provider's API key.
// Lookup order: environment variable → OS keychain → file fallback
func ResolveAPIKey(provider string) (string, error) {
	envVar, err := APIKeyEnvVar(provider)
	if err != nil {
		return "", err
	}

	// First check environment variable
	apiKey := strings.TrimSpace(os.Getenv(envVar))
	if apiKey != "" {
		return apiKey, nil
	}

	// Fall back to secrets store (keychain or file)
	cfg, err := Load("")
	if err != nil {
		return "", fmt.Errorf("load config: %w", err)
	}

	credentialsDir := filepath.Join(cfg.DataDir, "..", "credentials")
	store := secrets.NewStore(credentialsDir)

	key, err := store.Get(provider)
	if err != nil {
		if err == secrets.ErrNotFound {
			return "", fmt.Errorf(
				"missing API key: set the %s environment variable or run 'termai auth login %s'",
				envVar,
				provider,
			)
		}
		return "", fmt.Errorf("retrieve API key: %w", err)
	}

	return key, nil
}

// MaskAPIKey returns a masked version of an API key for display
func MaskAPIKey(key string) string {
	return secrets.MaskAPIKey(key)
}
