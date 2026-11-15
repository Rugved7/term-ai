package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Rugved7/termai/internal/llm"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient(baseURL string) (*Client, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")

	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid Ollama base URL: %w", err)
	}

	if (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") ||
		parsedURL.Host == "" {
		return nil, fmt.Errorf(
			"invalid Ollama base URL %q: expected an absolute HTTP or HTTPS URL",
			baseURL,
		)
	}

	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}, nil
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []llm.Message `json:"messages"`
	Stream   bool          `json:"stream"`
}

type chatResponse struct {
	Message llm.Message `json:"message"`
}

type streamResponse struct {
	Message llm.Message `json:"message"`
	Done    bool        `json:"done"`
	Error   string      `json:"error"`
}

type tagsResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

func (c *Client) Chat(
	ctx context.Context,
	req llm.ChatRequest,
) (llm.ChatResponse, error) {
	payload := chatRequest{
		Model:    req.Model,
		Messages: req.Messages,
		Stream:   false,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return llm.ChatResponse{}, fmt.Errorf("encode chat request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.BaseURL+"/api/chat",
		bytes.NewReader(body),
	)
	if err != nil {
		return llm.ChatResponse{}, fmt.Errorf("create chat request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return llm.ChatResponse{}, fmt.Errorf("Ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return llm.ChatResponse{}, fmt.Errorf("Ollama returned HTTP %d", resp.StatusCode)
	}

	var result chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return llm.ChatResponse{}, fmt.Errorf("decode chat response: %w", err)
	}

	return llm.ChatResponse{
		Message: result.Message,
	}, nil
}

func (c *Client) Models(ctx context.Context) ([]string, error) {
	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.BaseURL+"/api/tags",
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create models request: %w", err)
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("Ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ollama returned HTTP %d", resp.StatusCode)
	}

	var result tagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode models response: %w", err)
	}

	models := make([]string, 0, len(result.Models))
	for _, model := range result.Models {
		name := strings.TrimSpace(model.Name)
		if name != "" {
			models = append(models, name)
		}
	}

	return models, nil
}

func (c *Client) ChatStream(
	ctx context.Context,
	req llm.ChatRequest,
	onChunk func(string),
) (llm.ChatResponse, error) {
	payload := chatRequest{
		Model:    req.Model,
		Messages: req.Messages,
		Stream:   true,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return llm.ChatResponse{}, fmt.Errorf("encode chat request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.BaseURL+"/api/chat",
		bytes.NewReader(body),
	)
	if err != nil {
		return llm.ChatResponse{}, fmt.Errorf("create chat request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return llm.ChatResponse{}, fmt.Errorf("Ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return llm.ChatResponse{}, fmt.Errorf("Ollama returned HTTP %d", resp.StatusCode)
	}

	var fullResponse strings.Builder
	decoder := json.NewDecoder(resp.Body)

	for {
		var chunk streamResponse

		if err := decoder.Decode(&chunk); err != nil {
			if err == io.EOF {
				break
			}

			return llm.ChatResponse{}, fmt.Errorf(
				"decode Ollama stream: %w",
				err,
			)
		}

		if chunk.Error != "" {
			return llm.ChatResponse{}, fmt.Errorf("Ollama stream error: %s", chunk.Error)
		}

		if chunk.Message.Content != "" {
			fullResponse.WriteString(chunk.Message.Content)

			if onChunk != nil {
				onChunk(chunk.Message.Content)
			}
		}

		if chunk.Done {
			break
		}
	}

	return llm.ChatResponse{
		Message: llm.Message{
			Role:    "assistant",
			Content: fullResponse.String(),
		},
	}, nil
}
