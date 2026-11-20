package secrets

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zalando/go-keyring"
)

const (
	serviceName = "termai"
)

var (
	// ErrNotFound is returned when a credential is not found
	ErrNotFound = errors.New("credential not found")
)

// Store handles secure credential storage with OS keychain and file fallback
type Store struct {
	credentialsDir string
}

// NewStore creates a new credential store
// credentialsDir is used for file-based fallback (typically ~/.config/termai/credentials)
func NewStore(credentialsDir string) *Store {
	return &Store{
		credentialsDir: credentialsDir,
	}
}

// Set stores a credential for the given provider
// It attempts to use the OS keychain first, falling back to file storage
func (s *Store) Set(provider, apiKey string) error {
	if provider == "" {
		return errors.New("provider cannot be empty")
	}

	if apiKey == "" {
		return errors.New("API key cannot be empty")
	}

	// Try OS keychain first
	err := keyring.Set(serviceName, provider, apiKey)
	if err == nil {
		return nil
	}

	// Fall back to file storage
	return s.setFile(provider, apiKey)
}

// Get retrieves a credential for the given provider
// Lookup order: environment variable → OS keychain → file
func (s *Store) Get(provider string) (string, error) {
	if provider == "" {
		return "", errors.New("provider cannot be empty")
	}

	// 1. Check environment variable
	envKey := os.Getenv(envVarName(provider))
	if envKey != "" {
		return envKey, nil
	}

	// 2. Try OS keychain
	key, err := keyring.Get(serviceName, provider)
	if err == nil {
		return key, nil
	}

	// 3. Try file fallback
	key, err = s.getFile(provider)
	if err == nil {
		return key, nil
	}

	return "", ErrNotFound
}

// Delete removes a credential for the given provider
func (s *Store) Delete(provider string) error {
	if provider == "" {
		return errors.New("provider cannot be empty")
	}

	// Try to delete from keychain
	_ = keyring.Delete(serviceName, provider)

	// Also delete from file
	return s.deleteFile(provider)
}

// Has checks if a credential exists for the given provider
func (s *Store) Has(provider string) bool {
	_, err := s.Get(provider)
	return err == nil
}

// setFile stores a credential in a file with restricted permissions
func (s *Store) setFile(provider, apiKey string) error {
	if err := os.MkdirAll(s.credentialsDir, 0700); err != nil {
		return fmt.Errorf("create credentials directory: %w", err)
	}

	filePath := filepath.Join(s.credentialsDir, provider)

	// Write with restricted permissions (owner read/write only)
	if err := os.WriteFile(filePath, []byte(apiKey), 0600); err != nil {
		return fmt.Errorf("write credential file: %w", err)
	}

	return nil
}

// getFile retrieves a credential from a file
func (s *Store) getFile(provider string) (string, error) {
	filePath := filepath.Join(s.credentialsDir, provider)

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("read credential file: %w", err)
	}

	return string(data), nil
}

// deleteFile removes a credential file
func (s *Store) deleteFile(provider string) error {
	filePath := filepath.Join(s.credentialsDir, provider)

	err := os.Remove(filePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete credential file: %w", err)
	}

	return nil
}

// envVarName returns the environment variable name for a provider
func envVarName(provider string) string {
	switch provider {
	case "gemini":
		return "GEMINI_API_KEY"
	case "openai":
		return "OPENAI_API_KEY"
	case "anthropic":
		return "ANTHROPIC_API_KEY"
	default:
		return ""
	}
}
