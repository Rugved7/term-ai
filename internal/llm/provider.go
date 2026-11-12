package llm

import "context"

// Message represents a single conversation message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Chatrequest represents a LLM request
type ChatRequest struct {
	Model    string
	Messages []Message
}

// ChatResponse represents an LLM response
type ChatResponse struct {
	Message Message
}

// Provider defines the contract for every AI Provider
type Provider interface {
	Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
	Models(ctx context.Context) ([]string, error)
}
