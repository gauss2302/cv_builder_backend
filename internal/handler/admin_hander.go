package handler

import (
	"cv_builder/internal/domain"
	"net/http"
)

type AdminHandler struct {
	userRepo domain.UserRepository
}

func NewAdminHandler(userRepo domain.UserRepository) *AdminHandler {
	return &AdminHandler{
		userRepo: userRepo,
	}
}

func (h *AdminHandler) GeUsersHandler(w http.ResponseWriter, r *http.Request) {
	claims, err := GetClaimsFromContext(r.Context())
	if err != nil {
		RespondWithError(w, http.StatusUnauthorized, "Not Authorized", "UNAUTHORIZED")
		return
	}

	RespondWithJSON(w, http.StatusOK, map[string]any{"message": "Admin route accessed successfully",
		"admin_id": claims.UserID,
		"email":    claims.Email})
}
