package service

import (
	"context"
	"cv_builder/internal/domain"
	"cv_builder/internal/repository"
	"cv_builder/pkg/auth"
	"cv_builder/pkg/security"
	"errors"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"time"
)

var (
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrUserAlreadyExists    = errors.New("user already exists")
	ErrUserNotFound         = errors.New("user not found")
	ErrInvalidToken         = errors.New("invalid token")
	ErrExpiredToken         = errors.New("token expired")
	ErrInvalidSession       = errors.New("invalid session")
	ErrPasswordResetExpired = errors.New("password reset expired")
	ErrPasswordResetUsed    = errors.New("password reset already used")
)

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}
type AuthServiceConfig struct {
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
	ResetTokenExpiry   time.Duration
}

type AuthService struct {
	userRepo domain.UserRepository
	jwt      *auth.JWT
	config   AuthServiceConfig
}

func NewAuthService(userRepo domain.UserRepository, jwt *auth.JWT, config AuthServiceConfig) *AuthService {
	return &AuthService{
		userRepo: userRepo,
		jwt:      jwt,
		config:   config,
	}
}

func (s *AuthService) Register(ctx context.Context, email, password, role string) (*domain.User, error) {
	existingUser, err := s.userRepo.GetUserByEmail(ctx, email)
	if err == nil && existingUser != nil {
		return nil, ErrUserAlreadyExists
	} else if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	passwordHash, err := security.HashPassword(password, nil)
	if err != nil {
		log.Error().Err(err).Msg("failed to hash pwd")
		return nil, err
	}

	user := &domain.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: passwordHash,
		Role:         role,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.userRepo.CreateUser(ctx, user); err != nil {
		log.Error().Err(err).Msg("failed to create user")
		if errors.Is(err, repository.ErrConflict) {
			return nil, ErrUserAlreadyExists
		}
		return nil, err
	}

	return user, nil

}

func (s *AuthService) Login(ctx context.Context, email, password, userAgent, clientIP string) (*TokenPair, error) {
	user, err := s.userRepo.GetUserByEmail(ctx, email)

	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	verifyPassword, err := security.VerifyPassword(password, user.PasswordHash)

	if err != nil {
		log.Error().Err(err).Msg("failed to verify pwd")
		return nil, err
	}

	if !verifyPassword {
		return nil, ErrInvalidCredentials
	}

	accessToken, err := s.jwt.GenerateAccessToken(user.ID.String(), user.Email, user.Role)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate access token")
		return nil, err
	}

	refreshToken, err := s.jwt.GenerateRefreshToken(user.ID.String(), user.Email, user.Role)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate refresh token")
		return nil, err
	}

	session := &domain.Session{
		ID:           uuid.New(),
		UserID:       user.ID,
		RefreshToken: refreshToken,
		UserAgent:    userAgent,
		ClientIP:     clientIP,
		ExpiresAt:    time.Now().Add(s.config.RefreshTokenExpiry),
		CreatedAt:    time.Now(),
	}

	if err := s.userRepo.CreateSession(ctx, session); err != nil {
		log.Error().Err(err).Msg("Failed to create session")
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.config.AccessTokenExpiry.Seconds()),
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	session, err := s.userRepo.GetSessionByToken(ctx, refreshToken)

	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil
		}
		return err
	}

	return s.userRepo.DeleteSession(ctx, session.ID)
}

func (s *AuthService) LogoutAll(ctx context.Context, userId uuid.UUID) error {
	return s.userRepo.DeleteUserSessions(ctx, userId)
}

func (s *AuthService) RequestPasswordReset(ctx context.Context, email string) (string, error) {

	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return "", ErrUserNotFound
		}
		return "", err
	}

	// make a reset token
	resetToken, err := s.jwt.GenerateResetToken(user.ID.String(), user.Email)
	if err != nil {
		log.Error().Err(err).Msg("failed to gen reset token")
		return "", nil
	}

	//save updated version
	reset := &domain.PasswordReset{
		ID:        uuid.New(),
		UserID:    user.ID,
		Token:     resetToken,
		ExpiresAt: time.Now().Add(s.config.ResetTokenExpiry),
		CreatedAt: time.Now(),
	}

	if err := s.userRepo.CreatePasswordReset(ctx, reset); err != nil {
		log.Error().Err(err).Msg("failed to create pwd reset")
	}
	return resetToken, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken, userAgent, clientIp string) (*TokenPair, error) {
	claims, err := s.jwt.ValidateRefreshToken(refreshToken)

	if err != nil {
		if errors.Is(err, auth.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	userId, err := uuid.Parse(claims.UserID)
	if err != nil {
		log.Error().Err(err).Msg("invalid user id in token")
		return nil, ErrInvalidToken
	}

	user, err := s.userRepo.GetUserById(ctx, userId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	session, err := s.userRepo.GetSessionByToken(ctx, refreshToken)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInvalidSession
		}
		return nil, err
	}

	if time.Now().After(session.ExpiresAt) {
		_ = s.userRepo.DeleteSession(ctx, session.ID)
		return nil, ErrExpiredToken
	}

	newAccessToken, err := s.jwt.GenerateAccessToken(user.ID.String(), user.Email, user.Role)
	if err != nil {
		log.Error().Err(err).Msg("failed to gen access token")
		return nil, err
	}

	newRefreshToken, err := s.jwt.GenerateRefreshToken(user.ID.String(), user.Email, user.Role)
	if err != nil {
		log.Error().Err(err).Msg("failed to gen refresh token")
		return nil, err
	}

	if err := s.userRepo.DeleteSession(ctx, session.ID); err != nil {
		log.Error().Err(err).Msg("failed to delete old session")
	}

	newSession := &domain.Session{
		ID:           uuid.New(),
		UserID:       user.ID,
		RefreshToken: newRefreshToken,
		UserAgent:    userAgent,
		ClientIP:     clientIp,
		ExpiresAt:    time.Now().Add(s.config.ResetTokenExpiry),
		CreatedAt:    time.Now(),
	}

	if err := s.userRepo.CreateSession(ctx, newSession); err != nil {
		log.Error().Err(err).Msg("failed to create new session")
	}

	return &TokenPair{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int64(s.config.AccessTokenExpiry.Seconds()),
	}, nil

}

func (s *AuthService) ResetPassword(ctx context.Context, resetToken, newPassword string) error {
	claims, err := s.jwt.ValidateResetToken(resetToken)
	if err != nil {
		if errors.Is(err, auth.ErrTokenExpired) {
			return ErrExpiredToken
		}
		return ErrInvalidToken
	}

	userId, err := uuid.Parse(claims.UserID)
	if err != nil {
		log.Error().Err(err).Msg("Invalid user ID in token")
		return ErrInvalidToken
	}

	user, err := s.userRepo.GetUserById(ctx, userId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrUserNotFound
		}
		return err
	}

	reset, err := s.userRepo.GetPasswordResetByToken(ctx, resetToken)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrInvalidToken
		}
		return err
	}

	if time.Now().After(reset.ExpiresAt) {
		return ErrPasswordResetExpired
	}

	if !reset.UsedAt.IsZero() {
		return ErrPasswordResetUsed
	}

	passwordHash, err := security.HashPassword(newPassword, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to hash password")
		return err
	}

	user.PasswordHash = passwordHash
	user.UpdatedAt = time.Now()

	if err := s.userRepo.UpdateUser(ctx, user); err != nil {
		log.Error().Err(err).Msg("Failed to update user password")
		return err
	}

	if err := s.userRepo.MarkPasswordResetUsed(ctx, reset.ID); err != nil {
		log.Error().Err(err).Msg("Failed to mark password reset as used")
		// Continue anyway, just log the error
	}

	if err := s.userRepo.DeleteUserSessions(ctx, user.ID); err != nil {
		log.Error().Err(err).Msg("Failed to delete user sessions")
	}

	return nil

}

func (s *AuthService) ValidateAccessToken(accessToken string) (*auth.JWTClaims, error) {
	claims, err := s.jwt.ValidateAccessToken(accessToken)

	if err != nil {
		if errors.Is(err, auth.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}
	return claims, err
}
