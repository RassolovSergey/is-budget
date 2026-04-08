package snapshots

import (
	"net/http"

	"isbudget/backend/internal/auth"
	"isbudget/backend/internal/common"

	"github.com/go-playground/validator/v10"
)

type Handler struct {
	svc      *Service
	validate *validator.Validate
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc, validate: common.NewValidator()}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	projectID, ok := common.ParseUUIDParam(w, r, "projectId", "project")
	if !ok {
		return
	}
	var req createSnapshotRequest
	if !common.DecodeAndValidate(w, r, h.validate, &req) {
		return
	}
	item, err := h.svc.Create(r.Context(), user.ID, projectID, req.GrossAmount)
	if err != nil {
		common.WriteMappedError(w, err, http.StatusInternalServerError, "payout_snapshot_create_failed", "failed to create payout snapshot",
			common.ErrorMapping{Err: ErrSnapshotCreateDenied, Status: http.StatusForbidden, Code: "forbidden"},
		)
		return
	}
	common.WriteJSON(w, http.StatusCreated, item)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	projectID, ok := common.ParseUUIDParam(w, r, "projectId", "project")
	if !ok {
		return
	}
	items, err := h.svc.List(r.Context(), user.ID, projectID)
	if err != nil {
		common.WriteMappedError(w, err, http.StatusInternalServerError, "payout_snapshot_list_failed", "failed to list payout snapshots",
			common.ErrorMapping{Err: ErrSnapshotProjectReadDenied, Status: http.StatusForbidden, Code: "forbidden"},
		)
		return
	}
	common.WriteJSON(w, http.StatusOK, items)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	snapshotID, ok := common.ParseUUIDParam(w, r, "id", "payout snapshot")
	if !ok {
		return
	}
	item, err := h.svc.Get(r.Context(), user.ID, snapshotID)
	if err != nil {
		common.WriteMappedError(w, err, http.StatusInternalServerError, "payout_snapshot_get_failed", "failed to load payout snapshot",
			common.ErrorMapping{Err: ErrSnapshotProjectReadDenied, Status: http.StatusForbidden, Code: "forbidden"},
			common.ErrorMapping{Err: ErrSnapshotNotFound, Status: http.StatusNotFound, Code: "not_found"},
		)
		return
	}
	common.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) UpdateSettlement(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	snapshotID, ok := common.ParseUUIDParam(w, r, "id", "payout snapshot")
	if !ok {
		return
	}
	var req updateSettlementRequest
	if !common.DecodeAndValidate(w, r, h.validate, &req) {
		return
	}
	item, err := h.svc.UpdateSettlement(r.Context(), user.ID, snapshotID, req.SettlementStatus, req.SettlementNote)
	if err != nil {
		common.WriteMappedError(w, err, http.StatusInternalServerError, "payout_snapshot_settlement_failed", "failed to update payout snapshot settlement",
			common.ErrorMapping{Err: ErrSnapshotUpdateDenied, Status: http.StatusForbidden, Code: "forbidden"},
			common.ErrorMapping{Err: ErrSnapshotNotFound, Status: http.StatusNotFound, Code: "not_found"},
			common.ErrorMapping{Err: ErrSnapshotSettlementStatus, Status: http.StatusBadRequest, Code: "invalid_settlement_status"},
			common.ErrorMapping{Err: ErrSnapshotSettlementMove, Status: http.StatusBadRequest, Code: "invalid_settlement_transition"},
		)
		return
	}
	common.WriteJSON(w, http.StatusOK, item)
}
