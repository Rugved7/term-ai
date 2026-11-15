package ollama

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/Rugved7/termai/internal/llm"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		wantURL string
		wantErr bool
	}{
		{
			name:    "valid HTTP URL",
			baseURL: "http://localhost:11434",
			wantURL: "http://localhost:11434",
		},
		{
			name:    "valid HTTPS URL",
			baseURL: "https://ollama.example.com",
			wantURL: "https://ollama.example.com",
		},
		{
			name:    "trims whitespace and trailing slash",
			baseURL: "  http://localhost:11434/  ",
			wantURL: "http://localhost:11434",
		},
		{
			name:    "empty URL",
			baseURL: "",
			wantErr: true,
		},
		{
			name:    "relative URL",
			baseURL: "/api",
			wantErr: true,
		},
		{
			name:    "unsupported scheme",
			baseURL: "ftp://localhost:11434",
			wantErr: true,
		},
		{
			name:    "missing host",
			baseURL: "http://",
			wantErr: true,
		},
		{
			name:    "malformed URL",
			baseURL: "://invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.baseURL)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if client.BaseURL != tt.wantURL {
				t.Errorf(
					"BaseURL = %q, want %q",
					client.BaseURL,
					tt.wantURL,
				)
			}

			if client.HTTPClient == nil {
				t.Fatal("HTTPClient should not be nil")
			}
		})
	}
}

func TestChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %q, want POST", r.Method)
			}

			if r.URL.Path != "/api/chat" {
				t.Errorf("path = %q, want /api/chat", r.URL.Path)
			}

			if got := r.Header.Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", got)
			}

			var request chatRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Errorf("decode request: %v", err)
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}

			if request.Model != "llama3.2:3b" {
				t.Errorf("model = %q, want llama3.2:3b", request.Model)
			}

			if request.Stream {
				t.Error("Stream = true, want false")
			}

			if len(request.Messages) != 1 {
				t.Fatalf("message count = %d, want 1", len(request.Messages))
			}

			if request.Messages[0].Role != "user" {
				t.Errorf(
					"message role = %q, want user",
					request.Messages[0].Role,
				)
			}

			if request.Messages[0].Content != "Explain Go interfaces" {
				t.Errorf(
					"message content = %q, want expected prompt",
					request.Messages[0].Content,
				)
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(chatResponse{
				Message: llm.Message{
					Role:    "assistant",
					Content: "An interface defines behavior.",
				},
			})
		},
	))
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	response, err := client.Chat(context.Background(), llm.ChatRequest{
		Model: "llama3.2:3b",
		Messages: []llm.Message{
			{
				Role:    "user",
				Content: "Explain Go interfaces",
			},
		},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}

	if response.Message.Role != "assistant" {
		t.Errorf(
			"response role = %q, want assistant",
			response.Message.Role,
		)
	}

	if response.Message.Content != "An interface defines behavior." {
		t.Errorf(
			"response content = %q, want expected response",
			response.Message.Content,
		)
	}
}

func TestModels(t *testing.T) {
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
					{"name": "  codellama:7b  "},
					{"name": "   "}
				]
			}`)
		},
	))
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	models, err := client.Models(context.Background())
	if err != nil {
		t.Fatalf("Models: %v", err)
	}

	want := []string{
		"llama3.2:3b",
		"codellama:7b",
	}

	if !reflect.DeepEqual(models, want) {
		t.Errorf("models = %#v, want %#v", models, want)
	}
}

func TestChatStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %q, want POST", r.Method)
			}

			if r.URL.Path != "/api/chat" {
				t.Errorf("path = %q, want /api/chat", r.URL.Path)
			}

			var request chatRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Errorf("decode request: %v", err)
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}

			if !request.Stream {
				t.Error("Stream = false, want true")
			}

			w.Header().Set("Content-Type", "application/x-ndjson")

			_, _ = fmt.Fprintln(w, `{"message":{"role":"assistant","content":"Hello "},"done":false}`)
			_, _ = fmt.Fprintln(w, `{"message":{"role":"assistant","content":"world!"},"done":false}`)
			_, _ = fmt.Fprintln(w, `{"message":{"role":"assistant","content":""},"done":true}`)
		},
	))
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	var chunks []string

	response, err := client.ChatStream(
		context.Background(),
		llm.ChatRequest{
			Model: "llama3.2:3b",
			Messages: []llm.Message{
				{
					Role:    "user",
					Content: "Say hello",
				},
			},
		},
		func(chunk string) {
			chunks = append(chunks, chunk)
		},
	)
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}

	wantChunks := []string{"Hello ", "world!"}
	if !reflect.DeepEqual(chunks, wantChunks) {
		t.Errorf("chunks = %#v, want %#v", chunks, wantChunks)
	}

	if response.Message.Role != "assistant" {
		t.Errorf(
			"response role = %q, want assistant",
			response.Message.Role,
		)
	}

	if response.Message.Content != "Hello world!" {
		t.Errorf(
			"response content = %q, want %q",
			response.Message.Content,
			"Hello world!",
		)
	}
}

func TestChatStreamReturnsOllamaError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/x-ndjson")
			_, _ = fmt.Fprintln(w, `{"error":"model not found"}`)
		},
	))
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = client.ChatStream(
		context.Background(),
		llm.ChatRequest{Model: "missing-model"},
		nil,
	)

	if err == nil {
		t.Fatal("expected an error, got nil")
	}

	if !strings.Contains(err.Error(), "model not found") {
		t.Errorf("error = %q, want model not found", err)
	}
}

func TestModelsReturnsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "server unavailable", http.StatusServiceUnavailable)
		},
	))
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = client.Models(context.Background())
	if err == nil {
		t.Fatal("expected an HTTP error, got nil")
	}

	if !strings.Contains(err.Error(), "HTTP 503") {
		t.Errorf("error = %q, want HTTP 503", err)
	}
}

func TestChatReturnsMalformedJSONError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"message":`)
		},
	))
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = client.Chat(context.Background(), llm.ChatRequest{
		Model: "llama3.2:3b",
	})
	if err == nil {
		t.Fatal("expected a JSON decoding error, got nil")
	}

	if !strings.Contains(err.Error(), "decode chat response") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestChatRespectsContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			<-r.Context().Done()
		},
	))
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = client.Chat(ctx, llm.ChatRequest{
		Model: "llama3.2:3b",
	})
	if err == nil {
		t.Fatal("expected a context cancellation error, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", err)
	}
}

func TestHTTPResponsesReturnAPIError(t *testing.T) {
	tests := []struct {
		name string
		path string
		call func(*Client) error
	}{
		{
			name: "Chat",
			path: "/api/chat",
			call: func(c *Client) error {
				_, err := c.Chat(context.Background(), llm.ChatRequest{
					Model: "missing-model",
				})
				return err
			},
		},
		{
			name: "Models",
			path: "/api/tags",
			call: func(c *Client) error {
				_, err := c.Models(context.Background())
				return err
			},
		},
		{
			name: "ChatStream",
			path: "/api/chat",
			call: func(c *Client) error {
				_, err := c.ChatStream(
					context.Background(),
					llm.ChatRequest{Model: "missing-model"},
					nil,
				)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != tt.path {
						t.Errorf("path = %q, want %q", r.URL.Path, tt.path)
					}

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusNotFound)

					_, _ = fmt.Fprintln(w, `{"error":"model not found"}`)
				},
			))
			defer server.Close()

			client, err := NewClient(server.URL)
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}

			err = tt.call(client)
			if err == nil {
				t.Fatal("expected an error, got nil")
			}

			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("error type = %T, want *APIError", err)
			}

			if apiErr.StatusCode != http.StatusNotFound {
				t.Errorf(
					"status code = %d, want %d",
					apiErr.StatusCode,
					http.StatusNotFound,
				)
			}

			if apiErr.Message != "model not found" {
				t.Errorf(
					"message = %q, want %q",
					apiErr.Message,
					"model not found",
				)
			}
		})
	}
}
