package telegram

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/AndreyPiganov/test-task-expenses/backend/internal/domain"
	"github.com/AndreyPiganov/test-task-expenses/backend/internal/service"
)

const (
	pollTimeout     = 30 * time.Second
	initialBackoff  = time.Second
	maximumBackoff  = 30 * time.Second
	welcomeResponse = "Я подключён и запомнил вас. На следующем этапе научусь понимать и сохранять расходы."
)

type APIClient interface {
	GetUpdates(context.Context, int, time.Duration) ([]Update, error)
	SendMessage(context.Context, int64, string) error
}

type MemberService interface {
	Upsert(context.Context, service.UpsertFamilyMemberInput) (domain.FamilyMember, error)
}

type MessageProcessor interface {
	Process(context.Context, int64, string) (string, error)
}

type Bot struct {
	client    APIClient
	members   MemberService
	processor MessageProcessor
	logger    *slog.Logger
}

func NewBot(
	client APIClient,
	members MemberService,
	processor MessageProcessor,
	logger *slog.Logger,
) *Bot {
	return &Bot{client: client, members: members, processor: processor, logger: logger}
}

func (b *Bot) Run(ctx context.Context) error {
	offset := 0
	backoff := initialBackoff

	for {
		updates, err := b.client.GetUpdates(ctx, offset, pollTimeout)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}

			b.logger.ErrorContext(ctx, "receive Telegram updates", "error", err)
			if !waitForRetry(ctx, backoff) {
				return nil
			}
			backoff = min(backoff*2, maximumBackoff)
			continue
		}

		backoff = initialBackoff
		for _, update := range updates {
			if update.ID >= offset {
				offset = update.ID + 1
			}
			b.handleUpdate(ctx, update)
		}
	}
}

func (b *Bot) handleUpdate(ctx context.Context, update Update) {
	if update.Message == nil || update.Message.From == nil || update.Message.From.IsBot {
		return
	}

	message := update.Message
	user := message.From
	_, err := b.members.Upsert(ctx, service.UpsertFamilyMemberInput{
		TelegramUserID: user.ID,
		Username:       user.Username,
		DisplayName:    displayName(*user),
	})
	if err != nil {
		b.logger.ErrorContext(ctx, "register Telegram user", "user_id", user.ID, "error", err)
		b.send(ctx, message.Chat.ID, "Не удалось зарегистрировать пользователя. Попробуйте позже.")
		return
	}

	response := welcomeResponse
	if strings.TrimSpace(message.Text) == "" {
		response = "Пожалуйста, отправьте расход текстовым сообщением."
	} else if b.processor != nil {
		response, err = b.processor.Process(ctx, user.ID, message.Text)
		if err != nil {
			b.logger.ErrorContext(ctx, "process Telegram message", "user_id", user.ID, "error", err)
			response = "Не удалось обработать сообщение. Попробуйте ещё раз."
		}
	}

	b.send(ctx, message.Chat.ID, response)
}

func (b *Bot) send(ctx context.Context, chatID int64, text string) {
	if err := b.client.SendMessage(ctx, chatID, text); err != nil && !errors.Is(err, context.Canceled) {
		b.logger.ErrorContext(ctx, "send Telegram message", "chat_id", chatID, "error", err)
	}
}

func displayName(user User) string {
	name := strings.TrimSpace(user.FirstName + " " + user.LastName)
	if name != "" {
		return name
	}
	if user.Username != "" {
		return "@" + user.Username
	}

	return strconv.FormatInt(user.ID, 10)
}

func waitForRetry(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
