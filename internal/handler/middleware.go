package handler

import (
	"context"
	"cv_builder/internal/service"
	"cv_builder/pkg/auth"
	"errors"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"net/http"
	"strings"
	"time"
)

type contextKey int

const (
	userContextKey contextKey = iota
	claimsContextKey
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
	claims, ok := ctx.Value(claimsContextKey).(*auth.JWTClaims)

	if !ok {
		return uuid.Nil, errors.New("no claims in context")
	}

	userId, err := uuid.Parse(claims.UserID)
	if err != nil {
		return uuid.Nil, err
	}

	return userId, nil
}

func GetClaimsFromContext(ctx context.Context) (*auth.JWTClaims, error) {
	claims, ok := ctx.Value(claimsContextKey).(*auth.JWTClaims)
	if !ok {
		return nil, errors.New("no claims in context")
	}
	return claims, nil
}

type SessionLogger struct{}

func NewSessionLogger() *SessionLogger {
	return &SessionLogger{}
}

func (l *SessionLogger) LogActivity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		var userId string
		var email string
		var role string

		if claims, ok := r.Context().Value(claimsContextKey).(auth.JWTClaims); ok {
			userId = claims.UserID
			email = claims.Email
			role = claims.Role
		}

		rw := &responseWriter{w, http.StatusOK}

		next.ServeHTTP(rw, r)

		duration := time.Since(startTime)

		log.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status", rw.statusCode).
			Str("user_id", userId).
			Str("email", email).
			Str("role", role).
			Str("ip", getClientIP(r)).
			Str("user_agent", r.UserAgent()).
			Dur("duration", duration).
			Msg("API request")

	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func extractTokenFromHeader(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("no auth header")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", errors.New("invalid auth header format")
	}
	return parts[1], nil
}

func (m *AuthMiddleware) AuthRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		token, err := extractTokenFromHeader(r)
		if err != nil {
			RespondWithError(w, http.StatusUnauthorized, "Unauthorized", "INVALID_TOKEN")
			return
		}

		// Validate token
		claims, err := m.authService.ValidateAccessToken(token)
		if err != nil {
			if errors.Is(err, errors.New("token expired")) {
				RespondWithError(w, http.StatusUnauthorized, "Token expired", "TOKEN_EXPIRED")
				return
			}
			RespondWithError(w, http.StatusUnauthorized, "Invalid token", "INVALID_TOKEN")
			return
		}

		// Add claims to context
		ctx := context.WithValue(r.Context(), claimsContextKey, claims)

		// Continue with the next handler
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *AuthMiddleware) RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value(claimsContextKey).(*auth.JWTClaims)
			if !ok {
				RespondWithError(w, http.StatusUnauthorized, "Not Authorized", "UNAUTHORIZED")
				return
			}

			if claims.Role != role {
				RespondWithError(w, http.StatusForbidden, "Forbidden", "FORBIDDEN")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
