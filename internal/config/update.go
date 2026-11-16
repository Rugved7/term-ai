package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Set updates a supported configuration field and persists the result.
// If the configuration file does not exist, defaults are used.
func Set(path, key, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s cannot be empty", key)
	}

	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return err
		}
	}

	cfg, err := Load(path)
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	value = strings.TrimSpace(value)

	switch key {
	case "provider":
		cfg.Provider = value

	case "model":
		cfg.Model = value

	case "ollama.base_url":
		if err := validateOllamaURL(value); err != nil {
			return err
		}
		cfg.Ollama.BaseURL = value

	default:
		return fmt.Errorf(
			"unsupported configuration key %q",
			key,
		)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	return writeConfig(path, cfg)
}

func validateOllamaURL(rawURL string) error {
	parsedURL, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return fmt.Errorf("invalid Ollama URL: %w", err)
	}

	if (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") ||
		parsedURL.Host == "" {
		return errors.New(
			"Ollama URL must be an absolute HTTP or HTTPS URL",
		)
	}

	return nil
}

// writeConfig writes configuration through a temporary file
// before replacing the destination.
func writeConfig(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create configuration directory: %w", err)
	}

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("marshal configuration: %w", err)
	}

	tempFile, err := os.CreateTemp(
		filepath.Dir(path),
		".config-*.tmp",
	)
	if err != nil {
		return fmt.Errorf("create temporary configuration: %w", err)
	}

	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	if err := tempFile.Chmod(0600); err != nil {
		tempFile.Close()
		return fmt.Errorf("set configuration permissions: %w", err)
	}

	if _, err := tempFile.Write(data); err != nil {
		tempFile.Close()
		return fmt.Errorf("write configuration: %w", err)
	}

	if err := tempFile.Sync(); err != nil {
		tempFile.Close()
		return fmt.Errorf("sync configuration: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close temporary configuration: %w", err)
	}

	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("replace configuration: %w", err)
	}

	return nil
}
