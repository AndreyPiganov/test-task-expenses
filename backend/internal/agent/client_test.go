package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientComplete(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("path = %q, want /v1/chat/completions", r.URL.Path)
		}
		if authorization := r.Header.Get("Authorization"); authorization != "Bearer secret" {
			t.Errorf("Authorization = %q, want bearer token", authorization)
		}

		var payload struct {
			Model    string        `json:"model"`
			Messages []chatMessage `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if payload.Model != "test-model" || len(payload.Messages) != 2 {
			t.Errorf("payload = %+v, want model and two messages", payload)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices": [{
				"message": {
					"role": "assistant",
					"content": "{\"action\":\"create_expense\"}"
				}
			}]
		}`))
	}))
	defer server.Close()

	client := newClient(server.URL+"/v1", "secret", "test-model", server.Client())
	result, err := client.Complete(context.Background(), "system", "кофе 350")
	if err != nil {
		t.Fatalf("Complete() returned an error: %v", err)
	}
	if result != `{"action":"create_expense"}` {
		t.Errorf("result = %q, want JSON command", result)
	}
}
