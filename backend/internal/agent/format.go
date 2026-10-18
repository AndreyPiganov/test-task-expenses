package agent

import (
	"fmt"

	"github.com/AndreyPiganov/test-task-expenses/backend/internal/currency"
	"github.com/AndreyPiganov/test-task-expenses/backend/internal/domain"
	"github.com/shopspring/decimal"
)

func formatCreatedExpense(
	expense domain.Expense,
	originalAmount decimal.Decimal,
	currencyCode string,
	conversion *currency.Conversion,
) string {
	comment := expense.Comment
	if comment == "" {
		comment = string(expense.Category)
	}
	conversionText := ""
	if conversion != nil {
		conversionText = fmt.Sprintf(
			" (из %s %s по курсу Банка России на %s)",
			originalAmount.String(),
			currencyCode,
			conversion.RateDate.Format("02.01.2006"),
		)
	}

	return fmt.Sprintf(
		"Записал: %s — %s ₽%s, категория «%s», дата %s.",
		comment,
		expense.Amount.String(),
		conversionText,
		expense.Category,
		expense.SpentAt.Format("02.01.2006"),
	)
}

func formatExpenseLine(expense domain.Expense) string {
	comment := expense.Comment
	if comment == "" {
		comment = string(expense.Category)
	}

	return fmt.Sprintf(
		"%s — %s: %s ₽ (%s)",
		expense.SpentAt.Format("02.01.2006"),
		comment,
		expense.Amount.String(),
		expense.Category,
	)
}
