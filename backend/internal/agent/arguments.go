package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	toolCreateExpense        = "create_expense"
	toolListExpenses         = "list_expenses"
	toolSummarizeExpenses    = "summarize_expenses"
	toolUpdateExpense        = "update_expense"
	toolDeleteExpense        = "delete_expense"
	toolRequestClarification = "request_clarification"
	toolRespondUnsupported   = "respond_unsupported"

	selectorLast   = "last"
	selectorSingle = "single"
	dateLayout     = "2006-01-02"
)

type createExpenseArguments struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
	Category string `json:"category"`
	SpentAt  string `json:"spent_at"`
	Comment  string `json:"comment"`
}

type expenseFilterArguments struct {
	Category     *string `json:"category"`
	DateFrom     *string `json:"date_from"`
	DateTo       *string `json:"date_to"`
	CommentQuery *string `json:"comment_query"`
}

type updateExpenseArguments struct {
	Selector     string  `json:"selector"`
	Category     *string `json:"category"`
	DateFrom     *string `json:"date_from"`
	DateTo       *string `json:"date_to"`
	CommentQuery *string `json:"comment_query"`
	NewAmount    *string `json:"new_amount"`
	NewCategory  *string `json:"new_category"`
	NewSpentAt   *string `json:"new_spent_at"`
	NewComment   *string `json:"new_comment"`
	ClearComment bool    `json:"clear_comment"`
}

func (a updateExpenseArguments) filter() expenseFilterArguments {
	return expenseFilterArguments{
		Category:     a.Category,
		DateFrom:     a.DateFrom,
		DateTo:       a.DateTo,
		CommentQuery: a.CommentQuery,
	}
}

type deleteExpenseArguments struct {
	Selector     string  `json:"selector"`
	Category     *string `json:"category"`
	DateFrom     *string `json:"date_from"`
	DateTo       *string `json:"date_to"`
	CommentQuery *string `json:"comment_query"`
}

func (a deleteExpenseArguments) filter() expenseFilterArguments {
	return expenseFilterArguments{
		Category:     a.Category,
		DateFrom:     a.DateFrom,
		DateTo:       a.DateTo,
		CommentQuery: a.CommentQuery,
	}
}

type messageArguments struct {
	Message string `json:"message"`
}

func decodeToolArguments[T any](raw string) (T, error) {
	var arguments T
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&arguments); err != nil {
		return arguments, fmt.Errorf("decode tool arguments: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return arguments, errors.New("tool returned more than one JSON value")
	}

	return arguments, nil
}

func optionalString(value *string) string {
	if value == nil {
		return ""
	}

	return strings.TrimSpace(*value)
}
