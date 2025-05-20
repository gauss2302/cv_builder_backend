package handler

import (
	"cv_builder/internal/domain"
	"cv_builder/internal/repository"
	"errors"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"net/http"
)

type UserHandler struct {
	userRepo   domain.UserRepository
	resumeRepo domain.ResumeRepository
}

func NewUserHandler(userRepo domain.UserRepository, resumeRepo domain.ResumeRepository) *UserHandler {
	return &UserHandler{
		userRepo:   userRepo,
		resumeRepo: resumeRepo,
	}
}

func (h *UserHandler) GetProfileHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, err := GetClaimsFromContext(r.Context())
	if err != nil {
		RespondWithError(w, http.StatusUnauthorized, "Not Authorized", "UNAUTHORIZED")
		return
	}

	userId, err := uuid.Parse(claims.UserID)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Invalid user ID", "INTERNAL_SERVER_ERROR")
		return
	}

	user, err := h.userRepo.GetUserById(ctx, userId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			RespondWithError(w, http.StatusNotFound, "User not found", "NOT_FOUND")
			return
		}
		RespondWithError(w, http.StatusInternalServerError, "Failed to get user profile", "INTERNAL_SERVER_ERROR")
		return
	}

	resumes, err := h.resumeRepo.GetCVByUserId(ctx, userId)
	if err != nil {
		log.Error().Err(err).Str("user_id", userId.String()).Msg("Failed to get user resumes")
		// Continue without resumes
	}

	type resumeInfo struct {
		ID        string `json:"id"`
		CreatedAt string `json:"created_at"`
	}

	type profileResponse struct {
		UserID    string       `json:"user_id"`
		Email     string       `json:"email"`
		Role      string       `json:"role"`
		CreatedAt string       `json:"created_at"`
		Resumes   []resumeInfo `json:"resumes,omitempty"`
	}

	response := profileResponse{UserID: user.ID.String(), Email: user.Email, Role: user.Role, CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z")}

	if resumes != nil {
		resumeList := make([]resumeInfo, len(resumes))
		for i, resume := range resumes {
			resumeList[i] = resumeInfo{
				ID:        resume.ID.String(),
				CreatedAt: resume.CreatedAt.Format("2006-01-02T15:04:05Z"),
			}
		}
		response.Resumes = resumeList
	}
	RespondWithJSON(w, http.StatusOK, response)
}
