package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/AndreyPiganov/test-task-expenses/backend/internal/domain"
	"github.com/AndreyPiganov/test-task-expenses/backend/internal/service"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const (
	dateLayout     = "2006-01-02"
	maxRequestBody = 1 << 20
)

type createExpenseRequest struct {
	UserID   int64           `json:"user_id"`
	Amount   decimal.Decimal `json:"amount"`
	Category domain.Category `json:"category"`
	SpentAt  string          `json:"spent_at"`
	Comment  string          `json:"comment"`
}

type updateExpenseRequest struct {
	UserID   *int64           `json:"user_id"`
	Amount   *decimal.Decimal `json:"amount"`
	Category *domain.Category `json:"category"`
	SpentAt  *string          `json:"spent_at"`
	Comment  *string          `json:"comment"`
}

type expenseResponse struct {
	ID        uuid.UUID       `json:"id"`
	UserID    int64           `json:"user_id"`
	Amount    decimal.Decimal `json:"amount"`
	Category  domain.Category `json:"category"`
	SpentAt   string          `json:"spent_at"`
	Comment   string          `json:"comment,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type expensePageResponse struct {
	Items  []expenseResponse `json:"items"`
	Total  int               `json:"total"`
	Limit  int               `json:"limit"`
	Offset int               `json:"offset"`
}

func (a *api) expensesCollection(w http.ResponseWriter, r *http.Request) {
	if a.expenses == nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "service unavailable"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		a.listExpenses(w, r)
	case http.MethodPost:
		a.createExpense(w, r)
	default:
		methodNotAllowed(w, "GET, POST")
	}
}

func (a *api) expenseItem(w http.ResponseWriter, r *http.Request) {
	if a.expenses == nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "service unavailable"})
		return
	}

	idText := strings.TrimPrefix(r.URL.Path, "/api/v1/expenses/")
	if idText == "" || strings.Contains(idText, "/") {
		notFound(w, r)
		return
	}

	id, err := uuid.Parse(idText)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid expense ID"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		a.getExpense(w, r, id)
	case http.MethodPatch:
		a.updateExpense(w, r, id)
	case http.MethodDelete:
		a.deleteExpense(w, r, id)
	default:
		methodNotAllowed(w, "GET, PATCH, DELETE")
	}
}

func (a *api) createExpense(w http.ResponseWriter, r *http.Request) {
	var request createExpenseRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	spentAt, err := time.Parse(dateLayout, request.SpentAt)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, errorResponse{Error: "spent_at must use YYYY-MM-DD"})
		return
	}

	expense, err := a.expenses.Create(r.Context(), service.CreateExpenseInput{
		UserID:   request.UserID,
		Amount:   request.Amount,
		Category: request.Category,
		SpentAt:  spentAt,
		Comment:  request.Comment,
	})
	if err != nil {
		a.writeServiceError(w, r, err)
		return
	}

	w.Header().Set("Location", "/api/v1/expenses/"+expense.ID.String())
	writeJSON(w, http.StatusCreated, toExpenseResponse(expense))
}

func (a *api) getExpense(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	expense, err := a.expenses.Get(r.Context(), id)
	if err != nil {
		a.writeServiceError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, toExpenseResponse(expense))
}

func (a *api) listExpenses(w http.ResponseWriter, r *http.Request) {
	filter, err := parseExpenseFilter(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	page, err := a.expenses.List(r.Context(), filter)
	if err != nil {
		a.writeServiceError(w, r, err)
		return
	}

	items := make([]expenseResponse, 0, len(page.Items))
	for _, expense := range page.Items {
		items = append(items, toExpenseResponse(expense))
	}

	writeJSON(w, http.StatusOK, expensePageResponse{
		Items:  items,
		Total:  page.Total,
		Limit:  page.Limit,
		Offset: page.Offset,
	})
}

func (a *api) updateExpense(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	var request updateExpenseRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	input := service.UpdateExpenseInput{
		UserID:   request.UserID,
		Amount:   request.Amount,
		Category: request.Category,
		Comment:  request.Comment,
	}
	if request.SpentAt != nil {
		spentAt, err := time.Parse(dateLayout, *request.SpentAt)
		if err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, errorResponse{Error: "spent_at must use YYYY-MM-DD"})
			return
		}
		input.SpentAt = &spentAt
	}

	expense, err := a.expenses.Update(r.Context(), id, input)
	if err != nil {
		a.writeServiceError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, toExpenseResponse(expense))
}

func (a *api) deleteExpense(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	if err := a.expenses.Delete(r.Context(), id); err != nil {
		a.writeServiceError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func parseExpenseFilter(r *http.Request) (service.ExpenseFilter, error) {
	query := r.URL.Query()
	var filter service.ExpenseFilter

	if value := query.Get("user_id"); value != "" {
		userID, err := strconv.ParseInt(value, 10, 64)
		if err != nil || userID <= 0 {
			return filter, errors.New("user_id must be a positive integer")
		}
		filter.UserID = &userID
	}
	if value := query.Get("category"); value != "" {
		category := domain.Category(value)
		if !category.Valid() {
			return filter, errors.New("unknown category")
		}
		filter.Category = &category
	}
	if value := query.Get("date_from"); value != "" {
		dateFrom, err := time.Parse(dateLayout, value)
		if err != nil {
			return filter, errors.New("date_from must use YYYY-MM-DD")
		}
		filter.DateFrom = &dateFrom
	}
	if value := query.Get("date_to"); value != "" {
		dateTo, err := time.Parse(dateLayout, value)
		if err != nil {
			return filter, errors.New("date_to must use YYYY-MM-DD")
		}
		filter.DateTo = &dateTo
	}
	if value := query.Get("limit"); value != "" {
		limit, err := strconv.Atoi(value)
		if err != nil || limit <= 0 {
			return filter, errors.New("limit must be a positive integer")
		}
		filter.Limit = limit
	}
	if value := query.Get("offset"); value != "" {
		offset, err := strconv.Atoi(value)
		if err != nil || offset < 0 {
			return filter, errors.New("offset must be a non-negative integer")
		}
		filter.Offset = offset
	}

	return filter, nil
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("request body must contain one JSON object")
	}

	return nil
}

func toExpenseResponse(expense domain.Expense) expenseResponse {
	return expenseResponse{
		ID:        expense.ID,
		UserID:    expense.UserID,
		Amount:    expense.Amount,
		Category:  expense.Category,
		SpentAt:   expense.SpentAt.Format(dateLayout),
		Comment:   expense.Comment,
		CreatedAt: expense.CreatedAt,
		UpdatedAt: expense.UpdatedAt,
	}
}

func (a *api) writeServiceError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		writeJSON(w, http.StatusUnprocessableEntity, errorResponse{Error: err.Error()})
	case errors.Is(err, service.ErrExpenseNotFound), errors.Is(err, service.ErrMemberNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: err.Error()})
	default:
		a.logger.ErrorContext(r.Context(), "request failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
	}
}
