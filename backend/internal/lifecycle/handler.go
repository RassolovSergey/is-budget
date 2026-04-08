package lifecycle

import (
	"net/http"
	"time"

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

func (h *Handler) GetConfig(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	projectID, ok := common.ParseUUIDParam(w, r, "projectId", "project")
	if !ok {
		return
	}
	item, err := h.svc.GetConfig(r.Context(), user.ID, projectID)
	if err != nil {
		common.WriteMappedError(w, err, http.StatusInternalServerError, "campaign_config_get_failed", "failed to load campaign config",
			common.ErrorMapping{Err: ErrLifecycleReadDenied, Status: http.StatusForbidden, Code: "forbidden"},
		)
		return
	}
	common.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	projectID, ok := common.ParseUUIDParam(w, r, "projectId", "project")
	if !ok {
		return
	}
	var req updateCampaignConfigRequest
	if !common.DecodeAndValidate(w, r, h.validate, &req) {
		return
	}
	var launchAt *time.Time
	if req.LaunchAt != nil && *req.LaunchAt != "" {
		parsed, err := time.Parse("2006-01-02T15:04", *req.LaunchAt)
		if err != nil {
			common.WriteError(w, http.StatusBadRequest, "launch_at must be in YYYY-MM-DDTHH:MM format")
			return
		}
		utc := parsed.UTC()
		launchAt = &utc
	}
	item, err := h.svc.UpdateConfig(r.Context(), user.ID, projectID, req.CurrentStage, launchAt, req.LatePledgesEnabled, req.PledgeOverTimeEnabled, req.PledgeManagerEnabled, req.DigitalOnlyFulfillment)
	if err != nil {
		common.WriteMappedError(w, err, http.StatusInternalServerError, "campaign_config_update_failed", "failed to update campaign config",
			common.ErrorMapping{Err: ErrLifecycleWriteDenied, Status: http.StatusForbidden, Code: "forbidden"},
			common.ErrorMapping{Err: ErrInvalidLifecycleStage, Status: http.StatusBadRequest, Code: "invalid_request"},
		)
		return
	}
	common.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) CreateSnapshot(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	projectID, ok := common.ParseUUIDParam(w, r, "projectId", "project")
	if !ok {
		return
	}
	var req createCampaignStageSnapshotRequest
	if !common.DecodeAndValidate(w, r, h.validate, &req) {
		return
	}
	item, err := h.svc.CreateSnapshot(r.Context(), user.ID, projectID, req.Stage, req.FollowersCount, req.Notes, req.GrossAmount)
	if err != nil {
		common.WriteMappedError(w, err, http.StatusInternalServerError, "campaign_stage_snapshot_create_failed", "failed to create campaign stage snapshot",
			common.ErrorMapping{Err: ErrLifecycleWriteDenied, Status: http.StatusForbidden, Code: "forbidden"},
			common.ErrorMapping{Err: ErrInvalidLifecycleStage, Status: http.StatusBadRequest, Code: "invalid_request"},
		)
		return
	}
	common.WriteJSON(w, http.StatusCreated, item)
}

func (h *Handler) ListSnapshots(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	projectID, ok := common.ParseUUIDParam(w, r, "projectId", "project")
	if !ok {
		return
	}
	items, err := h.svc.ListSnapshots(r.Context(), user.ID, projectID)
	if err != nil {
		common.WriteMappedError(w, err, http.StatusInternalServerError, "campaign_stage_snapshot_list_failed", "failed to list campaign stage snapshots",
			common.ErrorMapping{Err: ErrLifecycleReadDenied, Status: http.StatusForbidden, Code: "forbidden"},
		)
		return
	}
	common.WriteJSON(w, http.StatusOK, items)
}

func (h *Handler) GetSnapshot(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	snapshotID, ok := common.ParseUUIDParam(w, r, "id", "campaign stage snapshot")
	if !ok {
		return
	}
	item, err := h.svc.GetSnapshot(r.Context(), user.ID, snapshotID)
	if err != nil {
		common.WriteMappedError(w, err, http.StatusInternalServerError, "campaign_stage_snapshot_get_failed", "failed to load campaign stage snapshot",
			common.ErrorMapping{Err: ErrLifecycleReadDenied, Status: http.StatusForbidden, Code: "forbidden"},
			common.ErrorMapping{Err: ErrLifecycleSnapshotNotFound, Status: http.StatusNotFound, Code: "not_found"},
		)
		return
	}
	common.WriteJSON(w, http.StatusOK, item)
}
