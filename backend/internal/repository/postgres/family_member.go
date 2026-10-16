package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/AndreyPiganov/test-task-expenses/backend/internal/domain"
	"github.com/AndreyPiganov/test-task-expenses/backend/internal/repository"
)

type FamilyMemberRepository struct {
	db *sql.DB
}

func NewFamilyMemberRepository(db *sql.DB) *FamilyMemberRepository {
	return &FamilyMemberRepository{db: db}
}

func (r *FamilyMemberRepository) GetByTelegramID(
	ctx context.Context,
	telegramUserID int64,
) (domain.FamilyMember, error) {
	const query = `
		SELECT telegram_user_id, COALESCE(username, ''), display_name, created_at, updated_at
		FROM family_members
		WHERE telegram_user_id = $1`

	member, err := scanFamilyMember(r.db.QueryRowContext(ctx, query, telegramUserID))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.FamilyMember{}, repository.ErrNotFound
	}
	if err != nil {
		return domain.FamilyMember{}, fmt.Errorf("get family member: %w", err)
	}

	return member, nil
}

func (r *FamilyMemberRepository) List(ctx context.Context) ([]domain.FamilyMember, error) {
	const query = `
		SELECT telegram_user_id, COALESCE(username, ''), display_name, created_at, updated_at
		FROM family_members
		ORDER BY display_name, telegram_user_id`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list family members: %w", err)
	}
	defer rows.Close()

	members := make([]domain.FamilyMember, 0)
	for rows.Next() {
		member, scanErr := scanFamilyMember(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan family member list: %w", scanErr)
		}
		members = append(members, member)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate family member list: %w", err)
	}

	return members, nil
}

func (r *FamilyMemberRepository) Upsert(
	ctx context.Context,
	member domain.FamilyMember,
) (domain.FamilyMember, error) {
	const query = `
		INSERT INTO family_members (telegram_user_id, username, display_name)
		VALUES ($1, NULLIF($2, ''), $3)
		ON CONFLICT (telegram_user_id) DO UPDATE
		SET username = EXCLUDED.username, display_name = EXCLUDED.display_name
		RETURNING telegram_user_id, COALESCE(username, ''), display_name, created_at, updated_at`

	saved, err := scanFamilyMember(r.db.QueryRowContext(
		ctx,
		query,
		member.TelegramUserID,
		member.Username,
		member.DisplayName,
	))
	if err != nil {
		return domain.FamilyMember{}, fmt.Errorf("upsert family member: %w", err)
	}

	return saved, nil
}

func scanFamilyMember(row rowScanner) (domain.FamilyMember, error) {
	var member domain.FamilyMember
	err := row.Scan(
		&member.TelegramUserID,
		&member.Username,
		&member.DisplayName,
		&member.CreatedAt,
		&member.UpdatedAt,
	)
	return member, err
}
