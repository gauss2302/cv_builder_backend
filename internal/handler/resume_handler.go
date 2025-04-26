package handler

import (
	"cv_builder/internal/domain"
	"cv_builder/internal/repository"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"net/http"
)

type ResumeHandler struct {
	resumeRepo domain.ResumeRepository
}

func NewResumeHandler(resumeRepo domain.ResumeRepository) *ResumeHandler {
	return &ResumeHandler{
		resumeRepo: resumeRepo,
	}
}

func (h *ResumeHandler) GetResumeHandler(w http.ResponseWriter, r *http.Request) {
	claims, err := GetClaimsFromContext(r.Context())
	if err != nil {
		RespondWithError(w, http.StatusUnauthorized, "Unauthorized", "UNAUTHORIZED")
		return
	}
	userId, err := uuid.Parse(claims.UserID)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Invalid user ID", "INTERNAL_SERVER_ERROR")
		return
	}

	resumeId := r.PathValue("id")
	if resumeId == "" {
		RespondWithError(w, http.StatusBadRequest, "Resume ID is required", "INVALID_REQUEST")
		return
	}

	resumeUUID, err := uuid.Parse(resumeId)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid resume ID", "INVALID_REQUEST")
		return
	}

	resume, err := h.resumeRepo.GetCompleteResume(resumeUUID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			RespondWithError(w, http.StatusNotFound, "Resume not found", "NOT_FOUND")
			return
		}
		RespondWithError(w, http.StatusInternalServerError, "Failed to get resume", "INTERNAL_SERVER_ERROR")
		return
	}

	if resume.UserID != userId && claims.Role != "admin" {
		RespondWithError(w, http.StatusForbidden, "You don't have permission to access this resume", "FORBIDDEN")
		return
	}

	RespondWithJSON(w, http.StatusOK, resume)
}

func (h *ResumeHandler) CreateResumeHandler(w http.ResponseWriter, r *http.Request) {
	claims, err := GetClaimsFromContext(r.Context())
	if err != nil {
		RespondWithError(w, http.StatusUnauthorized, "Unauthorized", "UNAUTHORIZED")
		return
	}

	userId, err := uuid.Parse(claims.UserID)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Invalid user ID", "INTERNAL_SERVER_ERROR")
		return
	}

	resume, err := h.resumeRepo.CreateCV(userId)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to create resume", "INTERNAL_SERVER_ERROR")
		return
	}
	RespondWithJSON(w, http.StatusCreated, resume)
}

func (h *ResumeHandler) DeleteResumeHandler(w http.ResponseWriter, r *http.Request) {
	claims, err := GetClaimsFromContext(r.Context())

	if err != nil {
		RespondWithError(w, http.StatusUnauthorized, "Unauthorized", "UNAUTHORIZED")
		return
	}

	userId, err := uuid.Parse(claims.UserID)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Invalid user ID", "INTERNAL_SERVER_ERROR")
		return
	}

	resumeId := r.PathValue("id")
	if resumeId == "" {
		RespondWithError(w, http.StatusBadRequest, "Resume Id is required", "INVALID_REQUEST")
		return
	}

	resumeUUID, err := uuid.Parse(resumeId)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid resume ID", "INVALID_REQUEST")
		return
	}

	resume, err := h.resumeRepo.GetCVById(resumeUUID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			RespondWithError(w, http.StatusNotFound, "Resume not found", "NOT_FOUND")
			return
		}
		RespondWithError(w, http.StatusInternalServerError, "Failed to get resume", "INTERNAL_SERVER_ERROR")
		return
	}

	if resume.UserID != userId && claims.Role != "admin" {
		RespondWithError(w, http.StatusForbidden, "You don't have permission to delete this resume", "FORBIDDEN")
		return
	}

	if err := h.resumeRepo.DeleteCV(resumeUUID); err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to delete resume", "INTERNAL_SERVER_ERROR")
		return
	}

	RespondWithJSON(w, http.StatusOK, map[string]any{
		"message": "Resume deleted successfully",
	})

}

func (h *ResumeHandler) GetResumeListHandler(w http.ResponseWriter, r *http.Request) {
	claims, err := GetClaimsFromContext(r.Context())
	if err != nil {
		RespondWithError(w, http.StatusUnauthorized, "Unauthorized", "UNAUTHORIZED")
		return
	}

	// Get user ID from claims
	userId, err := uuid.Parse(claims.UserID)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Invalid user ID", "INTERNAL_SERVER_ERROR")
		return
	}

	// Get resumes from repository
	resumes, err := h.resumeRepo.GetCVByUserId(userId)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to get resumes", "INTERNAL_SERVER_ERROR")
		return
	}

	RespondWithJSON(w, http.StatusOK, resumes)
}

func (h *ResumeHandler) SavePersonalInfoHandler(w http.ResponseWriter, r *http.Request) {
	claims, err := GetClaimsFromContext(r.Context())
	if err != nil {
		RespondWithError(w, http.StatusUnauthorized, "Unauthorized", "UNAUTHORIZED")
		return
	}

	userId, err := uuid.Parse(claims.UserID)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Invalid user ID", "INTERNAL_SERVER_ERROR")
		return
	}

	resumeId := r.PathValue("id")
	if resumeId == "" {
		RespondWithError(w, http.StatusBadRequest, "Resume ID is required", "INVALID_REQUEST")
		return
	}

	resumeUUID, err := uuid.Parse(resumeId)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid resume ID", "INVALID_REQUEST")
		return
	}

	resume, err := h.resumeRepo.GetCVById(resumeUUID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			RespondWithError(w, http.StatusNotFound, "Resume not found", "NOT_FOUND")
			return
		}
		RespondWithError(w, http.StatusInternalServerError, "Failed to get resume", "INTERNAL_SERVER_ERROR")
		return
	}

	if resume.UserID != userId && claims.Role != "admin" {
		RespondWithError(w, http.StatusForbidden, "You don't have permission to update this resume", "FORBIDDEN")
		return
	}

	var personalInfo domain.PersonalInfo
	if err := json.NewDecoder(r.Body).Decode(&personalInfo); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST")
		return
	}

	if err := personalInfo.Validate(); err != nil {
		RespondWithError(w, http.StatusBadRequest, err.Error(), "VALIDATION_ERROR")
		return
	}

	personalInfo.BeforeSave()

	if err := h.resumeRepo.SavePersonalInfo(resumeUUID, &personalInfo); err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to save personal info", "INTERNAL_SERVER_ERROR")
		return
	}

	RespondWithJSON(w, http.StatusOK, map[string]any{
		"message": "Personal info saved successfully",
	})

}
