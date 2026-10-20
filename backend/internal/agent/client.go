package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxLLMResponseSize = 2 << 20

type Client struct {
	endpoint   string
	apiKey     string
	model      string
	httpClient *http.Client
}

type toolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function functionToolCall `json:"function"`
}

type functionToolCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

func NewClient(baseURL, apiKey, model string) *Client {
	return newClient(baseURL, apiKey, model, &http.Client{Timeout: 45 * time.Second})
}

func newClient(baseURL, apiKey, model string, httpClient *http.Client) *Client {
	return &Client{
		endpoint:   strings.TrimRight(baseURL, "/") + "/chat/completions",
		apiKey:     apiKey,
		model:      model,
		httpClient: httpClient,
	}
}

func (c *Client) SelectTool(
	ctx context.Context,
	systemPrompt string,
	userMessage string,
	tools []toolDefinition,
) (toolCall, error) {
	payload := struct {
		Model             string           `json:"model"`
		Messages          []chatMessage    `json:"messages"`
		Temperature       float64          `json:"temperature"`
		Tools             []toolDefinition `json:"tools"`
		ToolChoice        string           `json:"tool_choice"`
		ParallelToolCalls bool             `json:"parallel_tool_calls"`
	}{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userMessage},
		},
		Temperature:       0,
		Tools:             tools,
		ToolChoice:        "required",
		ParallelToolCalls: false,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return toolCall{}, fmt.Errorf("encode LLM request: %w", err)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.endpoint,
		bytes.NewReader(body),
	)
	if err != nil {
		return toolCall{}, fmt.Errorf("create LLM request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		request.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return toolCall{}, sanitizedLLMRequestError(err)
	}
	defer response.Body.Close()

	var result struct {
		Choices []struct {
			Message struct {
				ToolCalls []toolCall `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, maxLLMResponseSize)).Decode(&result); err != nil {
		return toolCall{}, fmt.Errorf("decode LLM response: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		message := http.StatusText(response.StatusCode)
		if result.Error != nil && result.Error.Message != "" {
			message = result.Error.Message
		}
		return toolCall{}, fmt.Errorf("LLM returned status %d: %s", response.StatusCode, message)
	}
	if len(result.Choices) == 0 {
		return toolCall{}, errors.New("LLM returned no choices")
	}
	calls := result.Choices[0].Message.ToolCalls
	if len(calls) != 1 {
		return toolCall{}, fmt.Errorf("LLM returned %d tool calls, want exactly one", len(calls))
	}
	call := calls[0]
	if call.Type != "function" || strings.TrimSpace(call.Function.Name) == "" ||
		strings.TrimSpace(call.Function.Arguments) == "" {
		return toolCall{}, errors.New("LLM returned an invalid function tool call")
	}

	return call, nil
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func sanitizedLLMRequestError(err error) error {
	var urlError *url.Error
	if errors.As(err, &urlError) {
		return fmt.Errorf("LLM request failed: %w", urlError.Err)
	}

	return errors.New("LLM request failed")
}
