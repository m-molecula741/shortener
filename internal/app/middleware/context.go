package middleware

import "context"

type contextKey string

const userIDKey contextKey = "userID"

// SetUserIDToContext добавляет ID пользователя в контекст
func SetUserIDToContext(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// GetUserIDFromContext извлекает ID пользователя из контекста
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDKey).(string)
	return userID, ok
}
