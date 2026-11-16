package gemini

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestModelsSuccess(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}

		if r.URL.Path != "/models" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		if got := r.Header.Get("x-goog-api-key"); got != "test-api-key" {
			t.Errorf("unexpected API key header: %q", got)
		}

		if got := r.URL.Query().Get("pageSize"); got != "1000" {
			t.Errorf("unexpected pageSize: %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{
			"models": [
				{
					"name": "models/gemini-2.5-flash",
					"supportedGenerationMethods": ["generateContent"]
				},
				{
					"name": "models/gemini-2.5-pro",
					"supportedGenerationMethods": ["generateContent"]
				},
				{
					"name": "models/embedding-001",
					"supportedGenerationMethods": ["embedContent"]
				}
			]
		}`)
	})

	models, err := client.Models(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{
		"gemini-2.5-flash",
		"gemini-2.5-pro",
	}

	if len(models) != len(expected) {
		t.Fatalf("expected %d models, got %d: %v", len(expected), len(models), models)
	}

	for i, model := range expected {
		if models[i] != model {
			t.Errorf("model[%d]: expected %q, got %q", i, model, models[i])
		}
	}
}

func TestModelsPagination(t *testing.T) {
	var requests int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++

		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Query().Get("pageToken") {
		case "":
			fmt.Fprintln(w, `{
			"models": [{
				"name": "models/gemini-2.5-flash",
				"supportedGenerationMethods": ["generateContent"]
			}],
			"nextPageToken": "next-page"
		}`)

		case "next-page":
			fmt.Fprintln(w, `{
			"models": [{
				"name": "models/gemini-2.5-pro",
				"supportedGenerationMethods": ["generateContent"]
			}]
		}`)

		default:
			t.Errorf("unexpected page token: %q", r.URL.Query().Get("pageToken"))
			http.Error(w, "unexpected page token", http.StatusBadRequest)
		}
	}))

	defer server.Close()

	client, err := NewClient("test-api-key")
	if err != nil {
		t.Fatal(err)
	}
	client.BaseURL = server.URL
	client.ModelsBaseURL = server.URL
	client.HTTPClient = server.Client()

	models, err := client.Models(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if requests != 2 {
		t.Errorf("expected 2 requests, got %d", requests)
	}

	expected := []string{
		"gemini-2.5-flash",
		"gemini-2.5-pro",
	}

	if len(models) != len(expected) {
		t.Fatalf("expected %d models, got %d", len(expected), len(models))
	}

	for i, model := range expected {
		if models[i] != model {
			t.Errorf("model[%d]: expected %q, got %q", i, model, models[i])
		}
	}
}

func TestModelsAPIError(t *testing.T) {
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

			_, err := client.Models(context.Background())
			if err == nil {
				t.Fatal("expected API error")
			}

			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("expected error containing %q, got %q", tt.want, err)
			}
		})
	}
}

func TestModelsMalformedResponse(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{invalid json`)
	})

	_, err := client.Models(context.Background())
	if err == nil {
		t.Fatal("expected malformed JSON error")
	}
}

func TestModelsContextCancellation(t *testing.T) {
	client, err := NewClient("test-api-key")
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = client.Models(ctx)
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func TestModelsRejectsRepeatedPageToken(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{
			"models": [],
			"nextPageToken": "same-token"
		}`)
	})

	_, err := client.Models(context.Background())
	if err == nil {
		t.Fatal("expected repeated page token error")
	}

	if !strings.Contains(err.Error(), "repeated page token") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestModelsExcludesUnsupportedInteractionsModels(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{
			"models": [
				{
					"name": "models/gemini-2.5-flash",
					"supportedGenerationMethods": ["generateContent"]
				},
				{
					"name": "models/some-unknown-model",
					"supportedGenerationMethods": ["generateContent"]
				},
				{
					"name": "models/embedding-001",
					"supportedGenerationMethods": ["embedContent"]
				}
			]
		}`)
	})

	models, err := client.Models(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if len(models) != 1 {
		t.Fatalf("expected 1 supported model, got %v", models)
	}

	if models[0] != "gemini-2.5-flash" {
		t.Errorf("unexpected model: %s", models[0])
	}
}
