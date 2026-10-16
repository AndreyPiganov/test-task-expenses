package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/AndreyPiganov/test-task-expenses/backend/internal/domain"
	"github.com/AndreyPiganov/test-task-expenses/backend/internal/service"
	"github.com/google/uuid"
)

type statusResponse struct {
	Status string `json:"status"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type ReadinessCheck func(context.Context) error

type ExpenseService interface {
	Create(context.Context, service.CreateExpenseInput) (domain.Expense, error)
	Get(context.Context, uuid.UUID) (domain.Expense, error)
	List(context.Context, service.ExpenseFilter) (service.ExpensePage, error)
	Update(context.Context, uuid.UUID, service.UpdateExpenseInput) (domain.Expense, error)
	Delete(context.Context, uuid.UUID) error
}

type FamilyMemberService interface {
	List(context.Context) ([]domain.FamilyMember, error)
}

type api struct {
	logger   *slog.Logger
	expenses ExpenseService
	members  FamilyMemberService
}

func NewHandler(
	logger *slog.Logger,
	readinessCheck ReadinessCheck,
	expenses ExpenseService,
	members FamilyMemberService,
) http.Handler {
	api := &api{logger: logger, expenses: expenses, members: members}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", health)
	mux.HandleFunc("/ready", ready(readinessCheck))
	mux.HandleFunc("/api/v1/expenses", api.expensesCollection)
	mux.HandleFunc("/api/v1/expenses/", api.expenseItem)
	mux.HandleFunc("/api/v1/members", api.familyMembers)
	mux.HandleFunc("/", notFound)

	return recoverPanic(logger, logRequest(logger, mux))
}

func health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	writeJSON(w, http.StatusOK, statusResponse{Status: "ok"})
}

func ready(check ReadinessCheck) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}

		if check != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()

			if err := check(ctx); err != nil {
				writeJSON(w, http.StatusServiceUnavailable, statusResponse{Status: "unavailable"})
				return
			}
		}

		writeJSON(w, http.StatusOK, statusResponse{Status: "ready"})
	}
}

func notFound(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusNotFound, errorResponse{Error: "not found"})
}

func methodNotAllowed(w http.ResponseWriter, allowedMethod string) {
	w.Header().Set("Allow", allowedMethod)
	writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(payload)
}

func logRequest(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		response := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(response, r)

		logger.InfoContext(
			r.Context(),
			"HTTP request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", response.status,
			"duration", time.Since(startedAt),
		)
	})
}

func recoverPanic(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.ErrorContext(
					r.Context(),
					"panic recovered",
					"error", recovered,
					"stack", string(debug.Stack()),
				)
				writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
			}
		}()

		next.ServeHTTP(w, r)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
