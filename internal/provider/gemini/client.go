package gemini

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Rugved7/termai/internal/llm"
)

// interactionsModels contains text-generation models documented
// as compatible with Google's Interactions API.
var interactionsModels = map[string]bool{
	"gemini-2.5-flash":       true,
	"gemini-2.5-pro":         true,
	"gemini-2.5-flash-lite":  true,
	"gemini-3-flash-preview": true,
	"gemini-3.1-pro-preview": true,
	"gemini-3.1-flash-lite":  true,
	"gemini-3.5-flash":       true,
	"gemini-3.6-flash":       true,
	"gemini-3.7-flash":       true,
	"gemini-3.8-flash":       true,
}

const (
	defaultBaseURL = "https://generativelanguage.googleapis.com/v1"
	maxErrorBody   = 64 * 1024
)

type Client struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient(apiKey string) (*Client, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("Gemini API key is required")
	}

	return &Client{
		APIKey:     apiKey,
		BaseURL:    defaultBaseURL,
		HTTPClient: &http.Client{},
	}, nil
}

// type interactionRequest struct {
// 	Model  string `json:"model"`
// 	Input  string `json:"input"`
// 	Store  bool   `json:"store"`
// 	Stream bool   `json:"stream,omitempty"`
// }

type interactionContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type interactionStep struct {
	Type    string               `json:"type"`
	Content []interactionContent `json:"content"`
}

type interactionRequest struct {
	Model             string            `json:"model"`
	Input             []interactionStep `json:"input"`
	SystemInstruction string            `json:"system_instruction,omitempty"`
	Store             bool              `json:"store"`
	Stream            bool              `json:"stream,omitempty"`
}

type interactionResponse struct {
	Steps []struct {
		Type    string `json:"type"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"steps"`
}

type modelsResponse struct {
	Models []struct {
		Name                       string   `json:"name"`
		SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
	} `json:"models"`

	NextPageToken string `json:"nextPageToken"`
}

type streamEvent struct {
	EventType string `json:"event_type"`
	Delta     struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"delta"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

type apiErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

func (c *Client) Chat(
	ctx context.Context,
	req llm.ChatRequest,
) (llm.ChatResponse, error) {
	if strings.TrimSpace(req.Model) == "" {
		return llm.ChatResponse{}, fmt.Errorf("Gemini model is required")
	}

	var steps []interactionStep
	var systemInstruction strings.Builder

	for _, message := range req.Messages {
		content := strings.TrimSpace(message.Content)
		if content == "" {
			continue
		}

		switch message.Role {
		case "system":
			systemInstruction.WriteString(content)
			systemInstruction.WriteString("\n")

		case "user":
			steps = append(steps, interactionStep{
				Type: "user_input",
				Content: []interactionContent{
					{
						Type: "text",
						Text: content,
					},
				},
			})

		case "assistant":
			steps = append(steps, interactionStep{
				Type: "model_output",
				Content: []interactionContent{
					{
						Type: "text",
						Text: content,
					},
				},
			})

		default:
			return llm.ChatResponse{}, fmt.Errorf(
				"unsupported message role %q", message.Role,
			)
		}
	}

	if len(steps) == 0 {
		return llm.ChatResponse{}, fmt.Errorf(
			"Gemini request requires at least one message",
		)
	}

	body := interactionRequest{
		Model:             req.Model,
		Input:             steps,
		SystemInstruction: strings.TrimSpace(systemInstruction.String()),
		Store:             false,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return llm.ChatResponse{}, fmt.Errorf(
			"encode Gemini request: %w", err,
		)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.BaseURL+"/interactions",
		bytes.NewReader(payload),
	)
	if err != nil {
		return llm.ChatResponse{}, fmt.Errorf(
			"create Gemini request: %w", err,
		)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", c.APIKey)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return llm.ChatResponse{}, fmt.Errorf(
			"send Gemini request: %w", err,
		)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(
			io.LimitReader(resp.Body, maxErrorBody),
		)

		var apiErr apiErrorResponse
		if json.Unmarshal(data, &apiErr) == nil &&
			apiErr.Error.Message != "" {
			return llm.ChatResponse{}, fmt.Errorf(
				"Gemini API error (%d): %s",
				resp.StatusCode,
				apiErr.Error.Message,
			)
		}

		return llm.ChatResponse{}, fmt.Errorf(
			"Gemini API returned status %d: %s",
			resp.StatusCode,
			strings.TrimSpace(string(data)),
		)
	}

	var result interactionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return llm.ChatResponse{}, fmt.Errorf(
			"decode Gemini response: %w", err,
		)
	}

	var answer strings.Builder

	for _, step := range result.Steps {
		if step.Type != "model_output" {
			continue
		}

		for _, content := range step.Content {
			if content.Type == "text" {
				answer.WriteString(content.Text)
			}
		}
	}

	if answer.Len() == 0 {
		return llm.ChatResponse{}, fmt.Errorf(
			"Gemini returned no text output",
		)
	}

	return llm.ChatResponse{
		Message: llm.Message{
			Role:    "assistant",
			Content: answer.String(),
		},
	}, nil
}

func (c *Client) Models(ctx context.Context) ([]string, error) {
	const maxPages = 100

	var models []string
	pageToken := ""

	for page := 0; page < maxPages; page++ {
		endpoint := c.BaseURL + "/models"

		httpReq, err := http.NewRequestWithContext(
			ctx,
			http.MethodGet,
			endpoint,
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("create Gemini models request: %w", err)
		}

		query := httpReq.URL.Query()
		query.Set("pageSize", "1000")

		if pageToken != "" {
			query.Set("pageToken", pageToken)
		}

		httpReq.URL.RawQuery = query.Encode()
		httpReq.Header.Set("x-goog-api-key", c.APIKey)

		resp, err := c.HTTPClient.Do(httpReq)
		if err != nil {
			return nil, fmt.Errorf("fetch Gemini models: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			data, _ := io.ReadAll(
				io.LimitReader(resp.Body, maxErrorBody),
			)
			resp.Body.Close()

			var apiErr apiErrorResponse
			if json.Unmarshal(data, &apiErr) == nil &&
				apiErr.Error.Message != "" {
				return nil, fmt.Errorf(
					"Gemini API error (%d): %s",
					resp.StatusCode,
					apiErr.Error.Message,
				)
			}

			return nil, fmt.Errorf(
				"Gemini API returned status %d: %s",
				resp.StatusCode,
				strings.TrimSpace(string(data)),
			)
		}

		var result modelsResponse

		err = json.NewDecoder(
			io.LimitReader(resp.Body, 10*1024*1024),
		).Decode(&result)

		resp.Body.Close()

		if err != nil {
			return nil, fmt.Errorf(
				"decode Gemini models response: %w", err,
			)
		}

		for _, model := range result.Models {
			supportsGeneration := false

			for _, method := range model.SupportedGenerationMethods {
				if method == "generateContent" {
					supportsGeneration = true
					break
				}
			}
			if supportsGeneration {
				name := strings.TrimPrefix(model.Name, "models/")

				if interactionsModels[name] {
					models = append(models, name)
				}
			}
		}

		if result.NextPageToken == "" {
			return models, nil
		}

		if result.NextPageToken == pageToken {
			return nil, fmt.Errorf(
				"Gemini models API returned a repeated page token",
			)
		}

		pageToken = result.NextPageToken
	}

	return nil, fmt.Errorf(
		"Gemini models API exceeded maximum pagination limit",
	)
}

func (c *Client) ChatStream(
	ctx context.Context,
	req llm.ChatRequest,
	onChunk func(string),
) (llm.ChatResponse, error) {
	if strings.TrimSpace(req.Model) == "" {
		return llm.ChatResponse{}, fmt.Errorf("Gemini model is required")
	}

	var steps []interactionStep
	var systemInstruction strings.Builder

	for _, message := range req.Messages {
		content := strings.TrimSpace(message.Content)
		if content == "" {
			continue
		}

		switch message.Role {
		case "system":
			systemInstruction.WriteString(content)
			systemInstruction.WriteString("\n")

		case "user":
			steps = append(steps, interactionStep{
				Type: "user_input",
				Content: []interactionContent{
					{Type: "text", Text: content},
				},
			})

		case "assistant":
			steps = append(steps, interactionStep{
				Type: "model_output",
				Content: []interactionContent{
					{Type: "text", Text: content},
				},
			})

		default:
			return llm.ChatResponse{}, fmt.Errorf(
				"unsupported message role %q", message.Role,
			)
		}
	}

	if len(steps) == 0 {
		return llm.ChatResponse{}, fmt.Errorf(
			"Gemini request requires at least one message",
		)
	}

	body := interactionRequest{
		Model:             req.Model,
		Input:             steps,
		SystemInstruction: strings.TrimSpace(systemInstruction.String()),
		Store:             false,
		Stream:            true,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return llm.ChatResponse{}, fmt.Errorf(
			"encode Gemini request: %w", err,
		)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.BaseURL+"/interactions",
		bytes.NewReader(payload),
	)
	if err != nil {
		return llm.ChatResponse{}, fmt.Errorf(
			"create Gemini request: %w", err,
		)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("x-goog-api-key", c.APIKey)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return llm.ChatResponse{}, fmt.Errorf(
			"send Gemini streaming request: %w", err,
		)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(
			io.LimitReader(resp.Body, maxErrorBody),
		)

		var apiErr apiErrorResponse
		if json.Unmarshal(data, &apiErr) == nil &&
			apiErr.Error.Message != "" {
			return llm.ChatResponse{}, fmt.Errorf(
				"Gemini API error (%d): %s",
				resp.StatusCode,
				apiErr.Error.Message,
			)
		}

		return llm.ChatResponse{}, fmt.Errorf(
			"Gemini API returned status %d: %s",
			resp.StatusCode,
			strings.TrimSpace(string(data)),
		)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 4096), 1024*1024)

	var answer strings.Builder
	var eventData strings.Builder
	var eventName string
	completed := false

	processEvent := func() error {
		data := strings.TrimSpace(eventData.String())
		eventData.Reset()

		if data == "" {
			eventName = ""
			return nil
		}

		if data == "[DONE]" {
			completed = true
			eventName = ""
			return nil
		}

		var event streamEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			return fmt.Errorf("decode Gemini stream event: %w", err)
		}

		if event.EventType == "" {
			event.EventType = eventName
		}

		eventName = ""

		switch event.EventType {
		case "step.delta":
			if event.Delta.Type == "text" && event.Delta.Text != "" {
				answer.WriteString(event.Delta.Text)

				if onChunk != nil {
					onChunk(event.Delta.Text)
				}
			}

		case "interaction.completed":
			completed = true

		case "interaction.failed":
			return fmt.Errorf("Gemini interaction failed")

		case "error":
			if event.Error.Message != "" {
				return fmt.Errorf("Gemini stream error: %s", event.Error.Message)
			}
			return fmt.Errorf("Gemini stream returned an error event")
		}

		return nil
	}

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			if err := processEvent(); err != nil {
				return llm.ChatResponse{}, err
			}
			continue
		}

		if strings.HasPrefix(line, "event:") {
			eventName = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			if eventData.Len() > 0 {
				eventData.WriteString("\n")
			}
			eventData.WriteString(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}

	if err := scanner.Err(); err != nil {
		if ctx.Err() != nil {
			return llm.ChatResponse{}, ctx.Err()
		}
		return llm.ChatResponse{}, fmt.Errorf(
			"read Gemini stream: %w", err,
		)
	}

	if err := processEvent(); err != nil {
		return llm.ChatResponse{}, err
	}

	if !completed {
		return llm.ChatResponse{}, errors.New(
			"Gemini stream ended before interaction completed",
		)
	}

	if answer.Len() == 0 {
		return llm.ChatResponse{}, fmt.Errorf(
			"Gemini returned no text output",
		)
	}

	return llm.ChatResponse{
		Message: llm.Message{
			Role:    "assistant",
			Content: answer.String(),
		},
	}, nil
}
