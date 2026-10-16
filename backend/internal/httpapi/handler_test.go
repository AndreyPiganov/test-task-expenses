package httpapi

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantBody   string
		wantAllow  string
	}{
		{
			name:       "health",
			method:     http.MethodGet,
			path:       "/health",
			wantStatus: http.StatusOK,
			wantBody:   "{\"status\":\"ok\"}\n",
		},
		{
			name:       "ready",
			method:     http.MethodGet,
			path:       "/ready",
			wantStatus: http.StatusOK,
			wantBody:   "{\"status\":\"ready\"}\n",
		},
		{
			name:       "method not allowed",
			method:     http.MethodPost,
			path:       "/health",
			wantStatus: http.StatusMethodNotAllowed,
			wantBody:   "{\"error\":\"method not allowed\"}\n",
			wantAllow:  http.MethodGet,
		},
		{
			name:       "not found",
			method:     http.MethodGet,
			path:       "/unknown",
			wantStatus: http.StatusNotFound,
			wantBody:   "{\"error\":\"not found\"}\n",
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := NewHandler(logger, nil, nil, nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequest(tt.method, tt.path, nil)
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", response.Code, tt.wantStatus)
			}
			if response.Body.String() != tt.wantBody {
				t.Errorf("body = %q, want %q", response.Body.String(), tt.wantBody)
			}
			if allow := response.Header().Get("Allow"); allow != tt.wantAllow {
				t.Errorf("Allow header = %q, want %q", allow, tt.wantAllow)
			}
			if contentType := response.Header().Get("Content-Type"); contentType != "application/json; charset=utf-8" {
				t.Errorf("Content-Type = %q, want application/json", contentType)
			}
		})
	}
}

func TestReadyReturnsUnavailableWhenDependencyIsDown(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := NewHandler(logger, func(context.Context) error {
		return errors.New("database is unavailable")
	}, nil, nil)
	request := httptest.NewRequest(http.MethodGet, "/ready", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
	if response.Body.String() != "{\"status\":\"unavailable\"}\n" {
		t.Errorf("body = %q, want unavailable status", response.Body.String())
	}
}
