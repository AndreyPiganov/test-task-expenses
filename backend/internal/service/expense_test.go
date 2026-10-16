package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/AndreyPiganov/test-task-expenses/backend/internal/domain"
	"github.com/AndreyPiganov/test-task-expenses/backend/internal/repository"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestExpenseServiceCreate(t *testing.T) {
	t.Parallel()

	members := &fakeMemberRepository{members: map[int64]domain.FamilyMember{
		123: {TelegramUserID: 123, DisplayName: "Андрей"},
	}}
	expenses := &fakeExpenseRepository{items: make(map[uuid.UUID]domain.Expense)}
	service := NewExpenseService(expenses, members)

	created, err := service.Create(context.Background(), CreateExpenseInput{
		UserID:   123,
		Amount:   decimal.RequireFromString("350.50"),
		Category: domain.CategoryFood,
		SpentAt:  time.Date(2026, time.October, 16, 0, 0, 0, 0, time.UTC),
		Comment:  "  кофе  ",
	})
	if err != nil {
		t.Fatalf("Create() returned an error: %v", err)
	}
	if created.ID == uuid.Nil {
		t.Fatal("Create() did not assign an ID")
	}
	if created.Comment != "кофе" {
		t.Errorf("Comment = %q, want кофе", created.Comment)
	}
}

func TestExpenseServiceCreateRejectsUnknownMember(t *testing.T) {
	t.Parallel()

	service := NewExpenseService(
		&fakeExpenseRepository{items: make(map[uuid.UUID]domain.Expense)},
		&fakeMemberRepository{members: make(map[int64]domain.FamilyMember)},
	)

	_, err := service.Create(context.Background(), CreateExpenseInput{
		UserID:   999,
		Amount:   decimal.NewFromInt(100),
		Category: domain.CategoryOther,
		SpentAt:  time.Now(),
	})
	if !errors.Is(err, ErrMemberNotFound) {
		t.Fatalf("Create() error = %v, want ErrMemberNotFound", err)
	}
}

func TestExpenseServiceUpdateAndDelete(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	expenses := &fakeExpenseRepository{items: map[uuid.UUID]domain.Expense{
		id: {
			ID:       id,
			UserID:   123,
			Amount:   decimal.NewFromInt(350),
			Category: domain.CategoryFood,
			SpentAt:  time.Now(),
		},
	}}
	members := &fakeMemberRepository{members: map[int64]domain.FamilyMember{
		123: {TelegramUserID: 123, DisplayName: "Андрей"},
	}}
	service := NewExpenseService(expenses, members)
	newAmount := decimal.NewFromInt(450)

	updated, err := service.Update(context.Background(), id, UpdateExpenseInput{Amount: &newAmount})
	if err != nil {
		t.Fatalf("Update() returned an error: %v", err)
	}
	if !updated.Amount.Equal(newAmount) {
		t.Errorf("Amount = %s, want %s", updated.Amount, newAmount)
	}

	if err := service.Delete(context.Background(), id); err != nil {
		t.Fatalf("Delete() returned an error: %v", err)
	}
	if _, err := service.Get(context.Background(), id); !errors.Is(err, ErrExpenseNotFound) {
		t.Fatalf("Get() after Delete() error = %v, want ErrExpenseNotFound", err)
	}
}

func TestExpenseServiceListAppliesDefaultPageSize(t *testing.T) {
	t.Parallel()

	expenses := &fakeExpenseRepository{items: make(map[uuid.UUID]domain.Expense)}
	service := NewExpenseService(expenses, &fakeMemberRepository{})

	page, err := service.List(context.Background(), ExpenseFilter{})
	if err != nil {
		t.Fatalf("List() returned an error: %v", err)
	}
	if page.Limit != defaultPageSize {
		t.Errorf("Limit = %d, want %d", page.Limit, defaultPageSize)
	}
	if expenses.lastFilter.Limit != defaultPageSize {
		t.Errorf("repository limit = %d, want %d", expenses.lastFilter.Limit, defaultPageSize)
	}
}

func TestExpenseServiceScopedMutationCannotAccessAnotherUser(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	originalAmount := decimal.NewFromInt(350)
	expenses := &fakeExpenseRepository{items: map[uuid.UUID]domain.Expense{
		id: {
			ID:       id,
			UserID:   123,
			Amount:   originalAmount,
			Category: domain.CategoryFood,
			SpentAt:  time.Now(),
		},
	}}
	service := NewExpenseService(expenses, &fakeMemberRepository{})
	newAmount := decimal.NewFromInt(1)

	_, err := service.UpdateForUser(
		context.Background(),
		999,
		id,
		UpdateExpenseInput{Amount: &newAmount},
	)
	if !errors.Is(err, ErrExpenseNotFound) {
		t.Fatalf("UpdateForUser() error = %v, want ErrExpenseNotFound", err)
	}
	if !expenses.items[id].Amount.Equal(originalAmount) {
		t.Fatal("another user's expense was modified")
	}

	err = service.DeleteForUser(context.Background(), 999, id)
	if !errors.Is(err, ErrExpenseNotFound) {
		t.Fatalf("DeleteForUser() error = %v, want ErrExpenseNotFound", err)
	}
	if _, exists := expenses.items[id]; !exists {
		t.Fatal("another user's expense was deleted")
	}
}

type fakeExpenseRepository struct {
	items      map[uuid.UUID]domain.Expense
	lastFilter repository.ExpenseFilter
}

func (r *fakeExpenseRepository) Create(_ context.Context, expense domain.Expense) (domain.Expense, error) {
	r.items[expense.ID] = expense
	return expense, nil
}

func (r *fakeExpenseRepository) GetByID(_ context.Context, id uuid.UUID) (domain.Expense, error) {
	expense, ok := r.items[id]
	if !ok {
		return domain.Expense{}, repository.ErrNotFound
	}
	return expense, nil
}

func (r *fakeExpenseRepository) GetByIDForUser(
	_ context.Context,
	id uuid.UUID,
	userID int64,
) (domain.Expense, error) {
	expense, ok := r.items[id]
	if !ok || expense.UserID != userID {
		return domain.Expense{}, repository.ErrNotFound
	}
	return expense, nil
}

func (r *fakeExpenseRepository) List(
	_ context.Context,
	filter repository.ExpenseFilter,
) ([]domain.Expense, int, error) {
	r.lastFilter = filter
	items := make([]domain.Expense, 0, len(r.items))
	for _, expense := range r.items {
		items = append(items, expense)
	}
	return items, len(items), nil
}

func (r *fakeExpenseRepository) Update(_ context.Context, expense domain.Expense) (domain.Expense, error) {
	if _, ok := r.items[expense.ID]; !ok {
		return domain.Expense{}, repository.ErrNotFound
	}
	r.items[expense.ID] = expense
	return expense, nil
}

func (r *fakeExpenseRepository) UpdateForUser(
	_ context.Context,
	expense domain.Expense,
	userID int64,
) (domain.Expense, error) {
	stored, ok := r.items[expense.ID]
	if !ok || stored.UserID != userID {
		return domain.Expense{}, repository.ErrNotFound
	}
	r.items[expense.ID] = expense
	return expense, nil
}

func (r *fakeExpenseRepository) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := r.items[id]; !ok {
		return repository.ErrNotFound
	}
	delete(r.items, id)
	return nil
}

func (r *fakeExpenseRepository) DeleteForUser(_ context.Context, id uuid.UUID, userID int64) error {
	expense, ok := r.items[id]
	if !ok || expense.UserID != userID {
		return repository.ErrNotFound
	}
	delete(r.items, id)
	return nil
}

type fakeMemberRepository struct {
	members map[int64]domain.FamilyMember
}

func (r *fakeMemberRepository) GetByTelegramID(_ context.Context, id int64) (domain.FamilyMember, error) {
	member, ok := r.members[id]
	if !ok {
		return domain.FamilyMember{}, repository.ErrNotFound
	}
	return member, nil
}

func (r *fakeMemberRepository) List(_ context.Context) ([]domain.FamilyMember, error) {
	members := make([]domain.FamilyMember, 0, len(r.members))
	for _, member := range r.members {
		members = append(members, member)
	}
	return members, nil
}

func (r *fakeMemberRepository) Upsert(
	_ context.Context,
	member domain.FamilyMember,
) (domain.FamilyMember, error) {
	if r.members == nil {
		r.members = make(map[int64]domain.FamilyMember)
	}
	r.members[member.TelegramUserID] = member
	return member, nil
}
