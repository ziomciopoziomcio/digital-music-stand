package auth

import (
	"context"
	"fmt"
)

type contextKey string

const userIDKey contextKey = "user_id"

func UserIDFromContext(ctx context.Context) (uint, error) {
	userID, ok := ctx.Value(userIDKey).(uint)
	if !ok {
		return 0, fmt.Errorf("user id not found in context")
	}
	return userID, nil
}
