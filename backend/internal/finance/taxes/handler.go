package taxes

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
	projectID, ok := parseProjectID(w, r)
	if !ok {
		return
	}
	items, err := h.svc.List(r.Context(), projectID)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to load tax rules")
		return
	}
	common.WriteJSON(w, http.StatusOK, items)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseProjectID(w, r)
	if !ok {
		return
	}
	var req upsertTaxRuleRequest
	if !common.DecodeAndValidate(w, r, h.validate, &req) {
		return
	}

	item, err := h.svc.Create(r.Context(), sqlc.CreateTaxRuleParams{
		ProjectID: projectID,
		Name:      req.Name,
		Percent:   req.Percent,
		Level:     req.Level,
	})
	if err != nil {
		common.WriteMappedError(w, err, http.StatusInternalServerError, "create_tax_rule_failed", "failed to create tax rule",
			common.ErrorMapping{Err: ErrInvalidTaxLevel, Status: http.StatusBadRequest, Code: "invalid_request"},
			common.ErrorMapping{Err: ErrInvalidTaxLevelSequence, Status: http.StatusBadRequest, Code: "invalid_request"},
			common.ErrorMapping{Err: ErrTaxLevelPercentOverflow, Status: http.StatusBadRequest, Code: "invalid_request"},
		)
		return
	}
	common.WriteJSON(w, http.StatusCreated, item)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseProjectID(w, r)
	if !ok {
		return
	}
	id, ok := common.ParseUUIDParam(w, r, "id", "tax rule")
	if !ok {
		return
	}
	var req upsertTaxRuleRequest
	if !common.DecodeAndValidate(w, r, h.validate, &req) {
		return
	}

	item, err := h.svc.Update(r.Context(), sqlc.UpdateTaxRuleParams{
		ProjectID: projectID,
		ID:        id,
		Name:      req.Name,
		Percent:   req.Percent,
		Level:     req.Level,
	})
	if err != nil {
		common.WriteMappedError(w, err, http.StatusInternalServerError, "update_tax_rule_failed", "failed to update tax rule",
			common.ErrorMapping{Err: ErrInvalidTaxLevel, Status: http.StatusBadRequest, Code: "invalid_request"},
			common.ErrorMapping{Err: ErrInvalidTaxLevelSequence, Status: http.StatusBadRequest, Code: "invalid_request"},
			common.ErrorMapping{Err: ErrTaxLevelPercentOverflow, Status: http.StatusBadRequest, Code: "invalid_request"},
		)
		return
	}
	common.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) Reorder(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseProjectID(w, r)
	if !ok {
		return
	}
	var req reorderTaxRulesRequest
	if !common.DecodeAndValidate(w, r, h.validate, &req) {
		return
	}

	orderedIDs := make([]uuid.UUID, 0, len(req.OrderedIDs))
	for _, idStr := range req.OrderedIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			common.WriteError(w, http.StatusBadRequest, "ordered_ids must be UUID")
			return
		}
		orderedIDs = append(orderedIDs, id)
	}

	if err := h.svc.Reorder(r.Context(), projectID, orderedIDs); err != nil {
		common.WriteMappedError(w, err, http.StatusInternalServerError, "reorder_tax_rules_failed", "failed to reorder tax rules",
			common.ErrorMapping{Err: ErrInvalidTaxOrder, Status: http.StatusBadRequest, Code: "invalid_request"},
		)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseProjectID(w, r)
	if !ok {
		return
	}
	id, ok := common.ParseUUIDParam(w, r, "id", "tax rule")
	if !ok {
		return
	}
	if err := h.svc.Delete(r.Context(), sqlc.DeleteTaxRuleParams{ProjectID: projectID, ID: id}); err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to delete tax rule")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseProjectID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	return common.ParseUUIDParam(w, r, "projectId", "project")
}
