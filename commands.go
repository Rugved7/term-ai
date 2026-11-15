package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Rugved7/termai/internal/config"
	"github.com/Rugved7/termai/internal/llm"
	"github.com/Rugved7/termai/internal/provider/ollama"
	"github.com/Rugved7/termai/internal/storage/sqlite"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "termai",
		Short:         "A local-first AI assistant for your terminal",
		Long:          "TermAI is a terminal-based AI client powered by local and configurable LLM providers.",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	rootCmd.PersistentFlags().String(
		"config",
		"",
		"Path to a custom configuration file",
	)

	rootCmd.AddCommand(
		newModelsCmd(),
		newChatCmd(),
		newConfigCmd(),
		newVersionCmd(),
	)

	return rootCmd
}

// configPath returns the explicitly supplied config path, or the default path.
func configPath(cmd *cobra.Command) (string, error) {
	path, err := cmd.Root().PersistentFlags().GetString("config")
	if err != nil {
		return "", fmt.Errorf("read config flag: %w", err)
	}

	if path != "" {
		return path, nil
	}

	path, err = config.DefaultPath()
	if err != nil {
		return "", fmt.Errorf("resolve config path: %w", err)
	}

	return path, nil
}

func loadAppConfig(cmd *cobra.Command) (config.Config, error) {
	path, err := configPath(cmd)
	if err != nil {
		return config.Config{}, err
	}

	cfg, err := config.Load(path)
	if err != nil {
		return config.Config{}, fmt.Errorf("load configuration: %w", err)
	}

	return cfg, nil
}

func newModelsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "models",
		Short: "List available Ollama models",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadAppConfig(cmd)
			if err != nil {
				return err
			}

			client, err := ollama.NewClient(cfg.Ollama.BaseURL)
			if err != nil {
				return fmt.Errorf("initialize Ollama client: %w", err)
			}

			models, err := client.Models(cmd.Context())
			if err != nil {
				return fmt.Errorf("list models: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Available models:")

			for _, model := range models {
				fmt.Fprintln(cmd.OutOrStdout(), " -", model)
			}

			return nil
		},
	}
}

func newChatCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chat [prompt]",
		Short: "Start an interactive chat or send a one-shot prompt",
		Long: `Start an interactive conversation with an AI model.

You can also provide a prompt directly to receive a one-shot response.

Examples:
  termai chat
  termai chat "Explain goroutines"
  termai chat -m llama3.2:3b
  termai --config ./custom.yaml chat`,
		Args: cobra.ArbitraryArgs,
	}

	cmd.Flags().StringP(
		"model",
		"m",
		"",
		"Model to use (overrides the configured model)",
	)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		cfg, err := loadAppConfig(cmd)
		if err != nil {
			return err
		}

		model := cfg.Model

		modelFlagChanged := cmd.Flags().Changed("model")
		if modelFlagChanged {
			model, err = cmd.Flags().GetString("model")
			if err != nil {
				return fmt.Errorf("read model flag: %w", err)
			}

			if strings.TrimSpace(model) == "" {
				return fmt.Errorf("model name cannot be empty")
			}
		}

		return runChat(cmd, args, cfg, model)
	}

	return cmd
}

func runChat(
	cmd *cobra.Command,
	args []string,
	cfg config.Config,
	model string,
) error {
	ctx := cmd.Context()

	client, err := ollama.NewClient(cfg.Ollama.BaseURL)
	if err != nil {
		return fmt.Errorf("initialize Ollama client: %w", err)
	}

	prompt := strings.Join(args, " ")

	// One-shot chat.
	if prompt != "" {
		request := llm.ChatRequest{
			Model: model,
			Messages: []llm.Message{
				{
					Role:    "user",
					Content: prompt,
				},
			},
		}

		chatCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()

		response, err := client.Chat(chatCtx, request)
		if err != nil {
			return fmt.Errorf("chat: %w", err)
		}

		fmt.Fprintln(cmd.OutOrStdout(), "\nTermAI:")
		fmt.Fprintln(cmd.OutOrStdout(), response.Message.Content)

		return nil
	}

	// Interactive chat requires persistent storage.
	store, err := sqlite.New(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("initialize storage: %w", err)
	}

	defer func() {
		if err := store.Close(); err != nil {
			fmt.Fprintln(cmd.ErrOrStderr(), "Database close error:", err)
		}
	}()

	startChat(client, model, store)

	return nil
}

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage TermAI configuration",
	}

	cmd.AddCommand(
		newConfigInitCmd(),
		newConfigShowCmd(),
		newConfigPathCmd(),
	)

	return cmd
}

func newConfigInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create the default configuration file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := configPath(cmd)
			if err != nil {
				return err
			}

			if err := config.CreateDefault(path); err != nil {
				if errors.Is(err, os.ErrExist) {
					return fmt.Errorf("configuration already exists at %q", path)
				}

				return fmt.Errorf("create configuration at %q: %w", path, err)
			}

			fmt.Fprintf(
				cmd.OutOrStdout(),
				"Configuration created: %s\n",
				path,
			)

			return nil
		},
	}
}

func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Display the effective configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadAppConfig(cmd)
			if err != nil {
				return err
			}

			output, err := yaml.Marshal(cfg)
			if err != nil {
				return fmt.Errorf("format configuration: %w", err)
			}

			_, err = cmd.OutOrStdout().Write(output)
			return err
		},
	}
}

func newConfigPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the configuration file path",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := configPath(cmd)
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), path)
			return nil
		},
	}
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print TermAI version information",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "TermAI version: %s\n", version)
			fmt.Fprintf(cmd.OutOrStdout(), "Git commit:     %s\n", commit)
			fmt.Fprintf(cmd.OutOrStdout(), "Build date:     %s\n", buildDate)
		},
	}
}
