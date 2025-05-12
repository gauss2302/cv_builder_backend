package security

import "context"

type contextKey int

const (
	sessionContextKey contextKey = iota
)

func SetSessionContext(ctx context.Context, sessionData SessionData) context.Context {
	return context.WithValue(ctx, sessionContextKey, sessionData)
}

func GetSessionFromContext(ctx context.Context) (SessionData, bool) {
	sessionData, ok := ctx.Value(sessionContextKey).(SessionData)
	return sessionData, ok
}
