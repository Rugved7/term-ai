package config

import (
	"fmt"
	"os"
	"strings"
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

// ResolveAPIKey retrieves a provider's API key from the environment.
func ResolveAPIKey(provider string) (string, error) {
	envVar, err := APIKeyEnvVar(provider)
	if err != nil {
		return "", err
	}

	apiKey := strings.TrimSpace(os.Getenv(envVar))
	if apiKey == "" {
		return "", fmt.Errorf(
			"missing API key: set the %s environment variable",
			envVar,
		)
	}

	return apiKey, nil
}
