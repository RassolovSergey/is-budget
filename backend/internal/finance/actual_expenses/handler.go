package actual_expenses

import (
	"net/http"
	"strings"
	"time"

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
	projectID, ok := parseProjectID(w, r)
	if !ok {
		return
	}
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	if fromStr == "" || toStr == "" {
		items, err := h.svc.ListAll(r.Context(), projectID)
		if err != nil {
			common.WriteError(w, http.StatusInternalServerError, "failed to load actual expenses")
			return
		}
		common.WriteJSON(w, http.StatusOK, items)
		return
	}

	from, ok := common.ParseDateValue(w, fromStr, "from")
	if !ok {
		return
	}
	to, ok := common.ParseDateValue(w, toStr, "to")
	if !ok {
		return
	}

	items, err := h.svc.ListRange(r.Context(), projectID, from, to)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to load actual expenses")
		return
	}
	common.WriteJSON(w, http.StatusOK, items)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseProjectID(w, r)
	if !ok {
		return
	}
	var req upsertRequest
	if !common.DecodeAndValidate(w, r, h.validate, &req) {
		return
	}
	paidAt, plannedExpenseID, ok := parseExpenseRefs(w, req)
	if !ok {
		return
	}
	item, err := h.svc.Create(r.Context(), sqlc.CreateActualExpenseParams{
		ProjectID:        projectID,
		PlannedExpenseID: plannedExpenseID,
		Title:            req.Title,
		Category:         req.Category,
		Amount:           req.Amount,
		PaidAt:           paidAt,
		Note:             req.Note,
	})
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to create actual expense")
		return
	}
	common.WriteJSON(w, http.StatusCreated, item)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseProjectID(w, r)
	if !ok {
		return
	}
	id, ok := common.ParseUUIDParam(w, r, "id", "actual expense")
	if !ok {
		return
	}
	var req upsertRequest
	if !common.DecodeAndValidate(w, r, h.validate, &req) {
		return
	}
	paidAt, plannedExpenseID, ok := parseExpenseRefs(w, req)
	if !ok {
		return
	}
	item, err := h.svc.Update(r.Context(), sqlc.UpdateActualExpenseParams{
		ProjectID:        projectID,
		ID:               id,
		PlannedExpenseID: plannedExpenseID,
		Title:            req.Title,
		Category:         req.Category,
		Amount:           req.Amount,
		PaidAt:           paidAt,
		Note:             req.Note,
	})
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to update actual expense")
		return
	}
	common.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseProjectID(w, r)
	if !ok {
		return
	}
	id, ok := common.ParseUUIDParam(w, r, "id", "actual expense")
	if !ok {
		return
	}
	if err := h.svc.Delete(r.Context(), sqlc.DeleteActualExpenseParams{ProjectID: projectID, ID: id}); err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to delete actual expense")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseProjectID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	return common.ParseUUIDParam(w, r, "projectId", "project")
}

func parseExpenseRefs(w http.ResponseWriter, req upsertRequest) (time.Time, *uuid.UUID, bool) {
	paidAt, ok := common.ParseDateValue(w, req.PaidAt, "paid_at")
	if !ok {
		return time.Time{}, nil, false
	}
	if strings.TrimSpace(req.PlannedExpenseID) == "" {
		return paidAt, nil, true
	}
	id, ok := common.ParseUUIDValue(w, req.PlannedExpenseID, "planned_expense_id")
	if !ok {
		return time.Time{}, nil, false
	}
	plannedExpenseID := &id
	return paidAt, plannedExpenseID, true
}
