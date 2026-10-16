package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var maxExpenseAmount = decimal.RequireFromString("999999999999.99")

type Category string

const (
	CategoryFood          Category = "еда"
	CategoryTransport     Category = "транспорт"
	CategoryHousing       Category = "жильё"
	CategoryEntertainment Category = "развлечения"
	CategoryHealth        Category = "здоровье"
	CategoryShopping      Category = "покупки"
	CategoryOther         Category = "другое"
)

var categories = [...]Category{
	CategoryFood,
	CategoryTransport,
	CategoryHousing,
	CategoryEntertainment,
	CategoryHealth,
	CategoryShopping,
	CategoryOther,
}

// Expense is a single spending record owned by one family member.
type Expense struct {
	ID        uuid.UUID       `json:"id"`
	UserID    int64           `json:"user_id"`
	Amount    decimal.Decimal `json:"amount"`
	Category  Category        `json:"category"`
	SpentAt   time.Time       `json:"spent_at"`
	Comment   string          `json:"comment,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

func (e Expense) Validate() error {
	switch {
	case e.ID == uuid.Nil:
		return errors.New("expense ID is required")
	case e.UserID <= 0:
		return errors.New("user ID must be positive")
	case !e.Amount.IsPositive():
		return errors.New("amount must be positive")
	case e.Amount.GreaterThan(maxExpenseAmount):
		return errors.New("amount is too large")
	case !e.Amount.Equal(e.Amount.Truncate(2)):
		return errors.New("amount must have at most two decimal places")
	case !e.Category.Valid():
		return errors.New("unknown expense category")
	case e.SpentAt.IsZero():
		return errors.New("spent date is required")
	case len([]rune(strings.TrimSpace(e.Comment))) > 1000:
		return errors.New("comment must not exceed 1000 characters")
	default:
		return nil
	}
}

func (c Category) Valid() bool {
	for _, allowed := range categories {
		if c == allowed {
			return true
		}
	}

	return false
}

func Categories() []Category {
	result := make([]Category, len(categories))
	copy(result, categories[:])
	return result
}
