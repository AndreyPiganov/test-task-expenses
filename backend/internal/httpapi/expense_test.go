package httpapi

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/AndreyPiganov/test-task-expenses/backend/internal/domain"
	"github.com/AndreyPiganov/test-task-expenses/backend/internal/service"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestCreateExpenseEndpoint(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	stub := &stubExpenseService{created: domain.Expense{
		ID:       id,
		UserID:   123,
		Amount:   decimal.RequireFromString("350.50"),
		Category: domain.CategoryFood,
		SpentAt:  time.Date(2026, time.October, 16, 0, 0, 0, 0, time.UTC),
		Comment:  "кофе",
	}}
	handler := newTestHandler(stub)
	body := bytes.NewBufferString(`{
		"user_id": 123,
		"amount": "350.50",
		"category": "еда",
		"spent_at": "2026-10-16",
		"comment": "кофе"
	}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/expenses", body)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusCreated, response.Body)
	}
	if location := response.Header().Get("Location"); location != "/api/v1/expenses/"+id.String() {
		t.Errorf("Location = %q, want expense URL", location)
	}
	if stub.createInput.UserID != 123 || stub.createInput.Comment != "кофе" {
		t.Errorf("Create input = %+v, want request data", stub.createInput)
	}
	if !strings.Contains(response.Body.String(), `"spent_at":"2026-10-16"`) {
		t.Errorf("body = %s, want formatted spent_at", response.Body)
	}
}

func TestCreateExpenseEndpointRejectsUnknownField(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(&stubExpenseService{})
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/expenses",
		strings.NewReader(`{"user_id":123,"unknown":true}`),
	)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestListExpensesEndpointRejectsInvalidFilter(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(&stubExpenseService{})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/expenses?limit=101", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d; body = %s", response.Code, http.StatusUnprocessableEntity, response.Body)
	}
}

func newTestHandler(expenses ExpenseService) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewHandler(logger, nil, expenses, nil)
}

type stubExpenseService struct {
	createInput service.CreateExpenseInput
	created     domain.Expense
}

func (s *stubExpenseService) Create(
	_ context.Context,
	input service.CreateExpenseInput,
) (domain.Expense, error) {
	s.createInput = input
	return s.created, nil
}

func (s *stubExpenseService) Get(context.Context, uuid.UUID) (domain.Expense, error) {
	return domain.Expense{}, nil
}

func (s *stubExpenseService) List(
	_ context.Context,
	filter service.ExpenseFilter,
) (service.ExpensePage, error) {
	if filter.Limit > 100 {
		return service.ExpensePage{}, service.ErrInvalidInput
	}
	return service.ExpensePage{}, nil
}

func (s *stubExpenseService) Update(
	context.Context,
	uuid.UUID,
	service.UpdateExpenseInput,
) (domain.Expense, error) {
	return domain.Expense{}, nil
}

func (s *stubExpenseService) Delete(context.Context, uuid.UUID) error {
	return nil
}
