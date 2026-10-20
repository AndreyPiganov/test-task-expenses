package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AndreyPiganov/test-task-expenses/backend/internal/domain"
)

func TestFamilyMembersEndpointUsesHTTPResponseDTO(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := NewHandler(logger, nil, nil, stubFamilyMemberService{
		members: []domain.FamilyMember{{
			TelegramUserID: 123,
			Username:       "andrey",
			DisplayName:    "Андрей",
		}},
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/members", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	body := response.Body.String()
	if !strings.Contains(body, `"telegram_user_id":123`) ||
		!strings.Contains(body, `"display_name":"Андрей"`) {
		t.Errorf("body = %s, want member response fields", body)
	}
}

type stubFamilyMemberService struct {
	members []domain.FamilyMember
}

func (s stubFamilyMemberService) List(context.Context) ([]domain.FamilyMember, error) {
	return s.members, nil
}
