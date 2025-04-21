package handler

import (
	"context"
	"cv_builder/internal/service"
	"cv_builder/pkg/auth"
	"errors"
	"github.com/google/uuid"
	"net/http"
	"strings"
)

type contextKey int

const (
	userContextKey contextKey = iota
	claimsContentKey
)

type AuthMiddleware struct {
	authService *service.AuthService
}

func NewAuthMiddleware(authService *service.AuthService) *AuthMiddleware {
	return &AuthMiddleware{authService: authService}
}

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return xrip
	}

	ip := r.RemoteAddr
	if idx := strings.IndexByte(ip, ':'); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}
func GetUserIdFromContext(ctx context.Context) (uuid.UUID, error) {
	claims, ok := ctx.Value(claimsContentKey).(*auth.JWTClaims)

	if !ok {
		return uuid.Nil, errors.New("no clain,s in context")
	}

	userId, err := uuid.Parse(claims.UserID)
	if err != nil {
		return uuid.Nil, err
	}

	return userId, nil
}
