package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/Rugved7/termai/internal/llm"
)

func writeSSEEvent(w http.ResponseWriter, eventType, data string) {
	fmt.Fprintf(w, "event: %s\n", eventType)
	fmt.Fprintf(w, "data: %s\n\n", data)
}

func TestChatStreamSuccess(t *testing.T) {
	var receivedChunks []string

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		if r.URL.Path != "/interactions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		if got := r.Header.Get("Accept"); got != "text/event-stream" {
			t.Errorf("unexpected Accept header: %q", got)
		}

		if got := r.Header.Get("x-goog-api-key"); got != "test-api-key" {
			t.Errorf("unexpected API key header: %q", got)
		}

		var body interactionRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
			return
		}

		if !body.Stream {
			t.Error("expected stream=true")
		}

		if body.Store {
			t.Error("expected store=false")
		}

		if body.Model != "gemini-test-model" {
			t.Errorf("unexpected model: %q", body.Model)
		}

		w.Header().Set("Content-Type", "text/event-stream")

		writeSSEEvent(w, "interaction.created",
			`{"event_type":"interaction.created"}`)

		writeSSEEvent(w, "step.start",
			`{"event_type":"step.start","step":{"type":"thought"}}`)

		writeSSEEvent(w, "step.delta",
			`{"event_type":"step.delta","delta":{"type":"thought_signature","signature":"test-signature"}}`)

		writeSSEEvent(w, "step.stop",
			`{"event_type":"step.stop"}`)

		writeSSEEvent(w, "step.start",
			`{"event_type":"step.start","step":{"type":"model_output"}}`)

		writeSSEEvent(w, "step.delta",
			`{"event_type":"step.delta","delta":{"type":"text","text":"Hello "}}`)

		writeSSEEvent(w, "step.delta",
			`{"event_type":"step.delta","delta":{"type":"text","text":"from Gemini!"}}`)

		writeSSEEvent(w, "step.stop",
			`{"event_type":"step.stop"}`)

		writeSSEEvent(w, "interaction.completed",
			`{"event_type":"interaction.completed"}`)
	})

	response, err := client.ChatStream(
		context.Background(),
		llm.ChatRequest{
			Model: "gemini-test-model",
			Messages: []llm.Message{
				{Role: "user", Content: "Hello"},
			},
		},
		func(chunk string) {
			receivedChunks = append(receivedChunks, chunk)
		},
	)

	if err != nil {
		t.Fatal(err)
	}

	expected := "Hello from Gemini!"

	if response.Message.Content != expected {
		t.Errorf("expected %q, got %q", expected, response.Message.Content)
	}

	if response.Message.Role != "assistant" {
		t.Errorf("expected assistant role, got %q", response.Message.Role)
	}

	if len(receivedChunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(receivedChunks))
	}

	if receivedChunks[0] != "Hello " ||
		receivedChunks[1] != "from Gemini!" {
		t.Errorf("unexpected chunks: %v", receivedChunks)
	}

	if strings.Contains(response.Message.Content, "test-signature") {
		t.Error("thought signature leaked into response")
	}
}

func TestChatStreamFiltersNonTextDeltas(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")

		writeSSEEvent(w, "step.delta",
			`{"event_type":"step.delta","delta":{"type":"thought_summary","text":"Private thought"}}`)

		writeSSEEvent(w, "step.delta",
			`{"event_type":"step.delta","delta":{"type":"text","text":"Visible answer"}}`)

		writeSSEEvent(w, "interaction.completed",
			`{"event_type":"interaction.completed"}`)
	})

	response, err := client.ChatStream(
		context.Background(),
		llm.ChatRequest{
			Model: "gemini-test-model",
			Messages: []llm.Message{
				{Role: "user", Content: "Hello"},
			},
		},
		nil,
	)

	if err != nil {
		t.Fatal(err)
	}

	if response.Message.Content != "Visible answer" {
		t.Errorf("unexpected response: %q", response.Message.Content)
	}
}

func TestChatStreamAPIError(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				fmt.Fprintln(w, tt.body)
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
				t.Fatal("expected API error")
			}

			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("expected error containing %q, got %q", tt.want, err)
			}
		})
	}
}

func TestChatStreamMalformedEvent(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")

		writeSSEEvent(w, "step.delta", `{invalid json`)
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
		t.Fatal("expected malformed event error")
	}

	if !strings.Contains(err.Error(), "decode Gemini stream event") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestChatStreamErrorEvent(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")

		writeSSEEvent(w, "error",
			`{"event_type":"error","error":{"message":"Generation failed"}}`)
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
		t.Fatal("expected stream error")
	}

	if !strings.Contains(err.Error(), "Generation failed") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestChatStreamInteractionFailed(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")

		writeSSEEvent(w, "interaction.failed",
			`{"event_type":"interaction.failed"}`)
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
		t.Fatal("expected interaction failure")
	}
}

func TestChatStreamRejectsIncompleteStream(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")

		writeSSEEvent(w, "step.delta",
			`{"event_type":"step.delta","delta":{"type":"text","text":"Partial response"}}`)
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
		t.Fatal("expected incomplete stream error")
	}

	if !strings.Contains(err.Error(), "before interaction completed") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestChatStreamContextCancellation(t *testing.T) {
	client, err := NewClient("test-api-key")
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = client.ChatStream(
		ctx,
		llm.ChatRequest{
			Model: "gemini-test-model",
			Messages: []llm.Message{
				{Role: "user", Content: "Hello"},
			},
		},
		nil,
	)

	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func TestChatStreamRejectsEmptyModel(t *testing.T) {
	client, err := NewClient("test-api-key")
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.ChatStream(
		context.Background(),
		llm.ChatRequest{
			Messages: []llm.Message{
				{Role: "user", Content: "Hello"},
			},
		},
		nil,
	)

	if err == nil {
		t.Fatal("expected empty model error")
	}
}

func TestChatStreamDoneWithoutCompletion(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")

		fmt.Fprintln(w, "data: [DONE]")
		fmt.Fprintln(w)
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
		t.Fatal("expected error when stream ends without interaction.completed")
	}

	expected := "Gemini stream ended before interaction completed"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected error containing %q, got %q", expected, err)
	}
}
