package service

import (
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

func (s *AuthService) Register(email, password, role string) (*domain.User, error) {
	existingUser, err := s.userRepo.GetUserByEmail(email)
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

	if err := s.userRepo.CreateUser(user); err != nil {
		log.Error().Err(err).Msg("failed to create user")
		if errors.Is(err, repository.ErrConflict) {
			return nil, ErrUserAlreadyExists
		}
		return nil, err
	}

	return user, nil

}

func (s *AuthService) Login(email, password, userAgent, clientIP string) (*TokenPair, error) {
	user, err := s.userRepo.GetUserByEmail(email)

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
		log.Error().Err(err).Msg("failed to generate access token")
		return nil, err
	}

	refreshToken, err := s.jwt.GenerateRefreshToken(user.ID.String(), user.Email, user.Role)
	if err != nil {
		log.Error().Err(err).Msg("failed to generate refresh token")
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

	if err := s.userRepo.CreateSession(session); err != nil {
		log.Error().Err(err).Msg("failed to create session")
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.config.AccessTokenExpiry.Seconds()),
	}, nil
}

func (s *AuthService) Logout(refreshToken string) error {
	session, err := s.userRepo.GetSessionByToken(refreshToken)

	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil
		}
		return err
	}

	return s.userRepo.DeleteSession(session.ID)
}

func (s *AuthService) LogoutAll(userId uuid.UUID) error {
	return s.userRepo.DeleteUserSessions(userId)
}

func (s *AuthService) RequestPasswordReset(email string) (string, error) {

	user, err := s.userRepo.GetUserByEmail(email)
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

	if err := s.userRepo.CreatePasswordReset(reset); err != nil {
		log.Error().Err(err).Msg("failed to create pwd reset")
	}
	return resetToken, nil
}
