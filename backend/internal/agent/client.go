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

func (c *Client) Complete(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	payload := struct {
		Model          string         `json:"model"`
		Messages       []chatMessage  `json:"messages"`
		Temperature    float64        `json:"temperature"`
		ResponseFormat map[string]any `json:"response_format"`
	}{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userMessage},
		},
		Temperature:    0,
		ResponseFormat: map[string]any{"type": "json_object"},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode LLM request: %w", err)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.endpoint,
		bytes.NewReader(body),
	)
	if err != nil {
		return "", fmt.Errorf("create LLM request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		request.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return "", sanitizedLLMRequestError(err)
	}
	defer response.Body.Close()

	var result struct {
		Choices []struct {
			Message chatMessage `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, maxLLMResponseSize)).Decode(&result); err != nil {
		return "", fmt.Errorf("decode LLM response: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		message := http.StatusText(response.StatusCode)
		if result.Error != nil && result.Error.Message != "" {
			message = result.Error.Message
		}
		return "", fmt.Errorf("LLM returned status %d: %s", response.StatusCode, message)
	}
	if len(result.Choices) == 0 || strings.TrimSpace(result.Choices[0].Message.Content) == "" {
		return "", errors.New("LLM returned an empty response")
	}

	return result.Choices[0].Message.Content, nil
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
