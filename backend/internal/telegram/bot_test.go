package telegram

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/AndreyPiganov/test-task-expenses/backend/internal/domain"
	"github.com/AndreyPiganov/test-task-expenses/backend/internal/service"
)

func TestBotRegistersTelegramUserAndReplies(t *testing.T) {
	t.Parallel()

	client := &fakeAPIClient{}
	members := &fakeMemberService{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	bot := NewBot(client, members, nil, logger)

	bot.handleUpdate(context.Background(), Update{
		ID: 1,
		Message: &Message{
			From: &User{
				ID:        123,
				FirstName: "Андрей",
				LastName:  "Пиганов",
				Username:  "andrey",
			},
			Chat: Chat{ID: 456},
			Text: "кофе 350",
		},
	})

	if members.input.TelegramUserID != 123 {
		t.Errorf("TelegramUserID = %d, want 123", members.input.TelegramUserID)
	}
	if members.input.DisplayName != "Андрей Пиганов" {
		t.Errorf("DisplayName = %q, want full name", members.input.DisplayName)
	}
	if len(client.messages) != 1 {
		t.Fatalf("sent messages = %d, want 1", len(client.messages))
	}
	if client.messages[0].chatID != 456 || client.messages[0].text != welcomeResponse {
		t.Errorf("sent message = %+v, want welcome message in source chat", client.messages[0])
	}
}

func TestDisplayNameFallbacks(t *testing.T) {
	t.Parallel()

	if got := displayName(User{ID: 1, Username: "andrey"}); got != "@andrey" {
		t.Errorf("displayName() = %q, want @andrey", got)
	}
	if got := displayName(User{ID: 123}); got != "123" {
		t.Errorf("displayName() = %q, want numeric ID", got)
	}
}

type sentMessage struct {
	chatID int64
	text   string
}

type fakeAPIClient struct {
	messages []sentMessage
}

func (c *fakeAPIClient) GetUpdates(context.Context, int, time.Duration) ([]Update, error) {
	return nil, nil
}

func (c *fakeAPIClient) SendMessage(_ context.Context, chatID int64, text string) error {
	c.messages = append(c.messages, sentMessage{chatID: chatID, text: text})
	return nil
}

type fakeMemberService struct {
	input service.UpsertFamilyMemberInput
}

func (s *fakeMemberService) Upsert(
	_ context.Context,
	input service.UpsertFamilyMemberInput,
) (domain.FamilyMember, error) {
	s.input = input
	return domain.FamilyMember{
		TelegramUserID: input.TelegramUserID,
		Username:       input.Username,
		DisplayName:    input.DisplayName,
	}, nil
}
