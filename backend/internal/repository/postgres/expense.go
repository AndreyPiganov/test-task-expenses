package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/AndreyPiganov/test-task-expenses/backend/internal/domain"
	"github.com/AndreyPiganov/test-task-expenses/backend/internal/repository"
	"github.com/google/uuid"
)

type ExpenseRepository struct {
	db *sql.DB
}

func NewExpenseRepository(db *sql.DB) *ExpenseRepository {
	return &ExpenseRepository{db: db}
}

func (r *ExpenseRepository) Create(ctx context.Context, expense domain.Expense) (domain.Expense, error) {
	const query = `
		INSERT INTO expenses (id, user_id, amount, category, spent_at, comment)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''))
		RETURNING id, user_id, amount, category, spent_at, COALESCE(comment, ''), created_at, updated_at`

	created, err := scanExpense(r.db.QueryRowContext(
		ctx,
		query,
		expense.ID,
		expense.UserID,
		expense.Amount,
		expense.Category,
		expense.SpentAt,
		expense.Comment,
	))
	if err != nil {
		return domain.Expense{}, fmt.Errorf("create expense: %w", err)
	}

	return created, nil
}

func (r *ExpenseRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.Expense, error) {
	const query = `
		SELECT id, user_id, amount, category, spent_at, COALESCE(comment, ''), created_at, updated_at
		FROM expenses
		WHERE id = $1`

	expense, err := scanExpense(r.db.QueryRowContext(ctx, query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Expense{}, repository.ErrNotFound
	}
	if err != nil {
		return domain.Expense{}, fmt.Errorf("get expense: %w", err)
	}

	return expense, nil
}

func (r *ExpenseRepository) GetByIDForUser(
	ctx context.Context,
	id uuid.UUID,
	userID int64,
) (domain.Expense, error) {
	const query = `
		SELECT id, user_id, amount, category, spent_at, COALESCE(comment, ''), created_at, updated_at
		FROM expenses
		WHERE id = $1 AND user_id = $2`

	expense, err := scanExpense(r.db.QueryRowContext(ctx, query, id, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Expense{}, repository.ErrNotFound
	}
	if err != nil {
		return domain.Expense{}, fmt.Errorf("get expense for user: %w", err)
	}

	return expense, nil
}

func (r *ExpenseRepository) List(
	ctx context.Context,
	filter repository.ExpenseFilter,
) ([]domain.Expense, int, error) {
	conditions, args := expenseConditions(filter)
	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}

	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM expenses"+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count expenses: %w", err)
	}

	queryArgs := append([]any(nil), args...)
	limitPosition := len(queryArgs) + 1
	queryArgs = append(queryArgs, filter.Limit)
	offsetPosition := len(queryArgs) + 1
	queryArgs = append(queryArgs, filter.Offset)

	query := `
		SELECT id, user_id, amount, category, spent_at, COALESCE(comment, ''), created_at, updated_at
		FROM expenses` + where + `
		ORDER BY spent_at DESC, created_at DESC
		LIMIT $` + fmt.Sprint(limitPosition) + ` OFFSET $` + fmt.Sprint(offsetPosition)

	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list expenses: %w", err)
	}
	defer rows.Close()

	expenses := make([]domain.Expense, 0, filter.Limit)
	for rows.Next() {
		expense, scanErr := scanExpense(rows)
		if scanErr != nil {
			return nil, 0, fmt.Errorf("scan expense list: %w", scanErr)
		}
		expenses = append(expenses, expense)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate expense list: %w", err)
	}

	return expenses, total, nil
}

func (r *ExpenseRepository) Update(ctx context.Context, expense domain.Expense) (domain.Expense, error) {
	const query = `
		UPDATE expenses
		SET user_id = $2, amount = $3, category = $4, spent_at = $5, comment = NULLIF($6, '')
		WHERE id = $1
		RETURNING id, user_id, amount, category, spent_at, COALESCE(comment, ''), created_at, updated_at`

	updated, err := scanExpense(r.db.QueryRowContext(
		ctx,
		query,
		expense.ID,
		expense.UserID,
		expense.Amount,
		expense.Category,
		expense.SpentAt,
		expense.Comment,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Expense{}, repository.ErrNotFound
	}
	if err != nil {
		return domain.Expense{}, fmt.Errorf("update expense: %w", err)
	}

	return updated, nil
}

func (r *ExpenseRepository) UpdateForUser(
	ctx context.Context,
	expense domain.Expense,
	userID int64,
) (domain.Expense, error) {
	const query = `
		UPDATE expenses
		SET amount = $3, category = $4, spent_at = $5, comment = NULLIF($6, '')
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, amount, category, spent_at, COALESCE(comment, ''), created_at, updated_at`

	updated, err := scanExpense(r.db.QueryRowContext(
		ctx,
		query,
		expense.ID,
		userID,
		expense.Amount,
		expense.Category,
		expense.SpentAt,
		expense.Comment,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Expense{}, repository.ErrNotFound
	}
	if err != nil {
		return domain.Expense{}, fmt.Errorf("update expense for user: %w", err)
	}

	return updated, nil
}

func (r *ExpenseRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM expenses WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete expense: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get deleted expense count: %w", err)
	}
	if rowsAffected == 0 {
		return repository.ErrNotFound
	}

	return nil
}

func (r *ExpenseRepository) DeleteForUser(ctx context.Context, id uuid.UUID, userID int64) error {
	result, err := r.db.ExecContext(
		ctx,
		"DELETE FROM expenses WHERE id = $1 AND user_id = $2",
		id,
		userID,
	)
	if err != nil {
		return fmt.Errorf("delete expense for user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get deleted expense count for user: %w", err)
	}
	if rowsAffected == 0 {
		return repository.ErrNotFound
	}

	return nil
}

func expenseConditions(filter repository.ExpenseFilter) ([]string, []any) {
	conditions := make([]string, 0, 5)
	args := make([]any, 0, 5)

	add := func(condition string, value any) {
		args = append(args, value)
		conditions = append(conditions, fmt.Sprintf(condition, len(args)))
	}

	if filter.UserID != nil {
		add("user_id = $%d", *filter.UserID)
	}
	if filter.Category != nil {
		add("category = $%d", *filter.Category)
	}
	if filter.DateFrom != nil {
		add("spent_at >= $%d", *filter.DateFrom)
	}
	if filter.DateTo != nil {
		add("spent_at <= $%d", *filter.DateTo)
	}
	if filter.CommentQuery != "" {
		add("COALESCE(comment, '') ILIKE '%%' || $%d || '%%'", filter.CommentQuery)
	}

	return conditions, args
}

type rowScanner interface {
	Scan(...any) error
}

func scanExpense(row rowScanner) (domain.Expense, error) {
	var expense domain.Expense
	err := row.Scan(
		&expense.ID,
		&expense.UserID,
		&expense.Amount,
		&expense.Category,
		&expense.SpentAt,
		&expense.Comment,
		&expense.CreatedAt,
		&expense.UpdatedAt,
	)
	return expense, err
}
