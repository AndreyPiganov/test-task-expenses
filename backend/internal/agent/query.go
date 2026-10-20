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

func (p *Processor) listExpenses(
	ctx context.Context,
	userID int64,
	arguments expenseFilterArguments,
) (string, error) {
	filter, err := p.filterFromArguments(userID, arguments, 10)
	if err != nil {
		return "", err
	}

	page, err := p.expenses.List(ctx, filter)
	if err != nil {
		return "", fmt.Errorf("list Telegram user expenses: %w", err)
	}
	if len(page.Items) == 0 {
		return "Расходов по этому запросу не найдено.", nil
	}

	var response strings.Builder
	response.WriteString("Нашёл расходы:\n")
	for _, expense := range page.Items {
		response.WriteString("\n• ")
		response.WriteString(formatExpenseLine(expense))
	}
	if page.Total > len(page.Items) {
		response.WriteString(fmt.Sprintf("\n\nПоказаны первые %d из %d.", len(page.Items), page.Total))
	}

	return response.String(), nil
}

func (p *Processor) summarizeExpenses(
	ctx context.Context,
	userID int64,
	arguments expenseFilterArguments,
) (string, error) {
	filter, err := p.filterFromArguments(userID, arguments, 100)
	if err != nil {
		return "", err
	}

	totalAmount := decimal.Zero
	totalCount := 0
	for {
		page, listErr := p.expenses.List(ctx, filter)
		if listErr != nil {
			return "", fmt.Errorf("summarize Telegram user expenses: %w", listErr)
		}
		for _, expense := range page.Items {
			totalAmount = totalAmount.Add(expense.Amount)
		}
		totalCount += len(page.Items)

		if totalCount >= page.Total || len(page.Items) == 0 {
			break
		}
		filter.Offset += len(page.Items)
	}

	if totalCount == 0 {
		return "Расходов по этому запросу не найдено.", nil
	}

	return fmt.Sprintf("Итого: %s ₽. Количество расходов: %d.", totalAmount.String(), totalCount), nil
}

func (p *Processor) filterFromArguments(
	userID int64,
	arguments expenseFilterArguments,
	limit int,
) (service.ExpenseFilter, error) {
	filter := service.ExpenseFilter{
		UserID:       &userID,
		CommentQuery: optionalString(arguments.CommentQuery),
		Limit:        limit,
	}

	if categoryValue := optionalString(arguments.Category); categoryValue != "" {
		category := domain.Category(categoryValue)
		if !category.Valid() {
			return service.ExpenseFilter{}, fmt.Errorf("LLM returned unknown filter category %q", categoryValue)
		}
		filter.Category = &category
	}
	if dateFromValue := optionalString(arguments.DateFrom); dateFromValue != "" {
		dateFrom, err := time.ParseInLocation(dateLayout, dateFromValue, p.timezone)
		if err != nil {
			return service.ExpenseFilter{}, fmt.Errorf("parse LLM date_from: %w", err)
		}
		filter.DateFrom = &dateFrom
	}
	if dateToValue := optionalString(arguments.DateTo); dateToValue != "" {
		dateTo, err := time.ParseInLocation(dateLayout, dateToValue, p.timezone)
		if err != nil {
			return service.ExpenseFilter{}, fmt.Errorf("parse LLM date_to: %w", err)
		}
		filter.DateTo = &dateTo
	}

	return filter, nil
}
