package projects

import (
	"net/http"
	"strings"

	"isbudget/backend/internal/auth"
	"isbudget/backend/internal/common"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Handler struct {
	svc      *Service
	validate *validator.Validate
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc, validate: common.NewValidator()}
}

func (h *Handler) ListProjects(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	items, err := h.svc.ListProjects(r.Context(), user.ID)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to list projects")
		return
	}
	common.WriteJSON(w, http.StatusOK, items)
}

func (h *Handler) ListCategoryFolders(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	items, err := h.svc.ListCategoryFolders(r.Context(), user.ID)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to list category folders")
		return
	}
	common.WriteJSON(w, http.StatusOK, items)
}

func (h *Handler) CreateProject(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req projectRequest
	if !common.DecodeAndValidate(w, r, h.validate, &req) {
		return
	}
	item, err := h.svc.CreateProject(r.Context(), user.ID, req.Name, req.Description, req.DurationDays, req.CurrencyCode, req.SecondaryCurrencyCode, req.Category)
	if err != nil {
		common.WriteMappedError(w, err, http.StatusInternalServerError, "create_project_failed", "failed to create project",
			common.ErrorMapping{Err: ErrInvalidProjectDuration, Status: http.StatusBadRequest, Code: "invalid_request"},
			common.ErrorMapping{Err: ErrInvalidProjectCurrency, Status: http.StatusBadRequest, Code: "invalid_request"},
			common.ErrorMapping{Err: ErrInvalidSecondaryProjectCurrency, Status: http.StatusBadRequest, Code: "invalid_request"},
			common.ErrorMapping{Err: ErrSecondaryCurrencyMustDiffer, Status: http.StatusBadRequest, Code: "invalid_request"},
			common.ErrorMapping{Err: ErrInvalidProjectCategory, Status: http.StatusBadRequest, Code: "invalid_request"},
		)
		return
	}
	common.WriteJSON(w, http.StatusCreated, item)
}

func (h *Handler) GetProject(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseProjectID(w, r)
	if !ok {
		return
	}
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	item, err := h.svc.GetProject(r.Context(), projectID)
	if err != nil {
		if IsNotFound(err) {
			common.WriteError(w, http.StatusNotFound, "project not found")
			return
		}
		common.WriteError(w, http.StatusInternalServerError, "failed to load project")
		return
	}
	role, _ := h.svc.GetMemberRole(r.Context(), projectID, user.ID)
	common.WriteJSON(w, http.StatusOK, projectWithRoleResponse{
		Project:         item,
		CurrentUserRole: role,
	})
}

func (h *Handler) UpdateProject(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseProjectID(w, r)
	if !ok {
		return
	}
	var req projectRequest
	if !common.DecodeAndValidate(w, r, h.validate, &req) {
		return
	}
	item, err := h.svc.UpdateProject(r.Context(), projectID, req.Name, req.Description, req.DurationDays, req.CurrencyCode, req.SecondaryCurrencyCode, req.Category)
	if err != nil {
		if IsNotFound(err) {
			common.WriteErrorWithCode(w, http.StatusNotFound, "not_found", "project not found")
			return
		}
		common.WriteMappedError(w, err, http.StatusInternalServerError, "update_project_failed", "failed to update project",
			common.ErrorMapping{Err: ErrInvalidProjectDuration, Status: http.StatusBadRequest, Code: "invalid_request"},
			common.ErrorMapping{Err: ErrInvalidProjectCurrency, Status: http.StatusBadRequest, Code: "invalid_request"},
			common.ErrorMapping{Err: ErrInvalidSecondaryProjectCurrency, Status: http.StatusBadRequest, Code: "invalid_request"},
			common.ErrorMapping{Err: ErrSecondaryCurrencyMustDiffer, Status: http.StatusBadRequest, Code: "invalid_request"},
			common.ErrorMapping{Err: ErrInvalidProjectCategory, Status: http.StatusBadRequest, Code: "invalid_request"},
			common.ErrorMapping{Err: ErrInvalidProjectStatus, Status: http.StatusBadRequest, Code: "invalid_request"},
			common.ErrorMapping{Err: ErrProjectDurationEditOnlyInPreparation, Status: http.StatusBadRequest, Code: "invalid_request"},
		)
		return
	}
	common.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) ActivateProject(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseProjectID(w, r)
	if !ok {
		return
	}

	item, err := h.svc.ActivateProject(r.Context(), projectID)
	if err != nil {
		if IsNotFound(err) {
			common.WriteErrorWithCode(w, http.StatusNotFound, "not_found", "project not found")
			return
		}
		common.WriteMappedError(w, err, http.StatusInternalServerError, "activate_project_failed", "failed to activate project",
			common.ErrorMapping{Err: ErrInvalidProjectStatus, Status: http.StatusBadRequest, Code: "invalid_request"},
			common.ErrorMapping{Err: ErrProjectActivationOnlyInPreparation, Status: http.StatusBadRequest, Code: "invalid_request"},
		)
		return
	}

	common.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseProjectID(w, r)
	if !ok {
		return
	}
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	role, err := h.svc.GetMemberRole(r.Context(), projectID, user.ID)
	if err != nil {
		if IsNotFound(err) {
			common.WriteError(w, http.StatusNotFound, "project member not found")
			return
		}
		common.WriteError(w, http.StatusInternalServerError, "failed to check member role")
		return
	}
	if role != "PM" {
		common.WriteError(w, http.StatusForbidden, "only project administrator can delete project")
		return
	}
	if err := h.svc.DeleteProject(r.Context(), projectID); err != nil {
		if IsNotFound(err) {
			common.WriteError(w, http.StatusNotFound, "project not found")
			return
		}
		common.WriteError(w, http.StatusInternalServerError, "failed to delete project")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) LeaveProject(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseProjectID(w, r)
	if !ok {
		return
	}
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err := h.svc.LeaveProject(r.Context(), projectID, user.ID); err != nil {
		common.WriteMappedError(w, err, http.StatusInternalServerError, "leave_project_failed", "failed to leave project",
			common.ErrorMapping{Err: ErrPMCannotLeaveProject, Status: http.StatusBadRequest, Code: "invalid_request", Message: ErrPMCannotLeaveProject.Error()},
			common.ErrorMapping{Err: ErrLastPM, Status: http.StatusBadRequest, Code: "invalid_request", Message: ErrLastPM.Error()},
			common.ErrorMapping{Err: ErrProjectAdministratorFixed, Status: http.StatusBadRequest, Code: "invalid_request", Message: ErrProjectAdministratorFixed.Error()},
			common.ErrorMapping{Err: pgx.ErrNoRows, Status: http.StatusNotFound, Code: "not_found", Message: "project member not found"},
		)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListMembers(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseProjectID(w, r)
	if !ok {
		return
	}
	items, err := h.svc.ListMembers(r.Context(), projectID)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to list members")
		return
	}
	common.WriteJSON(w, http.StatusOK, items)
}

func (h *Handler) AddMember(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseProjectID(w, r)
	if !ok {
		return
	}
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req addMemberRequest
	if !common.DecodeAndValidate(w, r, h.validate, &req) {
		return
	}

	if strings.TrimSpace(req.UserID) == "" && strings.TrimSpace(req.Email) == "" && strings.TrimSpace(req.Nickname) == "" {
		common.WriteError(w, http.StatusBadRequest, "provide user_id or email or nickname")
		return
	}

	var userID uuid.UUID
	var err error
	if strings.TrimSpace(req.UserID) != "" {
		userID, err = uuid.Parse(req.UserID)
		if err != nil {
			common.WriteError(w, http.StatusBadRequest, "invalid user_id")
			return
		}
	} else if strings.TrimSpace(req.Email) != "" {
		userID, err = h.svc.ResolveUserIDByEmail(r.Context(), req.Email)
		if err != nil {
			if IsNotFound(err) {
				common.WriteErrorWithCode(w, http.StatusNotFound, "not_found", "user not found")
				return
			}
			common.WriteErrorWithCode(w, http.StatusInternalServerError, "resolve_user_failed", "failed to resolve user by email")
			return
		}
	} else {
		userID, err = h.svc.ResolveUserIDByNickname(r.Context(), req.Nickname)
		if err != nil {
			if IsNotFound(err) {
				common.WriteErrorWithCode(w, http.StatusNotFound, "not_found", "user not found")
				return
			}
			common.WriteErrorWithCode(w, http.StatusInternalServerError, "resolve_user_failed", "failed to resolve user by nickname")
			return
		}
	}
	invite, err := h.svc.InviteMember(r.Context(), projectID, user.ID, userID, strings.ToUpper(req.Role), 7)
	if err != nil {
		common.WriteMappedError(w, err, http.StatusInternalServerError, "create_invite_failed", "failed to create invite",
			common.ErrorMapping{Err: ErrInvalidProjectRole, Status: http.StatusBadRequest, Code: "invalid_request", Message: ErrInvalidProjectRole.Error()},
			common.ErrorMapping{Err: ErrProjectAdministratorFixed, Status: http.StatusBadRequest, Code: "invalid_request", Message: ErrProjectAdministratorFixed.Error()},
			common.ErrorMapping{Err: ErrMemberAlreadyExists, Status: http.StatusConflict, Code: "conflict", Message: ErrMemberAlreadyExists.Error()},
			common.ErrorMapping{Err: ErrInviteAlreadyPending, Status: http.StatusConflict, Code: "conflict", Message: ErrInviteAlreadyPending.Error()},
			common.ErrorMapping{Err: pgx.ErrNoRows, Status: http.StatusNotFound, Code: "not_found", Message: "user not found"},
		)
		return
	}
	common.WriteJSON(w, http.StatusCreated, map[string]any{
		"status":          "pending",
		"invite_id":       invite.ID,
		"invited_user_id": invite.InvitedUserID,
		"expires_at":      invite.ExpiresAt,
	})
}

func (h *Handler) DeleteMember(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseProjectID(w, r)
	if !ok {
		return
	}
	memberID, err := uuid.Parse(chi.URLParam(r, "memberId"))
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid member id")
		return
	}
	if err := h.svc.DeleteMember(r.Context(), projectID, memberID); err != nil {
		common.WriteMappedError(w, err, http.StatusInternalServerError, "delete_member_failed", "failed to delete member",
			common.ErrorMapping{Err: ErrLastPM, Status: http.StatusBadRequest, Code: "invalid_request", Message: ErrLastPM.Error()},
			common.ErrorMapping{Err: ErrProjectAdministratorFixed, Status: http.StatusBadRequest, Code: "invalid_request", Message: ErrProjectAdministratorFixed.Error()},
			common.ErrorMapping{Err: pgx.ErrNoRows, Status: http.StatusNotFound, Code: "not_found", Message: "project or member not found"},
		)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListInvites(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseProjectID(w, r)
	if !ok {
		return
	}
	invites, err := h.svc.ListInvites(r.Context(), projectID)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to list invites")
		return
	}
	common.WriteJSON(w, http.StatusOK, invites)
}

func (h *Handler) RevokeInvite(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseProjectID(w, r)
	if !ok {
		return
	}
	inviteID, err := uuid.Parse(chi.URLParam(r, "inviteId"))
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid invite id")
		return
	}
	if err := h.svc.RevokeInvite(r.Context(), projectID, inviteID); err != nil {
		common.WriteMappedError(w, err, http.StatusInternalServerError, "revoke_invite_failed", "failed to revoke invite",
			common.ErrorMapping{Err: ErrInviteCannotBeRevoked, Status: http.StatusBadRequest, Code: "invalid_request", Message: ErrInviteCannotBeRevoked.Error()},
		)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListIncomingInvites(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	invites, err := h.svc.ListIncomingInvites(r.Context(), user.ID)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to list incoming invites")
		return
	}
	common.WriteJSON(w, http.StatusOK, invites)
}

func (h *Handler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	token := chi.URLParam(r, "token")
	result, err := h.svc.AcceptInvite(r.Context(), token, user.ID)
	if err != nil {
		common.WriteMappedError(w, err, http.StatusInternalServerError, "accept_invite_failed", "failed to accept invite",
			common.ErrorMapping{Err: ErrInviteNotFound, Status: http.StatusNotFound, Code: "not_found", Message: ErrInviteNotFound.Error()},
			common.ErrorMapping{Err: ErrInviteExpired, Status: http.StatusBadRequest, Code: "invalid_request", Message: ErrInviteExpired.Error()},
			common.ErrorMapping{Err: ErrInviteAlreadyAccepted, Status: http.StatusBadRequest, Code: "invalid_request", Message: ErrInviteAlreadyAccepted.Error()},
			common.ErrorMapping{Err: ErrInviteAlreadyDeclined, Status: http.StatusBadRequest, Code: "invalid_request", Message: ErrInviteAlreadyDeclined.Error()},
			common.ErrorMapping{Err: ErrInviteNotForUser, Status: http.StatusBadRequest, Code: "invalid_request", Message: ErrInviteNotForUser.Error()},
			common.ErrorMapping{Err: ErrProjectAdministratorFixed, Status: http.StatusBadRequest, Code: "invalid_request", Message: ErrProjectAdministratorFixed.Error()},
			common.ErrorMapping{Err: ErrInviteLinksDisabled, Status: http.StatusBadRequest, Code: "invalid_request", Message: ErrInviteLinksDisabled.Error()},
		)
		return
	}
	common.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) DeclineInvite(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	token := chi.URLParam(r, "token")
	if err := h.svc.DeclineInvite(r.Context(), token, user.ID); err != nil {
		common.WriteMappedError(w, err, http.StatusInternalServerError, "decline_invite_failed", "failed to decline invite",
			common.ErrorMapping{Err: ErrInviteNotFound, Status: http.StatusNotFound, Code: "not_found", Message: ErrInviteNotFound.Error()},
			common.ErrorMapping{Err: ErrInviteExpired, Status: http.StatusBadRequest, Code: "invalid_request", Message: ErrInviteExpired.Error()},
			common.ErrorMapping{Err: ErrInviteAlreadyAccepted, Status: http.StatusBadRequest, Code: "invalid_request", Message: ErrInviteAlreadyAccepted.Error()},
			common.ErrorMapping{Err: ErrInviteAlreadyDeclined, Status: http.StatusBadRequest, Code: "invalid_request", Message: ErrInviteAlreadyDeclined.Error()},
			common.ErrorMapping{Err: ErrInviteNotForUser, Status: http.StatusBadRequest, Code: "invalid_request", Message: ErrInviteNotForUser.Error()},
			common.ErrorMapping{Err: ErrInviteLinksDisabled, Status: http.StatusBadRequest, Code: "invalid_request", Message: ErrInviteLinksDisabled.Error()},
		)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseProjectID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	return common.ParseUUIDParam(w, r, "projectId", "project")
}
