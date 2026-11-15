package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/Rugved7/termai/internal/config"
	"github.com/Rugved7/termai/internal/provider/ollama"
	"gopkg.in/yaml.v3"
)

// executeCommand executes a fresh TermAI root command and captures its output.
func executeCommand(args ...string) (string, string, error) {
	cmd := newRootCmd()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)

	err := cmd.Execute()

	return stdout.String(), stderr.String(), err
}

// writeTestConfig creates a temporary configuration file.
func writeTestConfig(
	t *testing.T,
	baseURL string,
	model string,
	provider string,
) string {
	t.Helper()

	cfg := config.Config{
		Provider: provider,
		Model:    model,
		DataDir:  filepath.Join(t.TempDir(), "data"),
		Ollama: config.OllamaConfig{
			BaseURL: baseURL,
		},
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal test config: %v", err)
	}

	path := filepath.Join(t.TempDir(), "config.yaml")

	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write test config: %v", err)
	}

	return path
}

func TestRootCommand(t *testing.T) {
	stdout, _, err := executeCommand()

	if err != nil {
		t.Fatalf("execute root command: %v", err)
	}

	for _, expected := range []string{
		"chat",
		"config",
		"models",
		"version",
	} {
		if !strings.Contains(stdout, expected) {
			t.Errorf("root help does not contain %q", expected)
		}
	}
}

func TestConfigInit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")

	stdout, _, err := executeCommand(
		"--config", path,
		"config", "init",
	)

	if err != nil {
		t.Fatalf("config init: %v", err)
	}

	if !strings.Contains(stdout, "Configuration created") {
		t.Errorf("unexpected output: %s", stdout)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("configuration file was not created: %v", err)
	}

	// A second initialization must not overwrite the existing file.
	_, _, err = executeCommand(
		"--config", path,
		"config", "init",
	)

	if err == nil {
		t.Fatal("expected an error when config already exists")
	}

	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestConfigPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "custom.yaml")

	stdout, _, err := executeCommand(
		"--config", path,
		"config", "path",
	)

	if err != nil {
		t.Fatalf("config path: %v", err)
	}

	if strings.TrimSpace(stdout) != path {
		t.Errorf(
			"config path = %q, want %q",
			strings.TrimSpace(stdout),
			path,
		)
	}
}

func TestConfigShow(t *testing.T) {
	path := writeTestConfig(
		t,
		"http://localhost:11434",
		"llama3.2:3b",
		"ollama",
	)

	stdout, _, err := executeCommand(
		"--config", path,
		"config", "show",
	)

	if err != nil {
		t.Fatalf("config show: %v", err)
	}

	for _, expected := range []string{
		"provider: ollama",
		"model: llama3.2:3b",
		"base_url: http://localhost:11434",
	} {
		if !strings.Contains(stdout, expected) {
			t.Errorf("config output does not contain %q", expected)
		}
	}
}

func TestConfigShowRejectsUnsupportedProvider(t *testing.T) {
	path := writeTestConfig(
		t,
		"http://localhost:11434",
		"test-model",
		"openai",
	)

	_, _, err := executeCommand(
		"--config", path,
		"config", "show",
	)

	if err == nil {
		t.Fatal("expected unsupported provider error")
	}

	if !strings.Contains(err.Error(), "unsupported provider") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestModelsCommand(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %q, want GET", r.Method)
			}

			if r.URL.Path != "/api/tags" {
				t.Errorf("path = %q, want /api/tags", r.URL.Path)
			}

			w.Header().Set("Content-Type", "application/json")

			_, _ = fmt.Fprint(w, `{
				"models": [
					{"name": "llama3.2:3b"},
					{"name": "codellama:7b"}
				]
			}`)
		},
	))
	defer server.Close()

	path := writeTestConfig(
		t,
		server.URL,
		"llama3.2:3b",
		"ollama",
	)

	stdout, _, err := executeCommand(
		"--config", path,
		"models",
	)

	if err != nil {
		t.Fatalf("models command: %v", err)
	}

	for _, expected := range []string{
		"Available models:",
		"llama3.2:3b",
		"codellama:7b",
	} {
		if !strings.Contains(stdout, expected) {
			t.Errorf("models output does not contain %q", expected)
		}
	}
}

func TestModelsCommandReturnsProviderError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			http.Error(
				w,
				"Ollama unavailable",
				http.StatusServiceUnavailable,
			)
		},
	))
	defer server.Close()

	path := writeTestConfig(
		t,
		server.URL,
		"llama3.2:3b",
		"ollama",
	)

	_, _, err := executeCommand(
		"--config", path,
		"models",
	)

	if err == nil {
		t.Fatal("expected provider error")
	}

	if !strings.Contains(err.Error(), "HTTP 503") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestChatCommandUsesModelOverride(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %q, want POST", r.Method)
			}

			if r.URL.Path != "/api/chat" {
				t.Errorf("path = %q, want /api/chat", r.URL.Path)
			}

			var request struct {
				Model    string `json:"model"`
				Messages []struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"messages"`
				Stream bool `json:"stream"`
			}

			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Errorf("decode request: %v", err)
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}

			if request.Model != "override-model" {
				t.Errorf(
					"model = %q, want override-model",
					request.Model,
				)
			}

			if request.Stream {
				t.Error("expected non-streaming request")
			}

			if len(request.Messages) != 1 {
				t.Fatalf("message count = %d, want 1", len(request.Messages))
			}

			if request.Messages[0].Role != "user" {
				t.Errorf("role = %q, want user", request.Messages[0].Role)
			}

			if request.Messages[0].Content != "Explain dependency injection" {
				t.Errorf(
					"prompt = %q, want expected prompt",
					request.Messages[0].Content,
				)
			}

			w.Header().Set("Content-Type", "application/json")

			_, _ = fmt.Fprint(w, `{
				"message": {
					"role": "assistant",
					"content": "Dependency injection provides dependencies externally."
				}
			}`)
		},
	))
	defer server.Close()

	path := writeTestConfig(
		t,
		server.URL,
		"configured-model",
		"ollama",
	)

	stdout, _, err := executeCommand(
		"--config", path,
		"chat",
		"-m", "override-model",
		"Explain dependency injection",
	)

	if err != nil {
		t.Fatalf("chat command: %v", err)
	}

	if !strings.Contains(
		stdout,
		"Dependency injection provides dependencies externally.",
	) {
		t.Errorf("unexpected chat output: %s", stdout)
	}
}

func TestChatCommandUsesConfiguredModel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var request struct {
				Model string `json:"model"`
			}

			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Errorf("decode request: %v", err)
				return
			}

			if request.Model != "configured-model" {
				t.Errorf(
					"model = %q, want configured-model",
					request.Model,
				)
			}

			w.Header().Set("Content-Type", "application/json")

			_, _ = fmt.Fprint(w, `{
				"message": {
					"role": "assistant",
					"content": "Hello from configured model."
				}
			}`)
		},
	))
	defer server.Close()

	path := writeTestConfig(
		t,
		server.URL,
		"configured-model",
		"ollama",
	)

	stdout, _, err := executeCommand(
		"--config", path,
		"chat",
		"Say hello",
	)

	if err != nil {
		t.Fatalf("chat command: %v", err)
	}

	if !strings.Contains(stdout, "Hello from configured model.") {
		t.Errorf("unexpected chat output: %s", stdout)
	}
}

func TestChatCommandRejectsEmptyModelOverride(t *testing.T) {
	path := writeTestConfig(
		t,
		"http://localhost:11434",
		"configured-model",
		"ollama",
	)

	_, _, err := executeCommand(
		"--config", path,
		"chat",
		"-m", "",
		"Say hello",
	)

	if err == nil {
		t.Fatal("expected empty model error")
	}

	if !strings.Contains(err.Error(), "model name cannot be empty") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFormatOllamaError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "deadline exceeded",
			err:  context.DeadlineExceeded,
			want: "timed out",
		},
		{
			name: "context canceled",
			err:  context.Canceled,
			want: "was canceled",
		},
		{
			name: "model not found",
			err: &ollama.APIError{
				StatusCode: http.StatusNotFound,
				Message:    "model not found",
			},
			want: "ollama pull <model-name>",
		},
		{
			name: "generic API error",
			err: &ollama.APIError{
				StatusCode: http.StatusInternalServerError,
				Message:    "internal server error",
			},
			want: "HTTP 500",
		},
		{
			name: "generic error",
			err:  errors.New("unexpected failure"),
			want: "unexpected failure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatOllamaError("test operation", tt.err)

			if got == nil {
				t.Fatal("expected an error")
			}

			if !strings.Contains(got.Error(), tt.want) {
				t.Errorf(
					"error = %q, want it to contain %q",
					got,
					tt.want,
				)
			}
		})
	}
}

func TestVerboseFlag(t *testing.T) {
	cmd := newRootCmd()

	flag := cmd.PersistentFlags().Lookup("verbose")
	if flag == nil {
		t.Fatal("expected --verbose persistent flag")
	}

	if flag.DefValue != "false" {
		t.Fatalf("expected verbose disabled by default, got %q", flag.DefValue)
	}
}

func TestConfigureLogger(t *testing.T) {
	tests := []struct {
		name    string
		verbose bool
		want    slog.Level
	}{
		{
			name:    "default logging",
			verbose: false,
			want:    slog.LevelWarn,
		},
		{
			name:    "verbose logging",
			verbose: true,
			want:    slog.LevelDebug,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newRootCmd()

			if err := cmd.PersistentFlags().Set(
				"verbose",
				strconv.FormatBool(tt.verbose),
			); err != nil {
				t.Fatal(err)
			}

			original := slog.Default()
			t.Cleanup(func() {
				slog.SetDefault(original)
			})

			if err := configureLogger(cmd); err != nil {
				t.Fatal(err)
			}

			if !slog.Default().Enabled(context.Background(), tt.want) {
				t.Fatalf("expected logging level %s to be enabled", tt.want)
			}

			if tt.verbose && slog.Default().Enabled(
				context.Background(),
				slog.LevelDebug-4,
			) {
				t.Fatal("unexpected logging level below debug")
			}

			if !tt.verbose && slog.Default().Enabled(
				context.Background(),
				slog.LevelInfo,
			) {
				t.Fatal("info logging should be disabled by default")
			}
		})
	}
}
