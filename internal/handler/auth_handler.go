package handler

import (
	"cv_builder/internal/service"
	"cv_builder/pkg/security"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"net/http"
	"time"
)

type AuthHandler struct {
	authService *service.AuthService
	validator   *validator.Validate
	rateLimiter *security.RateLimiter
}

func NewAuthHandler(authService *service.AuthService, redisClient *redis.Client) *AuthHandler {
	rateLimiterConfig := security.RateLimiterConfig{
		Redis:    redisClient,
		Limit:    100,
		Internal: time.Minute,
	}
	return &AuthHandler{
		authService: authService,
		validator:   validator.New(),
		rateLimiter: security.NewRateLimiter(rateLimiterConfig),
	}
}

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=100"`
	Role     string `json:"role" validate:"omitempty,oneof=user admin"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type PasswordResetRequestRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type PasswordResetRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8,max=100"`
}

func (h *AuthHandler) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	// Apply rate limiting
	if ok := h.applyRateLimit(w, r); !ok {
		return
	}

	// Parse request body
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST")
		return
	}

	// Validate request
	if err := h.validator.Struct(req); err != nil {
		validationErrors := err.(validator.ValidationErrors)
		RespondWithValidationError(w, validationErrors)
		return
	}

	// Set default role if not provided
	if req.Role == "" {
		req.Role = "user"
	}

	// Register user
	user, err := h.authService.Register(req.Email, req.Password, req.Role)
	if err != nil {
		if errors.Is(err, errors.New("user already exists")) {
			RespondWithError(w, http.StatusConflict, "User with this email already exists", "USER_EXISTS")
			return
		}
		log.Error().Err(err).Msg("Failed to register user")
		RespondWithError(w, http.StatusInternalServerError, "Failed to register user", "REGISTRATION_FAILED")
		return
	}

	// Return success response
	RespondWithJSON(w, http.StatusCreated, map[string]any{
		"message": "User registered successfully",
		"user_id": user.ID,
	})
}

func (h *AuthHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	if ok := h.applyRateLimit(w, r); !ok {
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, http.StatusBadRequest, "invalid request body", "IVALID_REQUEST")
		return
	}

	if err := h.validator.Struct(req); err != nil {
		validationErrors := err.(validator.ValidationErrors)
		RespondWithValidationError(w, validationErrors)
		return
	}

	userAgent := r.UserAgent()
	clientIP := getClientIP(r)

	tokens, err := h.authService.Login(req.Email, req.Password, userAgent, clientIP)

	if err != nil {
		if errors.Is(err, errors.New("invalid credentials")) {
			RespondWithError(w, http.StatusUnauthorized, "invalid email or pwd", "LOGIN_FAILED")
			return
		}
		log.Error().Err(err).Msg("failed to login user")
		RespondWithError(w, http.StatusInternalServerError, "failed to login user", "LOGIN_FAILED")
		return
	}
	RespondWithJSON(w, http.StatusOK, tokens)
}

func (h *AuthHandler) applyRateLimit(w http.ResponseWriter, r *http.Request) bool {
	count, err := h.rateLimiter.CheckRateLimit(r.Context(), r)
	if err != nil {
		if errors.Is(err, security.ErrRateLimitExceeded) {
			RespondWithError(w, http.StatusTooManyRequests, "rate limit exceeded", "RATE_LIM_EXCEEDED")
			return false
		}
		log.Error().Err(err).Msg("rate limit error")
	}
	w.Header().Set("X-RateLimit-Limit", "100")
	remaining := max(100-count, 0)
	w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
	w.Header().Set("X-RateLimit-Reset", "60")

	return true
}
