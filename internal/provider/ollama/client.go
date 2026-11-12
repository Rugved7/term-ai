package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Rugved7/termai/internal/llm"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient() *Client {
	return &Client{
		BaseURL: "http://localhost:11434",
		HTTPClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []llm.Message `json:"messages"`
	Stream   bool          `json:"stream"`
}

type chatResponse struct {
	Message llm.Message `json:"message"`
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
		return llm.ChatResponse{}, err
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.BaseURL+"/api/chat",
		bytes.NewReader(body),
	)
	if err != nil {
		return llm.ChatResponse{}, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return llm.ChatResponse{}, fmt.Errorf(
			"ollama request failed: %w", err,
		)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return llm.ChatResponse{}, fmt.Errorf(
			"ollama returned HTTP %d", resp.StatusCode,
		)
	}

	var result chatResponse

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return llm.ChatResponse{}, err
	}

	return llm.ChatResponse{
		Message: result.Message,
	}, nil
}

type tagsResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

func (c *Client) Models(ctx context.Context) ([]string, error) {

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.BaseURL+"/api/tags",
		nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf(
			"ollama request failed: %w", err,
		)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"ollama returned HTTP %d", resp.StatusCode,
		)
	}

	var result tagsResponse

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	models := make([]string, 0, len(result.Models))

	for _, model := range result.Models {
		models = append(models, strings.TrimSpace(model.Name))
	}

	return models, nil
}
