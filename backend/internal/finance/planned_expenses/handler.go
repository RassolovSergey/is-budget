package planned_expenses

import (
	"net/http"

	"isbudget/backend/internal/common"
	"isbudget/backend/internal/db/sqlc"

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
	projectID, ok := projectIDFromPath(w, r)
	if !ok {
		return
	}
	items, err := h.svc.List(r.Context(), projectID)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to load planned expenses")
		return
	}
	common.WriteJSON(w, http.StatusOK, items)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	projectID, ok := projectIDFromPath(w, r)
	if !ok {
		return
	}
	var req upsertRequest
	if !common.DecodeAndValidate(w, r, h.validate, &req) {
		return
	}
	dueDate, ok := common.ParseDateValue(w, req.DueDate, "due_date")
	if !ok {
		return
	}
	item, err := h.svc.Create(r.Context(), sqlc.CreatePlannedExpenseParams{
		ProjectID:   projectID,
		Category:    req.Category,
		Amount:      req.Amount,
		DueDate:     dueDate,
		IsMandatory: req.IsMandatory,
		Priority:    req.Priority,
		Note:        req.Note,
	})
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to create planned expense")
		return
	}
	common.WriteJSON(w, http.StatusCreated, item)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	projectID, ok := projectIDFromPath(w, r)
	if !ok {
		return
	}
	id, ok := common.ParseUUIDParam(w, r, "id", "planned expense")
	if !ok {
		return
	}
	var req upsertRequest
	if !common.DecodeAndValidate(w, r, h.validate, &req) {
		return
	}
	dueDate, ok := common.ParseDateValue(w, req.DueDate, "due_date")
	if !ok {
		return
	}
	item, err := h.svc.Update(r.Context(), sqlc.UpdatePlannedExpenseParams{
		ProjectID:   projectID,
		ID:          id,
		Category:    req.Category,
		Amount:      req.Amount,
		DueDate:     dueDate,
		IsMandatory: req.IsMandatory,
		Priority:    req.Priority,
		Note:        req.Note,
	})
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to update planned expense")
		return
	}
	common.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	projectID, ok := projectIDFromPath(w, r)
	if !ok {
		return
	}
	id, ok := common.ParseUUIDParam(w, r, "id", "planned expense")
	if !ok {
		return
	}
	if err := h.svc.Delete(r.Context(), sqlc.DeletePlannedExpenseParams{ProjectID: projectID, ID: id}); err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to delete planned expense")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func projectIDFromPath(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	return common.ParseUUIDParam(w, r, "projectId", "project")
}
