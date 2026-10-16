package repository

import (
	"context"
	"time"

	"github.com/AndreyPiganov/test-task-expenses/backend/internal/domain"
	"github.com/google/uuid"
)

type ExpenseFilter struct {
	UserID       *int64
	Category     *domain.Category
	DateFrom     *time.Time
	DateTo       *time.Time
	CommentQuery string
	Limit        int
	Offset       int
}

type ExpenseRepository interface {
	Create(context.Context, domain.Expense) (domain.Expense, error)
	GetByID(context.Context, uuid.UUID) (domain.Expense, error)
	GetByIDForUser(context.Context, uuid.UUID, int64) (domain.Expense, error)
	List(context.Context, ExpenseFilter) ([]domain.Expense, int, error)
	Update(context.Context, domain.Expense) (domain.Expense, error)
	UpdateForUser(context.Context, domain.Expense, int64) (domain.Expense, error)
	Delete(context.Context, uuid.UUID) error
	DeleteForUser(context.Context, uuid.UUID, int64) error
}

type FamilyMemberRepository interface {
	GetByTelegramID(context.Context, int64) (domain.FamilyMember, error)
	List(context.Context) ([]domain.FamilyMember, error)
	Upsert(context.Context, domain.FamilyMember) (domain.FamilyMember, error)
}
