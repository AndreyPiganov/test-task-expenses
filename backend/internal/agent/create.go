package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/AndreyPiganov/test-task-expenses/backend/internal/currency"
	"github.com/AndreyPiganov/test-task-expenses/backend/internal/domain"
	"github.com/AndreyPiganov/test-task-expenses/backend/internal/service"
	"github.com/shopspring/decimal"
)

func (p *Processor) createExpense(
	ctx context.Context,
	userID int64,
	message string,
	arguments createExpenseArguments,
) (string, error) {
	originalAmount, err := decimal.NewFromString(arguments.Amount)
	if err != nil {
		return "", fmt.Errorf("parse LLM amount: %w", err)
	}
	category := domain.Category(arguments.Category)
	if !category.Valid() {
		return "", fmt.Errorf("LLM returned unknown category %q", arguments.Category)
	}
	spentAt, err := time.ParseInLocation(dateLayout, arguments.SpentAt, p.timezone)
	if err != nil {
		return "", fmt.Errorf("parse LLM spent_at: %w", err)
	}

	amount := originalAmount
	currencyCode := resolveCurrency(message, arguments.Currency)
	var conversion *currency.Conversion
	if currencyCode != "RUB" {
		if p.currencyConverter == nil {
			return "", errors.New("currency converter is not configured")
		}
		converted, conversionErr := p.currencyConverter.ToRubles(
			ctx,
			originalAmount,
			currencyCode,
			spentAt,
		)
		if conversionErr != nil {
			return "", fmt.Errorf("convert %s expense to rubles: %w", currencyCode, conversionErr)
		}
		amount = converted.AmountRUB
		conversion = &converted
	}

	expense, err := p.expenses.Create(ctx, service.CreateExpenseInput{
		UserID:   userID,
		Amount:   amount,
		Category: category,
		SpentAt:  spentAt,
		Comment:  arguments.Comment,
	})
	if err != nil {
		return "", fmt.Errorf("create expense from Telegram: %w", err)
	}

	return formatCreatedExpense(expense, originalAmount, currencyCode, conversion), nil
}

func normalizeCurrency(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "" || value == "RUR" {
		return "RUB"
	}
	return value
}

func resolveCurrency(message, llmCurrency string) string {
	message = strings.ToLower(message)
	for _, alias := range explicitCurrencyAliases {
		for _, fragment := range alias.fragments {
			if strings.Contains(message, fragment) {
				return alias.code
			}
		}
		for _, word := range alias.words {
			if containsWord(message, word) {
				return alias.code
			}
		}
	}
	return normalizeCurrency(llmCurrency)
}

type currencyAlias struct {
	code      string
	fragments []string
	words     []string
}

var explicitCurrencyAliases = []currencyAlias{
	{code: "UAH", fragments: []string{"грив", "₴"}, words: []string{"грн", "uah"}},
	{code: "KZT", fragments: []string{"₸"}, words: []string{"тенге", "kzt"}},
}

func containsWord(text, target string) bool {
	for _, word := range strings.FieldsFunc(text, func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= 'а' && r <= 'я') || r == 'ё')
	}) {
		if word == target {
			return true
		}
	}
	return false
}
