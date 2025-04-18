package auth

import (
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog/log"
	"time"
)

const (
	// TokenTypeAccess is the token type for access tokens
	TokenTypeAccess = "access"
	// TokenTypeRefresh is the token type for refresh tokens
	TokenTypeRefresh = "refresh"
	// TokenTypeReset is the token type for password reset tokens
	TokenTypeReset = "reset"
)

var (
	ErrTokenExpired   = errors.New("token expired")
	ErrInvalidToken   = errors.New("invalid token")
	ErrWrongTokenType = errors.New("wrong token type")
)

type JWTClaims struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

type JWTConfig struct {
	Secret             string
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
	ResetTokenExpiry   time.Duration
	Issuer             string
	Audience           string
}

func DefaultJWTConfig() JWTConfig {
	return JWTConfig{
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		ResetTokenExpiry:   1 * time.Hour,
		Issuer:             "cv_builder",
		Audience:           "cv_builder_users",
	}
}

type JWT struct {
	config JWTConfig
}

func NewJWT(config JWTConfig) *JWT {
	if config.Secret == "" {
		panic("JWT secret is required")
	}
	if config.AccessTokenExpiry == 0 {
		config.AccessTokenExpiry = DefaultJWTConfig().AccessTokenExpiry
	}
	if config.RefreshTokenExpiry == 0 {
		config.RefreshTokenExpiry = DefaultJWTConfig().RefreshTokenExpiry
	}
	if config.ResetTokenExpiry == 0 {
		config.ResetTokenExpiry = DefaultJWTConfig().ResetTokenExpiry
	}
	if config.Issuer == "" {
		config.Issuer = DefaultJWTConfig().Issuer
	}
	if config.Audience == "" {
		config.Audience = DefaultJWTConfig().Audience
	}
	return &JWT{config: config}
}

func (j *JWT) GenerateAccessToken(userId, email, role string) (string, error) {

	return j.generateToken(userId, email, role, TokenTypeAccess, j.config.AccessTokenExpiry)
}

func (j *JWT) GenerateRefreshToken(userId, email, role string) (string, error) {
	return j.generateToken(userId, email, role, TokenTypeRefresh, j.config.RefreshTokenExpiry)
}

// token helper
func (j *JWT) generateToken(userId, email, role, tokenType string, expiry time.Duration) (string, error) {
	now := time.Now()
	expiresAt := now.Add(expiry)

	claims := JWTClaims{
		UserID:    userId,
		Email:     email,
		Role:      role,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.config.Issuer,
			Subject:   userId,
			Audience:  []string{j.config.Audience},
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(j.config.Secret))
	if err != nil {
		log.Error().Err(err).Msg("failed to sign JWT token")
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return signedToken, nil
}
