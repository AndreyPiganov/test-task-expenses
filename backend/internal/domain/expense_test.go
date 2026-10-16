package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestExpenseValidate(t *testing.T) {
	valid := Expense{
		ID:       uuid.New(),
		UserID:   123,
		Amount:   decimal.NewFromInt(350),
		Category: CategoryFood,
		SpentAt:  time.Now(),
	}

	if err := valid.Validate(); err != nil {
		t.Fatalf("valid expense returned an error: %v", err)
	}

	invalid := valid
	invalid.Amount = decimal.Zero
	if err := invalid.Validate(); err == nil {
		t.Fatal("expense with a zero amount returned nil error")
	}

	invalid = valid
	invalid.Category = Category("unknown")
	if err := invalid.Validate(); err == nil {
		t.Fatal("expense with an unknown category returned nil error")
	}
}

func TestCategoriesReturnsCopy(t *testing.T) {
	first := Categories()
	first[0] = CategoryOther

	second := Categories()
	if second[0] != CategoryFood {
		t.Fatal("Categories() returned mutable package state")
	}
}
