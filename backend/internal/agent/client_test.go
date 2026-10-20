package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientSelectTool(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("path = %q, want /v1/chat/completions", r.URL.Path)
		}
		if authorization := r.Header.Get("Authorization"); authorization != "Bearer secret" {
			t.Errorf("Authorization = %q, want bearer token", authorization)
		}

		var payload struct {
			Model             string           `json:"model"`
			Messages          []chatMessage    `json:"messages"`
			Tools             []toolDefinition `json:"tools"`
			ToolChoice        string           `json:"tool_choice"`
			ParallelToolCalls *bool            `json:"parallel_tool_calls"`
			ResponseFormat    json.RawMessage  `json:"response_format"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if payload.Model != "test-model" || len(payload.Messages) != 2 || len(payload.Tools) != 7 {
			t.Errorf("payload = %+v, want model, two messages, and seven tools", payload)
		}
		if payload.ToolChoice != "required" {
			t.Errorf("tool_choice = %q, want required", payload.ToolChoice)
		}
		if payload.ParallelToolCalls == nil || *payload.ParallelToolCalls {
			t.Error("parallel_tool_calls must be explicitly false")
		}
		if len(payload.ResponseFormat) != 0 {
			t.Errorf("response_format = %s, want omitted for tool calling", payload.ResponseFormat)
		}
		for _, tool := range payload.Tools {
			if !tool.Function.Strict {
				t.Errorf("tool %q is not strict", tool.Function.Name)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices": [{
				"message": {
					"role": "assistant",
					"tool_calls": [{
						"id": "call_123",
						"type": "function",
						"function": {
							"name": "create_expense",
							"arguments": "{\"amount\":\"350\"}"
						}
					}]
				}
			}]
		}`))
	}))
	defer server.Close()

	client := newClient(server.URL+"/v1", "secret", "test-model", server.Client())
	result, err := client.SelectTool(context.Background(), "system", "кофе 350", expenseTools())
	if err != nil {
		t.Fatalf("SelectTool() returned an error: %v", err)
	}
	if result.Function.Name != toolCreateExpense || result.Function.Arguments != `{"amount":"350"}` {
		t.Errorf("result = %+v, want create_expense tool call", result)
	}
}

func TestClientRejectsResponseWithoutToolCall(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"hello"}}]}`))
	}))
	defer server.Close()

	client := newClient(server.URL+"/v1", "", "test-model", server.Client())
	if _, err := client.SelectTool(context.Background(), "system", "hello", expenseTools()); err == nil {
		t.Fatal("SelectTool() accepted a response without a tool call")
	}
}

func TestClientRejectsMultipleToolCalls(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices":[{
				"message":{
					"tool_calls":[
						{"id":"one","type":"function","function":{"name":"list_expenses","arguments":"{}"}},
						{"id":"two","type":"function","function":{"name":"delete_expense","arguments":"{}"}}
					]
				}
			}]
		}`))
	}))
	defer server.Close()

	client := newClient(server.URL+"/v1", "", "test-model", server.Client())
	if _, err := client.SelectTool(context.Background(), "system", "hello", expenseTools()); err == nil {
		t.Fatal("SelectTool() accepted more than one tool call")
	}
}
