package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/AndreyPiganov/test-task-expenses/backend/internal/currency"
	"github.com/AndreyPiganov/test-task-expenses/backend/internal/domain"
	"github.com/AndreyPiganov/test-task-expenses/backend/internal/service"
	"github.com/google/uuid"
)

type LLM interface {
	Complete(context.Context, string, string) (string, error)
}

type ExpenseService interface {
	Create(context.Context, service.CreateExpenseInput) (domain.Expense, error)
	List(context.Context, service.ExpenseFilter) (service.ExpensePage, error)
	UpdateForUser(context.Context, int64, uuid.UUID, service.UpdateExpenseInput) (domain.Expense, error)
	DeleteForUser(context.Context, int64, uuid.UUID) error
}

// Processor translates natural-language messages into validated expense operations.
type Processor struct {
	llm               LLM
	expenses          ExpenseService
	currencyConverter currency.Converter
	timezone          *time.Location
	now               func() time.Time
}

func NewProcessor(
	llm LLM,
	expenses ExpenseService,
	timezone *time.Location,
	currencyConverters ...currency.Converter,
) *Processor {
	processor := &Processor{
		llm:      llm,
		expenses: expenses,
		timezone: timezone,
		now:      time.Now,
	}
	if len(currencyConverters) > 0 {
		processor.currencyConverter = currencyConverters[0]
	}
	return processor
}

func (p *Processor) Process(ctx context.Context, userID int64, message string) (string, error) {
	now := p.now().In(p.timezone)
	raw, err := p.llm.Complete(ctx, systemPrompt(now), message)
	if err != nil {
		return "", fmt.Errorf("complete expense command: %w", err)
	}

	command, err := parseCommand(raw)
	if err != nil {
		return "", err
	}

	switch command.Action {
	case actionCreateExpense:
		return p.createExpense(ctx, userID, message, command)
	case actionListExpenses:
		return p.listExpenses(ctx, userID, command)
	case actionSummarizeExpenses:
		return p.summarizeExpenses(ctx, userID, command)
	case actionUpdateExpense:
		return p.updateExpense(ctx, userID, message, command)
	case actionDeleteExpense:
		return p.deleteExpense(ctx, userID, message, command)
	case actionClarify, actionUnsupported:
		if strings.TrimSpace(command.Question) == "" {
			return "Уточните сумму и назначение расхода, например: «кофе 350».", nil
		}
		return command.Question, nil
	default:
		return "", fmt.Errorf("unsupported LLM action %q", command.Action)
	}
}
