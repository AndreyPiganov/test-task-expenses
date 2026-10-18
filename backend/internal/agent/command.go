package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	actionCreateExpense     = "create_expense"
	actionListExpenses      = "list_expenses"
	actionSummarizeExpenses = "summarize_expenses"
	actionUpdateExpense     = "update_expense"
	actionDeleteExpense     = "delete_expense"
	actionClarify           = "clarify"
	actionUnsupported       = "unsupported"
	selectorLast            = "last"
	selectorSingle          = "single"
	dateLayout              = "2006-01-02"
)

// parsedCommand is the structured command returned by the LLM.
// The selected action determines which fields are relevant.
type parsedCommand struct {
	Action       string `json:"action"`
	Amount       string `json:"amount"`
	Currency     string `json:"currency"`
	Category     string `json:"category"`
	SpentAt      string `json:"spent_at"`
	Comment      string `json:"comment"`
	DateFrom     string `json:"date_from"`
	DateTo       string `json:"date_to"`
	CommentQuery string `json:"comment_query"`
	Selector     string `json:"selector"`
	NewAmount    string `json:"new_amount"`
	NewCategory  string `json:"new_category"`
	NewSpentAt   string `json:"new_spent_at"`
	NewComment   string `json:"new_comment"`
	ClearComment bool   `json:"clear_comment"`
	Question     string `json:"question"`
}

func parseCommand(raw string) (parsedCommand, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var command parsedCommand
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&command); err != nil {
		return parsedCommand{}, fmt.Errorf("decode LLM command: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return parsedCommand{}, errors.New("LLM returned more than one JSON value")
	}
	if command.Action == "" {
		return parsedCommand{}, errors.New("LLM command has no action")
	}

	return command, nil
}
