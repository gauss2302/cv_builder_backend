package security

import "context"

type contextKey int

const (
	sessionContextKey contextKey = iota
)

// SetSessionContext adds session data to the context
func SetSessionContext(ctx context.Context, sessionData SessionData) context.Context {
	return context.WithValue(ctx, sessionContextKey, sessionData)
}

// GetSessionFromContext retrieves session data from the context
func GetSessionFromContext(ctx context.Context) (SessionData, bool) {
	sessionData, ok := ctx.Value(sessionContextKey).(SessionData)
	return sessionData, ok
}

func IsAuthenticated(ctx context.Context) bool {
	sessionData, ok := GetSessionFromContext(ctx)
	return ok && sessionData.UserId != ""
}

func GetUserId(ctx context.Context) string {
	sessionData, ok := GetSessionFromContext(ctx)
	if !ok {
		return ""
	}
	return sessionData.UserId
}

func GetUserRole(ctx context.Context) string {
	sessionData, ok := GetSessionFromContext(ctx)
	if !ok {
		return ""
	}
	return sessionData.Role
}

func HasRole(ctx context.Context, role string) bool {
	sessionData, ok := GetSessionFromContext(ctx)
	return ok && sessionData.Role == role
}
