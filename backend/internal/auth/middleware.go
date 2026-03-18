package auth

import (
	"net/http"
	"strings"

	"isbudget/backend/internal/common"
)

func Middleware(manager *JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			head := r.Header.Get("Authorization")
			if head == "" {
				common.WriteError(w, http.StatusUnauthorized, "missing authorization header")
				return
			}

			parts := strings.SplitN(head, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				common.WriteError(w, http.StatusUnauthorized, "invalid authorization header")
				return
			}

			claims, err := manager.Parse(parts[1])
			if err != nil {
				common.WriteError(w, http.StatusUnauthorized, "invalid token")
				return
			}

			ctx := WithUser(r.Context(), AuthUser{ID: claims.UserID, Email: claims.Email})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
