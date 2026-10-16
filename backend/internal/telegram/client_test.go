package telegram

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClientGetUpdates(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/botsecret/getUpdates" {
			t.Errorf("path = %q, want Telegram method path", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}

		var payload struct {
			Offset  int `json:"offset"`
			Timeout int `json:"timeout"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if payload.Offset != 42 || payload.Timeout != 30 {
			t.Errorf("payload = %+v, want offset 42 and timeout 30", payload)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"ok": true,
			"result": [{
				"update_id": 42,
				"message": {
					"message_id": 7,
					"from": {"id": 123, "is_bot": false, "first_name": "Андрей"},
					"chat": {"id": 123},
					"text": "кофе 350"
				}
			}]
		}`))
	}))
	defer server.Close()

	client := newClient(server.URL, "secret", server.Client())
	updates, err := client.GetUpdates(context.Background(), 42, 30*time.Second)
	if err != nil {
		t.Fatalf("GetUpdates() returned an error: %v", err)
	}
	if len(updates) != 1 || updates[0].Message.Text != "кофе 350" {
		t.Fatalf("updates = %+v, want one text update", updates)
	}
}

func TestClientDoesNotLeakTokenInNetworkError(t *testing.T) {
	t.Parallel()

	client := newClient("http://127.0.0.1:1", "very-secret-token", &http.Client{
		Timeout: 100 * time.Millisecond,
	})
	_, err := client.GetUpdates(context.Background(), 0, time.Second)
	if err == nil {
		t.Fatal("GetUpdates() returned nil error")
	}
	if strings.Contains(err.Error(), "very-secret-token") {
		t.Fatalf("error leaks bot token: %v", err)
	}
}
