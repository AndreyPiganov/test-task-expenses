package httpapi

import (
	"net/http"
	"time"

	"github.com/AndreyPiganov/test-task-expenses/backend/internal/domain"
)

type familyMemberResponse struct {
	TelegramUserID int64     `json:"telegram_user_id"`
	Username       string    `json:"username,omitempty"`
	DisplayName    string    `json:"display_name"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type familyMemberPageResponse struct {
	Items []familyMemberResponse `json:"items"`
}

func (a *api) familyMembers(w http.ResponseWriter, r *http.Request) {
	if a.members == nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "service unavailable"})
		return
	}

	members, err := a.members.List(r.Context())
	if err != nil {
		a.writeServiceError(w, r, err)
		return
	}

	items := make([]familyMemberResponse, 0, len(members))
	for _, member := range members {
		items = append(items, toFamilyMemberResponse(member))
	}

	writeJSON(w, http.StatusOK, familyMemberPageResponse{Items: items})
}

func toFamilyMemberResponse(member domain.FamilyMember) familyMemberResponse {
	return familyMemberResponse{
		TelegramUserID: member.TelegramUserID,
		Username:       member.Username,
		DisplayName:    member.DisplayName,
		CreatedAt:      member.CreatedAt,
		UpdatedAt:      member.UpdatedAt,
	}
}
