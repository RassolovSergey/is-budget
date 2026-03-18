package auth

import (
	"errors"
	"net/http"

	"isbudget/backend/internal/common"
	"isbudget/backend/internal/db/sqlc"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func RequireProjectPermission(q *sqlc.Queries, permission Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := UserFromContext(r.Context())
			if !ok {
				common.WriteError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			projectIDStr := chi.URLParam(r, "projectId")
			projectID, err := uuid.Parse(projectIDStr)
			if err != nil {
				common.WriteError(w, http.StatusBadRequest, "invalid project id")
				return
			}

			if err := q.CompleteProjectIfDue(r.Context(), projectID); err != nil {
				common.WriteError(w, http.StatusInternalServerError, "failed to refresh project status")
				return
			}

			member, err := q.GetProjectMemberByUser(r.Context(), sqlc.GetProjectMemberByUserParams{
				ProjectID: projectID,
				UserID:    user.ID,
			})
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					common.WriteError(w, http.StatusForbidden, "project access denied")
					return
				}
				common.WriteError(w, http.StatusInternalServerError, "failed to verify project access")
				return
			}

			if !IsAllowed(string(member.Role), permission) {
				common.WriteError(w, http.StatusForbidden, "insufficient role permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
