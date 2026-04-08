package pipeline

import (
	"net/http"

	"isbudget/backend/internal/common"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type Handler struct {
	svc      *Service
	validate *validator.Validate
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc, validate: common.NewValidator()}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseProjectID(w, r)
	if !ok {
		return
	}
	items, err := h.svc.List(r.Context(), projectID)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to load allocation blocks")
		return
	}
	common.WriteJSON(w, http.StatusOK, items)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseProjectID(w, r)
	if !ok {
		return
	}
	var req upsertAllocationBlockRequest
	if !common.DecodeAndValidate(w, r, h.validate, &req) {
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	item, err := h.svc.Create(r.Context(), projectID, UpsertBlockInput{
		Name:                 req.Name,
		Type:                 req.Type,
		LevelIndex:           req.LevelIndex,
		Enabled:              enabled,
		ValueType:            req.ValueType,
		Value:                req.Value,
		AppliesTo:            req.AppliesTo,
		TeamDistributionMode: req.TeamDistributionMode,
	})
	if err != nil {
		h.writeMappedError(w, err, "create")
		return
	}
	common.WriteJSON(w, http.StatusCreated, item)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseProjectID(w, r)
	if !ok {
		return
	}
	id, ok := parseBlockID(w, r)
	if !ok {
		return
	}
	var req upsertAllocationBlockRequest
	if !common.DecodeAndValidate(w, r, h.validate, &req) {
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	item, err := h.svc.Update(r.Context(), projectID, id, UpsertBlockInput{
		Name:                 req.Name,
		Type:                 req.Type,
		LevelIndex:           req.LevelIndex,
		Enabled:              enabled,
		ValueType:            req.ValueType,
		Value:                req.Value,
		AppliesTo:            req.AppliesTo,
		TeamDistributionMode: req.TeamDistributionMode,
	})
	if err != nil {
		h.writeMappedError(w, err, "update")
		return
	}
	common.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseProjectID(w, r)
	if !ok {
		return
	}
	id, ok := parseBlockID(w, r)
	if !ok {
		return
	}
	if err := h.svc.Delete(r.Context(), projectID, id); err != nil {
		h.writeMappedError(w, err, "delete")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Reorder(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseProjectID(w, r)
	if !ok {
		return
	}
	var req reorderAllocationBlocksRequest
	if !common.DecodeAndValidate(w, r, h.validate, &req) {
		return
	}

	items := make([]ReorderItemInput, 0, len(req.Items))
	for _, raw := range req.Items {
		id, err := uuid.Parse(raw.ID)
		if err != nil {
			common.WriteError(w, http.StatusBadRequest, "items.id must be UUID")
			return
		}
		items = append(items, ReorderItemInput{
			ID:         id,
			LevelIndex: raw.LevelIndex,
			OrderIndex: raw.OrderIndex,
		})
	}

	orderedIDs := make([]uuid.UUID, 0, len(req.OrderedIDs))
	for _, raw := range req.OrderedIDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			common.WriteError(w, http.StatusBadRequest, "ordered_ids must be UUID")
			return
		}
		orderedIDs = append(orderedIDs, id)
	}

	if err := h.svc.Reorder(r.Context(), projectID, items, orderedIDs); err != nil {
		h.writeMappedError(w, err, "reorder")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Preview(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseProjectID(w, r)
	if !ok {
		return
	}
	var req previewAllocationRequest
	if !common.DecodeAndValidate(w, r, h.validate, &req) {
		return
	}

	item, err := h.svc.Preview(r.Context(), projectID, PreviewInput{GrossAmount: req.GrossAmount})
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to build allocation preview")
		return
	}
	common.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) writeMappedError(w http.ResponseWriter, err error, action string) {
	common.WriteMappedError(w, err, http.StatusInternalServerError, "allocation_pipeline_"+action+"_failed", "finance pipeline mutation failed",
		common.ErrorMapping{Err: ErrAllocationBlockNotFound, Status: http.StatusNotFound, Code: "not_found"},
		common.ErrorMapping{Err: ErrAllocationInvalidType, Status: http.StatusBadRequest, Code: "invalid_request"},
		common.ErrorMapping{Err: ErrAllocationInvalidValueType, Status: http.StatusBadRequest, Code: "invalid_request"},
		common.ErrorMapping{Err: ErrAllocationInvalidAppliesTo, Status: http.StatusBadRequest, Code: "invalid_request"},
		common.ErrorMapping{Err: ErrAllocationInvalidValue, Status: http.StatusBadRequest, Code: "invalid_request"},
		common.ErrorMapping{Err: ErrAllocationInvalidOrderPayload, Status: http.StatusBadRequest, Code: "invalid_request"},
		common.ErrorMapping{Err: ErrAllocationInvalidLevelIndex, Status: http.StatusBadRequest, Code: "invalid_request"},
		common.ErrorMapping{Err: ErrAllocationInvalidTeamMode, Status: http.StatusBadRequest, Code: "invalid_request"},
		common.ErrorMapping{Err: ErrAllocationRemainderConflict, Status: http.StatusBadRequest, Code: "invalid_request"},
	)
}

func parseProjectID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	return common.ParseUUIDParam(w, r, "projectId", "project")
}

func parseBlockID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	return common.ParseUUIDParam(w, r, "id", "allocation block")
}
