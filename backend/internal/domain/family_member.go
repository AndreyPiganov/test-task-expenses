package domain

import "time"

type FamilyMember struct {
	TelegramUserID int64     `json:"telegram_user_id"`
	Username       string    `json:"username,omitempty"`
	DisplayName    string    `json:"display_name"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
