package telegram

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

const (
	defaultAPIBaseURL = "https://api.telegram.org"
	maxResponseSize   = 1 << 20
)

type Client struct {
	endpoint   string
	httpClient *http.Client
}

func NewClient(token string) *Client {
	return newClient(defaultAPIBaseURL, token, &http.Client{Timeout: 40 * time.Second})
}

func newClient(baseURL, token string, httpClient *http.Client) *Client {
	return &Client{
		endpoint:   strings.TrimRight(baseURL, "/") + "/bot" + token,
		httpClient: httpClient,
	}
}

func (c *Client) GetUpdates(
	ctx context.Context,
	offset int,
	timeout time.Duration,
) ([]Update, error) {
	payload := struct {
		Offset         int      `json:"offset"`
		Timeout        int      `json:"timeout"`
		AllowedUpdates []string `json:"allowed_updates"`
	}{
		Offset:         offset,
		Timeout:        int(timeout.Seconds()),
		AllowedUpdates: []string{"message"},
	}

	var updates []Update
	if err := c.call(ctx, "getUpdates", payload, &updates); err != nil {
		return nil, err
	}

	return updates, nil
}

func (c *Client) SendMessage(ctx context.Context, chatID int64, text string) error {
	payload := struct {
		ChatID int64  `json:"chat_id"`
		Text   string `json:"text"`
	}{ChatID: chatID, Text: text}

	return c.call(ctx, "sendMessage", payload, nil)
}

func (c *Client) call(ctx context.Context, method string, payload, result any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode Telegram %s request: %w", method, err)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.endpoint+"/"+method,
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("create Telegram %s request: %w", method, err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return sanitizedRequestError(method, err)
	}
	defer response.Body.Close()

	var envelope struct {
		OK          bool            `json:"ok"`
		Result      json.RawMessage `json:"result"`
		ErrorCode   int             `json:"error_code"`
		Description string          `json:"description"`
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, maxResponseSize))
	if err := decoder.Decode(&envelope); err != nil {
		return fmt.Errorf("decode Telegram %s response: %w", method, err)
	}
	if response.StatusCode != http.StatusOK || !envelope.OK {
		return fmt.Errorf(
			"Telegram %s failed with code %d: %s",
			method,
			envelope.ErrorCode,
			envelope.Description,
		)
	}
	if result != nil {
		if err := json.Unmarshal(envelope.Result, result); err != nil {
			return fmt.Errorf("decode Telegram %s result: %w", method, err)
		}
	}

	return nil
}

func sanitizedRequestError(method string, err error) error {
	var urlError *url.Error
	if errors.As(err, &urlError) {
		return fmt.Errorf("Telegram %s request failed: %w", method, urlError.Err)
	}

	return fmt.Errorf("Telegram %s request failed", method)
}
