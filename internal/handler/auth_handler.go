package handler

import (
	"cv_builder/config"
	"cv_builder/internal/service"
	"cv_builder/pkg/security"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	tgInitData "github.com/telegram-mini-apps/init-data-golang"
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
		Interval: time.Minute,
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

type TelegramRequest struct {
	InitData string `json:"initData"`
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
	ctx := r.Context()
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
	user, err := h.authService.Register(ctx, req.Email, req.Password, req.Role)
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

	ctx := r.Context()
	// Apply rate limiting
	if ok := h.applyRateLimit(w, r); !ok {
		return
	}

	// Parse request body
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST")
		return
	}

	// Validate request
	if err := h.validator.Struct(req); err != nil {
		var validationErrors validator.ValidationErrors
		errors.As(err, &validationErrors)
		RespondWithValidationError(w, validationErrors)
		return
	}

	// Get client info
	userAgent := r.UserAgent()
	clientIP := getClientIP(r)

	// Login user
	tokens, err := h.authService.Login(ctx, req.Email, req.Password, userAgent, clientIP)
	if err != nil {
		if errors.Is(err, errors.New("invalid credentials!!")) {
			// Return same error for invalid email or password to prevent user enumeration
			RespondWithError(w, http.StatusUnauthorized, "Invalid email or password", "INVALID_CREDENTIALS")
			return
		}
		log.Error().Err(err).Msg("Failed to login user")
		RespondWithError(w, http.StatusInternalServerError, "Failed to login user", "LOGIN_FAILED")
		return
	}

	// Return tokens
	RespondWithJSON(w, http.StatusOK, tokens)
}

func (h *AuthHandler) LogoutHandler(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()
	if ok := h.applyRateLimit(w, r); !ok {
		return
	}

	var req RefreshTokenRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, http.StatusBadRequest, "invalid req body", "INVALID_REQUEST")
		return
	}

	if err := h.validator.Struct(req); err != nil {
		validationErrors := err.(validator.ValidationErrors)
		RespondWithValidationError(w, validationErrors)
		return
	}

	if err := h.authService.Logout(ctx, req.RefreshToken); err != nil {
		log.Error().Err(err).Msg("failed to logout user")
		RespondWithError(w, http.StatusInternalServerError, "failed to logout user", "LOGOUT_FAILED")
		return
	}
	RespondWithJSON(w, http.StatusOK, map[string]any{
		"message": "user logged out success",
	})
}

func (h *AuthHandler) RefreshTokenHandler(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()
	if ok := h.applyRateLimit(w, r); !ok {
		return
	}

	var req RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, http.StatusBadRequest, "invalid body req", "INVALID_REQUEST")
		return
	}

	if err := h.validator.Struct(req); err != nil {
		validatonErrors := err.(validator.ValidationErrors)
		RespondWithValidationError(w, validatonErrors)
		return
	}

	userAgent := r.UserAgent()
	clientIP := getClientIP(r)

	tokens, err := h.authService.RefreshToken(ctx, req.RefreshToken, userAgent, clientIP)
	if err != nil {
		status := http.StatusUnauthorized
		code := "INVALID_TOKEN"
		message := "invalid refresh token"

		if errors.Is(err, errors.New("token expired")) {
			code = "TOKEN_EXPIRED"
			message = "Refresh token expired"
		} else if errors.Is(err, errors.New("invalid session")) {
			code = "INVALID_SESSION"
			message = "Invalid session"
		}

		RespondWithError(w, status, message, code)
		return
	}
	RespondWithJSON(w, http.StatusOK, tokens)
}

func (h *AuthHandler) RequestPasswordResetHandler(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()
	if ok := h.applyRateLimit(w, r); !ok {
		return
	}

	var req PasswordResetRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, http.StatusBadRequest, "invalid req body", "INVALID_REQUEST")
		return
	}

	if err := h.validator.Struct(req); err != nil {
		validaitonErrors := err.(validator.ValidationErrors)
		RespondWithValidationError(w, validaitonErrors)
		return
	}

	resetToken, err := h.authService.RequestPasswordReset(ctx, req.Email)
	if err != nil {
		if errors.Is(err, errors.New("user not found")) {
			RespondWithJSON(w, http.StatusOK, map[string]any{
				"message": "pwd reset instructons sent to email if exists",
			})
			return
		}
		log.Error().Err(err).Msg("failed to req pwd reset")
		RespondWithError(w, http.StatusInternalServerError, "failed to req pwd reset", "PASSWORD_RESET_FAILED")
		return
	}
	RespondWithJSON(w, http.StatusOK, map[string]any{
		"message": "pwd rest instructions sent",
		"token":   resetToken,
	})
}

func (h *AuthHandler) ResetPasswordHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if ok := h.applyRateLimit(w, r); !ok {
		return
	}

	var req PasswordResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, http.StatusBadRequest, "invalid req body", "INVALID_REQUEST")
		return
	}

	if err := h.validator.Struct(req); err != nil {
		validationErrors := err.(validator.ValidationErrors)
		RespondWithValidationError(w, validationErrors)
		return
	}

	if err := h.authService.ResetPassword(ctx, req.Token, req.NewPassword); err != nil {
		status := http.StatusBadRequest
		code := "PASSWORD_RESET_FAILED"
		message := "Failed to reset password"

		if errors.Is(err, errors.New("invalid token")) {
			code = "INVALID_TOKEN"
			message = "Invalid reset token"
		} else if errors.Is(err, errors.New("token expired")) {
			code = "TOKEN_EXPIRED"
			message = "Reset token expired"
		} else if errors.Is(err, errors.New("password reset already used")) {
			code = "TOKEN_USED"
			message = "Reset token already used"
		}

		RespondWithError(w, status, message, code)
		return
	}

	// Return success response
	RespondWithJSON(w, http.StatusOK, map[string]any{
		"message": "Password reset successfully",
	})
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

func (h *AuthHandler) LoginTelegram(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if ok := h.applyRateLimit(w, r); !ok {
		return
	}

	var req TelegramRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST")
		return
	}

	if req.InitData == "" {
		http.Error(w, "initData parameter is missing", http.StatusBadRequest)
		return
	}

	const MY_BOT_TOKEN = "7754477817:AAGYrJrfTfmaG-rE4Tv7gctnFU9qF4CXFYc"
	const INIT_DATA_EXPIRATION = 3 * time.Hour

	cfg, _ := config.Load()

	botToken := cfg.TelegramBotToken

	err := tgInitData.Validate(req.InitData, botToken, INIT_DATA_EXPIRATION)
	if err != nil {
		log.Printf("Validation error: %v\n", err)
		RespondWithError(w, http.StatusUnauthorized, "Failed to validate", "TG_VALIDATION_FAILED")
		return
	}

	parsedData, err := tgInitData.Parse(req.InitData)
	if err != nil {
		log.Printf("Parsing error: %v\n", err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to parse data", "TG_PARSE_ERROR")
		return
	}

	fmt.Println(parsedData) // For development logging
	RespondWithJSON(w, http.StatusOK, parsedData)
}
