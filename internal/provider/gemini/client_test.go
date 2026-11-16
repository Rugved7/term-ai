package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Rugved7/termai/internal/llm"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := NewClient("test-api-key")
	if err != nil {
		t.Fatal(err)
	}

	client.BaseURL = server.URL
	client.ModelsBaseURL = server.URL
	client.HTTPClient = server.Client()

	return client
}

func TestNewClient(t *testing.T) {
	client, err := NewClient("test-api-key")
	if err != nil {
		t.Fatal(err)
	}

	if client.APIKey != "test-api-key" {
		t.Error("API key was not assigned")
	}

	if client.HTTPClient == nil {
		t.Error("HTTP client should not be nil")
	}

	if client.BaseURL != defaultBaseURL {
		t.Errorf("unexpected base URL: %s", client.BaseURL)
	}
}

func TestNewClientRejectsEmptyAPIKey(t *testing.T) {
	_, err := NewClient("   ")
	if err == nil {
		t.Fatal("expected error for empty API key")
	}
}

func TestChatSuccess(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		if r.URL.Path != "/interactions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		if got := r.Header.Get("x-goog-api-key"); got != "test-api-key" {
			t.Errorf("unexpected API key header: %q", got)
		}

		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("unexpected content type: %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{
			"steps": [
				{
					"type": "user_input",
					"content": [{"type": "text", "text": "Hello"}]
				},
				{
					"type": "model_output",
					"content": [{"type": "text", "text": "Hello from Gemini!"}]
				}
			]
		}`)
	})

	response, err := client.Chat(context.Background(), llm.ChatRequest{
		Model: "gemini-test-model",
		Messages: []llm.Message{
			{Role: "user", Content: "Hello"},
		},
	})

	if err != nil {
		t.Fatal(err)
	}

	if response.Message.Role != "assistant" {
		t.Errorf("expected assistant role, got %q", response.Message.Role)
	}

	if response.Message.Content != "Hello from Gemini!" {
		t.Errorf("unexpected response: %q", response.Message.Content)
	}
}

func TestChatRequestPayload(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body interactionRequest

		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		if body.Model != "gemini-test-model" {
			t.Errorf("unexpected model: %q", body.Model)
		}

		if body.Store {
			t.Error("expected store=false")
		}

		if body.SystemInstruction != "You are a helpful assistant." {
			t.Errorf("unexpected system instruction: %q", body.SystemInstruction)
		}

		if len(body.Input) != 2 {
			t.Fatalf("expected 2 input steps, got %d", len(body.Input))
		}

		if body.Input[0].Type != "user_input" {
			t.Errorf("expected user_input, got %q", body.Input[0].Type)
		}

		if body.Input[1].Type != "model_output" {
			t.Errorf("expected model_output, got %q", body.Input[1].Type)
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{
			"steps": [{
				"type": "model_output",
				"content": [{"type": "text", "text": "Response"}]
			}]
		}`)
	})

	_, err := client.Chat(context.Background(), llm.ChatRequest{
		Model: "gemini-test-model",
		Messages: []llm.Message{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi"},
		},
	})

	if err != nil {
		t.Fatal(err)
	}
}

func TestChatAPIError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		want       string
	}{
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       `{"error":{"message":"Invalid API key"}}`,
			want:       "Invalid API key",
		},
		{
			name:       "rate limited",
			statusCode: http.StatusTooManyRequests,
			body:       `{"error":{"message":"Quota exceeded"}}`,
			want:       "Quota exceeded",
		},
		{
			name:       "server error",
			statusCode: http.StatusInternalServerError,
			body:       `{"error":{"message":"Internal server error"}}`,
			want:       "Internal server error",
		},
		{
			name:       "unstructured error",
			statusCode: http.StatusBadGateway,
			body:       "Bad gateway",
			want:       "Bad gateway",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				fmt.Fprintln(w, tt.body)
			})

			_, err := client.Chat(context.Background(), llm.ChatRequest{
				Model: "gemini-test-model",
				Messages: []llm.Message{
					{Role: "user", Content: "Hello"},
				},
			})

			if err == nil {
				t.Fatal("expected API error")
			}

			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("expected error containing %q, got %q", tt.want, err)
			}
		})
	}
}

func TestChatRejectsEmptyModel(t *testing.T) {
	client, err := NewClient("test-api-key")
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Chat(context.Background(), llm.ChatRequest{
		Messages: []llm.Message{
			{Role: "user", Content: "Hello"},
		},
	})

	if err == nil {
		t.Fatal("expected error for empty model")
	}
}

func TestChatRejectsEmptyMessages(t *testing.T) {
	client, err := NewClient("test-api-key")
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Chat(context.Background(), llm.ChatRequest{
		Model: "gemini-test-model",
	})

	if err == nil {
		t.Fatal("expected error for empty messages")
	}
}

func TestChatRejectsUnsupportedRole(t *testing.T) {
	client, err := NewClient("test-api-key")
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Chat(context.Background(), llm.ChatRequest{
		Model: "gemini-test-model",
		Messages: []llm.Message{
			{Role: "tool", Content: "Tool output"},
		},
	})

	if err == nil {
		t.Fatal("expected error for unsupported role")
	}
}

func TestChatRejectsMalformedResponse(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{invalid json`)
	})

	_, err := client.Chat(context.Background(), llm.ChatRequest{
		Model: "gemini-test-model",
		Messages: []llm.Message{
			{Role: "user", Content: "Hello"},
		},
	})

	if err == nil {
		t.Fatal("expected malformed JSON error")
	}
}

func TestChatRejectsResponseWithoutText(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"steps":[{"type":"thought"}]}`)
	})

	_, err := client.Chat(context.Background(), llm.ChatRequest{
		Model: "gemini-test-model",
		Messages: []llm.Message{
			{Role: "user", Content: "Hello"},
		},
	})

	if err == nil {
		t.Fatal("expected error for response without text")
	}
}

func TestChatContextCancellation(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.Chat(ctx, llm.ChatRequest{
		Model: "gemini-test-model",
		Messages: []llm.Message{
			{Role: "user", Content: "Hello"},
		},
	})

	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func TestModelsUsesModelsEndpoint(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}

		if r.URL.Path != "/models" {
			t.Errorf("path = %q, want /models", r.URL.Path)
		}

		if r.Header.Get("x-goog-api-key") != "test-api-key" {
			t.Error("missing API key header")
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{
			"models": [{
				"name": "models/gemini-3.8-flash",
				"supportedGenerationMethods": ["generateContent"]
			}]
		}`)
	})

	models, err := client.Models(context.Background())
	if err != nil {
		t.Fatalf("Models() error = %v", err)
	}

	if len(models) != 1 || models[0] != "gemini-3.8-flash" {
		t.Errorf("unexpected models: %v", models)
	}
}

func TestChatStreamTerminalStatuses(t *testing.T) {
	statuses := []struct {
		status string
		want   string
	}{
		{"failed", "failed"},
		{"cancelled", "cancelled"},
		{"incomplete", "incomplete"},
		{"budget_exceeded", "budget_exceeded"},
		{"requires_action", "requires an action"},
	}
	for _, tt := range statuses {
		t.Run(tt.status, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")

				fmt.Fprintf(w,
					"event: interaction.status_update\n"+
						"data: {\"event_type\":\"interaction.status_update\",\"status\":%q}\n\n",
					tt.status,
				)
			})

			_, err := client.ChatStream(
				context.Background(),
				llm.ChatRequest{
					Model: "gemini-test-model",
					Messages: []llm.Message{
						{Role: "user", Content: "Hello"},
					},
				},
				nil,
			)

			if err == nil {
				t.Fatal("expected error for terminal status")
			}

			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("expected error containing %q, got %q", tt.want, err)
			}
		})
	}
}
