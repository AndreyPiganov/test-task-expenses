package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/AndreyPiganov/test-task-expenses/backend/internal/domain"
	"github.com/AndreyPiganov/test-task-expenses/backend/internal/service"
	"github.com/shopspring/decimal"
)

func (p *Processor) updateExpense(
	ctx context.Context,
	userID int64,
	message string,
	arguments updateExpenseArguments,
) (string, error) {
	expense, clarification, err := p.selectExpense(
		ctx,
		userID,
		message,
		arguments.Selector,
		arguments.filter(),
	)
	if err != nil || clarification != "" {
		return clarification, err
	}

	input, hasChanges, err := p.updateInput(arguments)
	if err != nil {
		return "", err
	}
	if !hasChanges {
		return "Уточните, что именно нужно изменить: сумму, категорию, дату или комментарий.", nil
	}

	updated, err := p.expenses.UpdateForUser(ctx, userID, expense.ID, input)
	if err != nil {
		return "", fmt.Errorf("update Telegram user expense: %w", err)
	}

	return "Исправил: " + formatExpenseLine(updated) + ".", nil
}

func (p *Processor) deleteExpense(
	ctx context.Context,
	userID int64,
	message string,
	arguments deleteExpenseArguments,
) (string, error) {
	if !hasDeleteIntent(message) {
		return "Чтобы удалить расход, явно напишите «удали» и уточните, какой именно.", nil
	}

	expense, clarification, err := p.selectExpense(
		ctx,
		userID,
		message,
		arguments.Selector,
		arguments.filter(),
	)
	if err != nil || clarification != "" {
		return clarification, err
	}

	if err := p.expenses.DeleteForUser(ctx, userID, expense.ID); err != nil {
		return "", fmt.Errorf("delete Telegram user expense: %w", err)
	}

	return "Удалил: " + formatExpenseLine(expense) + ".", nil
}

func (p *Processor) selectExpense(
	ctx context.Context,
	userID int64,
	message string,
	selector string,
	filterArguments expenseFilterArguments,
) (domain.Expense, string, error) {
	selector = strings.TrimSpace(selector)
	if selector != selectorLast && selector != selectorSingle {
		return domain.Expense{}, "Уточните расход: например, «последний» или его название и дату.", nil
	}
	if selector == selectorSingle && !argumentsHaveFilter(filterArguments) {
		return domain.Expense{}, "Уточните, какой именно расход нужно изменить или удалить.", nil
	}
	if selector == selectorLast && !hasLastReference(message) {
		return domain.Expense{}, "Если нужен последний расход, явно напишите «последний».", nil
	}

	limit := 3
	if selector == selectorLast {
		limit = 1
	}
	filter, err := p.filterFromArguments(userID, filterArguments, limit)
	if err != nil {
		return domain.Expense{}, "", err
	}
	page, err := p.expenses.List(ctx, filter)
	if err != nil {
		return domain.Expense{}, "", fmt.Errorf("find Telegram user expense: %w", err)
	}
	if len(page.Items) == 0 {
		return domain.Expense{}, "Не нашёл подходящий расход.", nil
	}
	if selector == selectorSingle && page.Total > 1 {
		var response strings.Builder
		response.WriteString("Нашёл несколько подходящих расходов:\n")
		for _, expense := range page.Items {
			response.WriteString("\n• ")
			response.WriteString(formatExpenseLine(expense))
		}
		response.WriteString("\n\nУточните запрос, например добавьте дату или скажите «последний».")
		return domain.Expense{}, response.String(), nil
	}

	return page.Items[0], "", nil
}

func (p *Processor) updateInput(arguments updateExpenseArguments) (service.UpdateExpenseInput, bool, error) {
	var input service.UpdateExpenseInput
	hasChanges := false

	if newAmount := optionalString(arguments.NewAmount); newAmount != "" {
		amount, err := decimal.NewFromString(newAmount)
		if err != nil {
			return input, false, fmt.Errorf("parse LLM new_amount: %w", err)
		}
		input.Amount = &amount
		hasChanges = true
	}
	if newCategory := optionalString(arguments.NewCategory); newCategory != "" {
		category := domain.Category(newCategory)
		if !category.Valid() {
			return input, false, fmt.Errorf("LLM returned unknown new_category %q", newCategory)
		}
		input.Category = &category
		hasChanges = true
	}
	if newSpentAt := optionalString(arguments.NewSpentAt); newSpentAt != "" {
		spentAt, err := time.ParseInLocation(dateLayout, newSpentAt, p.timezone)
		if err != nil {
			return input, false, fmt.Errorf("parse LLM new_spent_at: %w", err)
		}
		input.SpentAt = &spentAt
		hasChanges = true
	}
	if arguments.NewComment != nil || arguments.ClearComment {
		comment := optionalString(arguments.NewComment)
		input.Comment = &comment
		hasChanges = true
	}

	return input, hasChanges, nil
}

func argumentsHaveFilter(arguments expenseFilterArguments) bool {
	return optionalString(arguments.Category) != "" || optionalString(arguments.DateFrom) != "" ||
		optionalString(arguments.DateTo) != "" || optionalString(arguments.CommentQuery) != ""
}

func hasDeleteIntent(message string) bool {
	message = strings.ToLower(message)
	return strings.Contains(message, "удал") || strings.Contains(message, "стер") ||
		strings.Contains(message, "убери")
}

func hasLastReference(message string) bool {
	message = strings.ToLower(message)
	return strings.Contains(message, "послед") || strings.Contains(message, "предыдущ")
}
