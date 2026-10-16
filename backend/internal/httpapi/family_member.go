package httpapi

import (
	"net/http"
)

func (a *api) familyMembers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	if a.members == nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "service unavailable"})
		return
	}

	members, err := a.members.List(r.Context())
	if err != nil {
		a.writeServiceError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": members})
}
