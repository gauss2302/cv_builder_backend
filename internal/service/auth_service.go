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
