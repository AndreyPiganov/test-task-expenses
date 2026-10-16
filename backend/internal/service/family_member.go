package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/AndreyPiganov/test-task-expenses/backend/internal/domain"
	"github.com/AndreyPiganov/test-task-expenses/backend/internal/repository"
)

type UpsertFamilyMemberInput struct {
	TelegramUserID int64
	Username       string
	DisplayName    string
}

type FamilyMemberService struct {
	members repository.FamilyMemberRepository
}

func NewFamilyMemberService(members repository.FamilyMemberRepository) *FamilyMemberService {
	return &FamilyMemberService{members: members}
}

func (s *FamilyMemberService) List(ctx context.Context) ([]domain.FamilyMember, error) {
	members, err := s.members.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list family members: %w", err)
	}

	return members, nil
}

func (s *FamilyMemberService) Upsert(
	ctx context.Context,
	input UpsertFamilyMemberInput,
) (domain.FamilyMember, error) {
	displayName := strings.TrimSpace(input.DisplayName)
	if input.TelegramUserID <= 0 {
		return domain.FamilyMember{}, fmt.Errorf("%w: Telegram user ID must be positive", ErrInvalidInput)
	}
	if displayName == "" {
		return domain.FamilyMember{}, fmt.Errorf("%w: display name is required", ErrInvalidInput)
	}

	member, err := s.members.Upsert(ctx, domain.FamilyMember{
		TelegramUserID: input.TelegramUserID,
		Username:       strings.TrimSpace(input.Username),
		DisplayName:    displayName,
	})
	if errors.Is(err, repository.ErrNotFound) {
		return domain.FamilyMember{}, ErrMemberNotFound
	}
	if err != nil {
		return domain.FamilyMember{}, fmt.Errorf("upsert family member: %w", err)
	}

	return member, nil
}
