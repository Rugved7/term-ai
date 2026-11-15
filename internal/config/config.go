package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Provider string       `yaml:"provider"`
	Model    string       `yaml:"model"`
	DataDir  string       `yaml:"data_dir"`
	Ollama   OllamaConfig `yaml:"ollama"`
}

type OllamaConfig struct {
	BaseURL string `yaml:"base_url"`
}

// Default returns the default TermAI configuration.
func Default() (Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return Config{}, fmt.Errorf("get home directory: %w", err)
	}

	return Config{
		Provider: "ollama",
		Model:    "llama3.2:3b",
		DataDir:  filepath.Join(homeDir, ".termai"),
		Ollama: OllamaConfig{
			BaseURL: "http://localhost:11434",
		},
	}, nil
}

// DefaultPath returns the default configuration file location.
func DefaultPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("get configuration directory: %w", err)
	}

	return filepath.Join(configDir, "termai", "config.yaml"), nil
}

// Load reads the configuration file.
// Missing files result in default configuration.
func Load(path string) (Config, error) {
	cfg, err := Default()
	if err != nil {
		return Config{}, err
	}

	if path == "" {
		path, err = DefaultPath()
		if err != nil {
			return Config{}, err
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}

		return Config{}, fmt.Errorf("read configuration: %w", err)
	}

	// Unmarshal into defaults so omitted YAML fields retain defaults.
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse configuration: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// Validate checks whether the configuration is usable.
func (c Config) Validate() error {
	if c.Provider != "ollama" {
		return fmt.Errorf("unsupported provider %q", c.Provider)
	}

	if c.Model == "" {
		return errors.New("model cannot be empty")
	}

	if c.DataDir == "" {
		return errors.New("data directory cannot be empty")
	}

	parsedURL, err := url.ParseRequestURI(c.Ollama.BaseURL)
	if err != nil {
		return fmt.Errorf("invalid Ollama URL: %w", err)
	}

	if (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") ||
		parsedURL.Host == "" {
		return errors.New("Ollama URL must be an absolute HTTP or HTTPS URL")
	}

	return nil
}

// CreateDefault creates a default configuration file without overwriting
// an existing configuration.
func CreateDefault(path string) error {
	cfg, err := Default()
	if err != nil {
		return err
	}

	if path == "" {
		path, err = DefaultPath()
		if err != nil {
			return err
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create configuration directory: %w", err)
	}

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("marshal configuration: %w", err)
	}

	file, err := os.OpenFile(
		path,
		os.O_WRONLY|os.O_CREATE|os.O_EXCL,
		0600,
	)
	if err != nil {
		return fmt.Errorf("create configuration file: %w", err)
	}

	if _, err := file.Write(data); err != nil {
		file.Close()
		os.Remove(path)
		return fmt.Errorf("write configuration: %w", err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("close configuration file: %w", err)
	}

	return nil
}
