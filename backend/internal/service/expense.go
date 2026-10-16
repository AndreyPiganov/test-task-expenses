package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/AndreyPiganov/test-task-expenses/backend/internal/domain"
	"github.com/AndreyPiganov/test-task-expenses/backend/internal/repository"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const (
	defaultPageSize = 50
	maxPageSize     = 100
)

type CreateExpenseInput struct {
	UserID   int64
	Amount   decimal.Decimal
	Category domain.Category
	SpentAt  time.Time
	Comment  string
}

type UpdateExpenseInput struct {
	UserID   *int64
	Amount   *decimal.Decimal
	Category *domain.Category
	SpentAt  *time.Time
	Comment  *string
}

type ExpenseFilter struct {
	UserID       *int64
	Category     *domain.Category
	DateFrom     *time.Time
	DateTo       *time.Time
	CommentQuery string
	Limit        int
	Offset       int
}

type ExpensePage struct {
	Items  []domain.Expense
	Total  int
	Limit  int
	Offset int
}

type ExpenseService struct {
	expenses repository.ExpenseRepository
	members  repository.FamilyMemberRepository
}

func NewExpenseService(
	expenses repository.ExpenseRepository,
	members repository.FamilyMemberRepository,
) *ExpenseService {
	return &ExpenseService{expenses: expenses, members: members}
}

func (s *ExpenseService) Create(ctx context.Context, input CreateExpenseInput) (domain.Expense, error) {
	if err := s.ensureMemberExists(ctx, input.UserID); err != nil {
		return domain.Expense{}, err
	}

	expense := domain.Expense{
		ID:       uuid.New(),
		UserID:   input.UserID,
		Amount:   input.Amount,
		Category: input.Category,
		SpentAt:  input.SpentAt,
		Comment:  strings.TrimSpace(input.Comment),
	}
	if err := expense.Validate(); err != nil {
		return domain.Expense{}, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	created, err := s.expenses.Create(ctx, expense)
	if err != nil {
		return domain.Expense{}, fmt.Errorf("create expense: %w", err)
	}

	return created, nil
}

func (s *ExpenseService) Get(ctx context.Context, id uuid.UUID) (domain.Expense, error) {
	expense, err := s.expenses.GetByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return domain.Expense{}, ErrExpenseNotFound
	}
	if err != nil {
		return domain.Expense{}, fmt.Errorf("get expense: %w", err)
	}

	return expense, nil
}

func (s *ExpenseService) List(ctx context.Context, filter ExpenseFilter) (ExpensePage, error) {
	if err := validateFilter(filter); err != nil {
		return ExpensePage{}, err
	}

	if filter.Limit == 0 {
		filter.Limit = defaultPageSize
	}

	items, total, err := s.expenses.List(ctx, repository.ExpenseFilter{
		UserID:       filter.UserID,
		Category:     filter.Category,
		DateFrom:     filter.DateFrom,
		DateTo:       filter.DateTo,
		CommentQuery: strings.TrimSpace(filter.CommentQuery),
		Limit:        filter.Limit,
		Offset:       filter.Offset,
	})
	if err != nil {
		return ExpensePage{}, fmt.Errorf("list expenses: %w", err)
	}

	return ExpensePage{
		Items:  items,
		Total:  total,
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}, nil
}

func (s *ExpenseService) UpdateForUser(
	ctx context.Context,
	userID int64,
	id uuid.UUID,
	input UpdateExpenseInput,
) (domain.Expense, error) {
	if userID <= 0 {
		return domain.Expense{}, fmt.Errorf("%w: user ID must be positive", ErrInvalidInput)
	}
	if input.UserID != nil && *input.UserID != userID {
		return domain.Expense{}, fmt.Errorf("%w: expense owner cannot be changed by the user", ErrInvalidInput)
	}
	if input.Amount == nil && input.Category == nil && input.SpentAt == nil && input.Comment == nil {
		return domain.Expense{}, fmt.Errorf("%w: at least one field is required", ErrInvalidInput)
	}

	expense, err := s.expenses.GetByIDForUser(ctx, id, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return domain.Expense{}, ErrExpenseNotFound
	}
	if err != nil {
		return domain.Expense{}, fmt.Errorf("get expense for user: %w", err)
	}

	applyExpenseChanges(&expense, input)
	if err := expense.Validate(); err != nil {
		return domain.Expense{}, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	updated, err := s.expenses.UpdateForUser(ctx, expense, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return domain.Expense{}, ErrExpenseNotFound
	}
	if err != nil {
		return domain.Expense{}, fmt.Errorf("update expense for user: %w", err)
	}

	return updated, nil
}

func (s *ExpenseService) Update(
	ctx context.Context,
	id uuid.UUID,
	input UpdateExpenseInput,
) (domain.Expense, error) {
	if input.UserID == nil && input.Amount == nil && input.Category == nil &&
		input.SpentAt == nil && input.Comment == nil {
		return domain.Expense{}, fmt.Errorf("%w: at least one field is required", ErrInvalidInput)
	}

	expense, err := s.Get(ctx, id)
	if err != nil {
		return domain.Expense{}, err
	}

	if input.UserID != nil && *input.UserID != expense.UserID {
		if err := s.ensureMemberExists(ctx, *input.UserID); err != nil {
			return domain.Expense{}, err
		}
		expense.UserID = *input.UserID
	}
	if input.Amount != nil {
		expense.Amount = *input.Amount
	}
	if input.Category != nil {
		expense.Category = *input.Category
	}
	if input.SpentAt != nil {
		expense.SpentAt = *input.SpentAt
	}
	if input.Comment != nil {
		expense.Comment = strings.TrimSpace(*input.Comment)
	}

	if err := expense.Validate(); err != nil {
		return domain.Expense{}, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	updated, err := s.expenses.Update(ctx, expense)
	if errors.Is(err, repository.ErrNotFound) {
		return domain.Expense{}, ErrExpenseNotFound
	}
	if err != nil {
		return domain.Expense{}, fmt.Errorf("update expense: %w", err)
	}

	return updated, nil
}

func (s *ExpenseService) Delete(ctx context.Context, id uuid.UUID) error {
	err := s.expenses.Delete(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrExpenseNotFound
	}
	if err != nil {
		return fmt.Errorf("delete expense: %w", err)
	}

	return nil
}

func (s *ExpenseService) DeleteForUser(ctx context.Context, userID int64, id uuid.UUID) error {
	if userID <= 0 {
		return fmt.Errorf("%w: user ID must be positive", ErrInvalidInput)
	}

	err := s.expenses.DeleteForUser(ctx, id, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrExpenseNotFound
	}
	if err != nil {
		return fmt.Errorf("delete expense for user: %w", err)
	}

	return nil
}

func (s *ExpenseService) ensureMemberExists(ctx context.Context, userID int64) error {
	if userID <= 0 {
		return fmt.Errorf("%w: user ID must be positive", ErrInvalidInput)
	}

	_, err := s.members.GetByTelegramID(ctx, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrMemberNotFound
	}
	if err != nil {
		return fmt.Errorf("get family member: %w", err)
	}

	return nil
}

func validateFilter(filter ExpenseFilter) error {
	switch {
	case filter.UserID != nil && *filter.UserID <= 0:
		return fmt.Errorf("%w: user ID must be positive", ErrInvalidInput)
	case filter.Category != nil && !filter.Category.Valid():
		return fmt.Errorf("%w: unknown category", ErrInvalidInput)
	case filter.DateFrom != nil && filter.DateTo != nil && filter.DateFrom.After(*filter.DateTo):
		return fmt.Errorf("%w: date_from must not be after date_to", ErrInvalidInput)
	case len([]rune(strings.TrimSpace(filter.CommentQuery))) > 200:
		return fmt.Errorf("%w: comment query must not exceed 200 characters", ErrInvalidInput)
	case filter.Limit < 0 || filter.Limit > maxPageSize:
		return fmt.Errorf("%w: limit must be between 1 and %d", ErrInvalidInput, maxPageSize)
	case filter.Offset < 0:
		return fmt.Errorf("%w: offset must not be negative", ErrInvalidInput)
	default:
		return nil
	}
}

func applyExpenseChanges(expense *domain.Expense, input UpdateExpenseInput) {
	if input.Amount != nil {
		expense.Amount = *input.Amount
	}
	if input.Category != nil {
		expense.Category = *input.Category
	}
	if input.SpentAt != nil {
		expense.SpentAt = *input.SpentAt
	}
	if input.Comment != nil {
		expense.Comment = strings.TrimSpace(*input.Comment)
	}
}
