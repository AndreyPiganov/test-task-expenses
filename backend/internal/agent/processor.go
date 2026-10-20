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
	SelectTool(context.Context, string, string, []toolDefinition) (toolCall, error)
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
	call, err := p.llm.SelectTool(ctx, systemPrompt(now), message, expenseTools())
	if err != nil {
		return "", fmt.Errorf("select expense tool: %w", err)
	}

	switch call.Function.Name {
	case toolCreateExpense:
		arguments, decodeErr := decodeToolArguments[createExpenseArguments](call.Function.Arguments)
		if decodeErr != nil {
			return "", decodeErr
		}
		return p.createExpense(ctx, userID, message, arguments)
	case toolListExpenses:
		arguments, decodeErr := decodeToolArguments[expenseFilterArguments](call.Function.Arguments)
		if decodeErr != nil {
			return "", decodeErr
		}
		return p.listExpenses(ctx, userID, arguments)
	case toolSummarizeExpenses:
		arguments, decodeErr := decodeToolArguments[expenseFilterArguments](call.Function.Arguments)
		if decodeErr != nil {
			return "", decodeErr
		}
		return p.summarizeExpenses(ctx, userID, arguments)
	case toolUpdateExpense:
		arguments, decodeErr := decodeToolArguments[updateExpenseArguments](call.Function.Arguments)
		if decodeErr != nil {
			return "", decodeErr
		}
		return p.updateExpense(ctx, userID, message, arguments)
	case toolDeleteExpense:
		arguments, decodeErr := decodeToolArguments[deleteExpenseArguments](call.Function.Arguments)
		if decodeErr != nil {
			return "", decodeErr
		}
		return p.deleteExpense(ctx, userID, message, arguments)
	case toolRequestClarification:
		arguments, decodeErr := decodeToolArguments[messageArguments](call.Function.Arguments)
		if decodeErr != nil {
			return "", decodeErr
		}
		if strings.TrimSpace(arguments.Message) == "" {
			return "Уточните сумму и назначение расхода, например: «кофе 350».", nil
		}
		return arguments.Message, nil
	case toolRespondUnsupported:
		arguments, decodeErr := decodeToolArguments[messageArguments](call.Function.Arguments)
		if decodeErr != nil {
			return "", decodeErr
		}
		if strings.TrimSpace(arguments.Message) == "" {
			return "Я помогаю только учитывать семейные расходы.", nil
		}
		return arguments.Message, nil
	default:
		return "", fmt.Errorf("unsupported LLM tool %q", call.Function.Name)
	}
}
