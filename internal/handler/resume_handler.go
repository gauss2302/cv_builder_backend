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
	ctx := r.Context()
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

	resume, err := h.resumeRepo.GetCompleteResume(ctx, resumeUUID)
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
	ctx := r.Context()
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

	resume, err := h.resumeRepo.CreateCV(ctx, userId)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to create resume", "INTERNAL_SERVER_ERROR")
		return
	}
	RespondWithJSON(w, http.StatusCreated, resume)
}

func (h *ResumeHandler) DeleteResumeHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
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

	resume, err := h.resumeRepo.GetCVById(ctx, resumeUUID)
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

	if err := h.resumeRepo.DeleteCV(ctx, resumeUUID); err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to delete resume", "INTERNAL_SERVER_ERROR")
		return
	}

	RespondWithJSON(w, http.StatusOK, map[string]any{
		"message": "Resume deleted successfully",
	})

}

func (h *ResumeHandler) GetResumeListHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
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
	resumes, err := h.resumeRepo.GetCVByUserId(ctx, userId)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to get resumes", "INTERNAL_SERVER_ERROR")
		return
	}

	RespondWithJSON(w, http.StatusOK, resumes)
}

func (h *ResumeHandler) SavePersonalInfoHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
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

	resume, err := h.resumeRepo.GetCVById(ctx, resumeUUID)
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

	if err := h.resumeRepo.SavePersonalInfo(ctx, resumeUUID, &personalInfo); err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to save personal info", "INTERNAL_SERVER_ERROR")
		return
	}

	RespondWithJSON(w, http.StatusOK, map[string]any{
		"message": "Personal info saved successfully",
	})

}

func (h *ResumeHandler) AddEducationHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
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

	resumeID := r.PathValue("id")
	if resumeID == "" {
		RespondWithError(w, http.StatusBadRequest, "Resume ID is required", "INVALID_REQUEST")
		return
	}

	resumeUUID, err := uuid.Parse(resumeID)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid resume ID", "INVALID_REQUEST")
		return
	}

	resume, err := h.resumeRepo.GetCVById(ctx, resumeUUID)
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

	var education domain.Education
	if err := json.NewDecoder(r.Body).Decode(&education); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST")
		return
	}

	if err := education.Validate(); err != nil {
		RespondWithError(w, http.StatusBadRequest, err.Error(), "VALIDATION_ERROR")
		return
	}

	education.BeforeSave()

	educationID, err := h.resumeRepo.AddEducation(ctx, resumeUUID, &education)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to add education", "INTERNAL_SERVER_ERROR")
		return
	}

	RespondWithJSON(w, http.StatusCreated, map[string]any{
		"id":      educationID,
		"message": "Education added successfully",
	})
}

func (h *ResumeHandler) GetEducationHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
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

	resume, err := h.resumeRepo.GetCVById(ctx, resumeUUID)
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

	education, err := h.resumeRepo.GetEducationByResume(ctx, resumeUUID)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to get education entries", "INTERNAL_SERVER_ERROR")
		return
	}

	RespondWithJSON(w, http.StatusOK, education)
}

func (h *ResumeHandler) DeleteEducationHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, err := GetClaimsFromContext(r.Context())
	if err != nil {
		RespondWithError(w, http.StatusUnauthorized, "Unauthorized", "UNAUTHORIZED")
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Invalid user ID", "INTERNAL_SERVER_ERROR")
		return
	}

	resumeId := r.PathValue("id")
	educationId := r.PathValue("educationId")

	if resumeId == "" {
		RespondWithError(w, http.StatusBadRequest, "Resume ID is required", "INVALID_REQUEST")
		return
	}

	if educationId == "" {
		RespondWithError(w, http.StatusBadRequest, "Education ID is required", "INVALID_REQUEST")
		return
	}

	resumeUUID, err := uuid.Parse(resumeId)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid resume ID", "INVALID_REQUEST")
		return
	}

	educationUUID, err := uuid.Parse(educationId)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid education ID", "INVALID_REQUEST")
		return
	}

	resume, err := h.resumeRepo.GetCVById(ctx, resumeUUID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			RespondWithError(w, http.StatusNotFound, "Resume not found", "NOT_FOUND")
			return
		}
		RespondWithError(w, http.StatusInternalServerError, "Failed to get resume", "INTERNAL_SERVER_ERROR")
		return
	}

	if resume.UserID != userID && claims.Role != "admin" {
		RespondWithError(w, http.StatusForbidden, "You don't have permission to update this resume", "FORBIDDEN")
		return
	}

	if err := h.resumeRepo.DeleteEducation(ctx, educationUUID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			RespondWithError(w, http.StatusNotFound, "Education entry not found", "NOT_FOUND")
			return
		}
		RespondWithError(w, http.StatusInternalServerError, "Failed to delete education entry", "INTERNAL_SERVER_ERROR")
		return
	}

	RespondWithJSON(w, http.StatusOK, map[string]any{
		"message": "Education entry deleted successfully",
	})
}

func (h *ResumeHandler) AddExperienceHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
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

	resume, err := h.resumeRepo.GetCVById(ctx, resumeUUID)
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

	var experience domain.Experience
	if err := json.NewDecoder(r.Body).Decode(&experience); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST")
		return
	}

	if err := experience.Validate(); err != nil {
		RespondWithError(w, http.StatusBadRequest, err.Error(), "VALIDATION_ERROR")
		return
	}

	experience.BeforeSave()

	experienceID, err := h.resumeRepo.AddExperience(ctx, resumeUUID, &experience)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to add experience", "INTERNAL_SERVER_ERROR")
		return
	}

	RespondWithJSON(w, http.StatusCreated, map[string]any{
		"id":      experienceID,
		"message": "Experience added successfully",
	})
}

func (h *ResumeHandler) GetExperienceHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
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

	resume, err := h.resumeRepo.GetCVById(ctx, resumeUUID)
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

	experience, err := h.resumeRepo.GetExperienceByResume(ctx, resumeUUID)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to get experience entries", "INTERNAL_SERVER_ERROR")
		return
	}

	RespondWithJSON(w, http.StatusOK, experience)
}

func (h *ResumeHandler) DeleteExperienceHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
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
	experienceID := r.PathValue("experienceId")

	if resumeId == "" {
		RespondWithError(w, http.StatusBadRequest, "Resume ID is required", "INVALID_REQUEST")
		return
	}

	if experienceID == "" {
		RespondWithError(w, http.StatusBadRequest, "Experience ID is required", "INVALID_REQUEST")
		return
	}

	resumeUUID, err := uuid.Parse(resumeId)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid resume ID", "INVALID_REQUEST")
		return
	}

	experienceUUID, err := uuid.Parse(experienceID)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid experience ID", "INVALID_REQUEST")
		return
	}

	resume, err := h.resumeRepo.GetCVById(ctx, resumeUUID)
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

	if err := h.resumeRepo.DeleteExperience(ctx, experienceUUID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			RespondWithError(w, http.StatusNotFound, "Experience entry not found", "NOT_FOUND")
			return
		}
		RespondWithError(w, http.StatusInternalServerError, "Failed to delete experience entry", "INTERNAL_SERVER_ERROR")
		return
	}

	RespondWithJSON(w, http.StatusOK, map[string]any{
		"message": "Experience entry deleted successfully",
	})
}

func (h *ResumeHandler) AddSkillHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
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

	resume, err := h.resumeRepo.GetCVById(ctx, resumeUUID)
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

	var skill domain.Skill
	if err := json.NewDecoder(r.Body).Decode(&skill); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST")
		return
	}

	if err := skill.Validate(); err != nil {
		RespondWithError(w, http.StatusBadRequest, err.Error(), "VALIDATION_ERROR")
		return
	}

	skill.BeforeSave()

	skillID, err := h.resumeRepo.AddSkill(ctx, resumeUUID, &skill)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to add skill", "INTERNAL_SERVER_ERROR")
		return
	}

	RespondWithJSON(w, http.StatusCreated, map[string]any{
		"id":      skillID,
		"message": "Skill added successfully",
	})
}

func (h *ResumeHandler) GetSkillsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
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

	resume, err := h.resumeRepo.GetCVById(ctx, resumeUUID)
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

	skills, err := h.resumeRepo.GetSkillsByCV(ctx, resumeUUID)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to get skills", "INTERNAL_SERVER_ERROR")
		return
	}

	RespondWithJSON(w, http.StatusOK, skills)
}

func (h *ResumeHandler) DeleteSkillHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
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
	skillID := r.PathValue("skillId")

	if resumeId == "" {
		RespondWithError(w, http.StatusBadRequest, "Resume ID is required", "INVALID_REQUEST")
		return
	}

	if skillID == "" {
		RespondWithError(w, http.StatusBadRequest, "Skill ID is required", "INVALID_REQUEST")
		return
	}

	resumeUUID, err := uuid.Parse(resumeId)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid resume ID", "INVALID_REQUEST")
		return
	}

	skillUUID, err := uuid.Parse(skillID)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid skill ID", "INVALID_REQUEST")

		return

	}

	resume, err := h.resumeRepo.GetCVById(ctx, resumeUUID)
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

	if err := h.resumeRepo.DeleteSkill(ctx, skillUUID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			RespondWithError(w, http.StatusNotFound, "Skill not found", "NOT_FOUND")
			return
		}
		RespondWithError(w, http.StatusInternalServerError, "Failed to delete skill", "INTERNAL_SERVER_ERROR")
		return
	}

	RespondWithJSON(w, http.StatusOK, map[string]any{
		"message": "Skill deleted successfully",
	})
}

func (h *ResumeHandler) AddProjectHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
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

	resume, err := h.resumeRepo.GetCVById(ctx, resumeUUID)
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

	var project domain.Project
	if err := json.NewDecoder(r.Body).Decode(&project); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST")
		return
	}

	if err := project.Validate(); err != nil {
		RespondWithError(w, http.StatusBadRequest, err.Error(), "VALIDATION_ERROR")
		return
	}

	project.BeforeSave()

	projectID, err := h.resumeRepo.AddProject(ctx, resumeUUID, &project)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to add project", "INTERNAL_SERVER_ERROR")
		return
	}

	RespondWithJSON(w, http.StatusCreated, map[string]any{
		"id":      projectID,
		"message": "Project added successfully",
	})
}

func (h *ResumeHandler) GetProjectsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
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

	resume, err := h.resumeRepo.GetCVById(ctx, resumeUUID)
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

	projects, err := h.resumeRepo.GetProjectByCV(ctx, resumeUUID)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to get projects", "INTERNAL_SERVER_ERROR")
		return
	}

	RespondWithJSON(w, http.StatusOK, projects)
}

func (h *ResumeHandler) DeleteProjectHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, err := GetClaimsFromContext(r.Context())
	if err != nil {
		RespondWithError(w, http.StatusUnauthorized, "Unauthorized", "UNAUTHORIZED")
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Invalid user ID", "INTERNAL_SERVER_ERROR")
		return
	}

	resumeId := r.PathValue("id")
	projectId := r.PathValue("projectId")

	if resumeId == "" {
		RespondWithError(w, http.StatusBadRequest, "Resume ID is required", "INVALID_REQUEST")
		return
	}

	if projectId == "" {
		RespondWithError(w, http.StatusBadRequest, "Project ID is required", "INVALID_REQUEST")
		return
	}
	resumeUUID, err := uuid.Parse(resumeId)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid resume ID", "INVALID_REQUEST")
		return
	}

	projectUUID, err := uuid.Parse(projectId)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid project ID", "INVALID_REQUEST")
		return
	}

	resume, err := h.resumeRepo.GetCVById(ctx, resumeUUID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			RespondWithError(w, http.StatusNotFound, "Resume not found", "NOT_FOUND")
			return
		}
		RespondWithError(w, http.StatusInternalServerError, "Failed to get resume", "INTERNAL_SERVER_ERROR")
		return
	}

	if resume.UserID != userID && claims.Role != "admin" {
		RespondWithError(w, http.StatusForbidden, "You don't have permission to update this resume", "FORBIDDEN")
		return
	}

	if err := h.resumeRepo.DeleteProject(ctx, projectUUID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			RespondWithError(w, http.StatusNotFound, "Project not found", "NOT_FOUND")
			return
		}
		RespondWithError(w, http.StatusInternalServerError, "Failed to delete project", "INTERNAL_SERVER_ERROR")
		return
	}

	RespondWithJSON(w, http.StatusOK, map[string]any{
		"message": "Project deleted successfully",
	})
}

func (h *ResumeHandler) AddCertificationHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
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

	resume, err := h.resumeRepo.GetCVById(ctx, resumeUUID)
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

	var certification domain.Certification
	if err := json.NewDecoder(r.Body).Decode(&certification); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST")
		return
	}

	if err := certification.Validate(); err != nil {
		RespondWithError(w, http.StatusBadRequest, err.Error(), "VALIDATION_ERROR")
		return
	}

	certification.BeforeSave()

	certificationID, err := h.resumeRepo.AddCertification(ctx, resumeUUID, &certification)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to add certification", "INTERNAL_SERVER_ERROR")
		return
	}

	RespondWithJSON(w, http.StatusCreated, map[string]any{
		"id":      certificationID,
		"message": "Certification added successfully",
	})
}

func (h *ResumeHandler) GetCertificationsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
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

	resume, err := h.resumeRepo.GetCVById(ctx, resumeUUID)
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

	certifications, err := h.resumeRepo.GetCertificationsByResume(ctx, resumeUUID)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to get certifications", "INTERNAL_SERVER_ERROR")
		return
	}

	RespondWithJSON(w, http.StatusOK, certifications)
}

func (h *ResumeHandler) DeleteCertificationHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
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
	certificationID := r.PathValue("certificationId")

	if resumeId == "" {
		RespondWithError(w, http.StatusBadRequest, "Resume ID is required", "INVALID_REQUEST")
		return
	}

	if certificationID == "" {
		RespondWithError(w, http.StatusBadRequest, "Certification ID is required", "INVALID_REQUEST")
		return
	}

	resumeUUID, err := uuid.Parse(resumeId)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid resume ID", "INVALID_REQUEST")
		return
	}

	certificationUUID, err := uuid.Parse(certificationID)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid certification ID", "INVALID_REQUEST")
		return
	}

	resume, err := h.resumeRepo.GetCVById(ctx, resumeUUID)
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

	if err := h.resumeRepo.DeleteCertification(ctx, certificationUUID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			RespondWithError(w, http.StatusNotFound, "Certification not found", "NOT_FOUND")
			return
		}
		RespondWithError(w, http.StatusInternalServerError, "Failed to delete certification", "INTERNAL_SERVER_ERROR")
		return
	}

	RespondWithJSON(w, http.StatusOK, map[string]any{
		"message": "Certification deleted successfully",
	})
}

func (h *ResumeHandler) GetPersonalInfoHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
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

	resume, err := h.resumeRepo.GetCVById(ctx, resumeUUID)
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

	personalInfo, err := h.resumeRepo.GetPersonalInfo(ctx, resumeUUID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			RespondWithJSON(w, http.StatusOK, nil)
			return
		}
		RespondWithError(w, http.StatusInternalServerError, "Failed to get personal info", "INTERNAL_SERVER_ERROR")
		return
	}

	RespondWithJSON(w, http.StatusOK, personalInfo)
}
