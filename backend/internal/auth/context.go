package auth

import (
	"context"

	"github.com/google/uuid"
)

type contextKey string

const userContextKey contextKey = "auth_user"

type AuthUser struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
}

func WithUser(ctx context.Context, user AuthUser) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func UserFromContext(ctx context.Context) (AuthUser, bool) {
	v := ctx.Value(userContextKey)
	if v == nil {
		return AuthUser{}, false
	}
	u, ok := v.(AuthUser)
	return u, ok
}
