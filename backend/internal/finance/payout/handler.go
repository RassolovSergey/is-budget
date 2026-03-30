package payout

import (
	"errors"
	"fmt"
	"net/http"

	"isbudget/backend/internal/auth"
	"isbudget/backend/internal/common"
	"isbudget/backend/internal/db/sqlc"

	"github.com/go-chi/chi/v5"
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

func (h *Handler) ListRules(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseProjectID(w, r)
	if !ok {
		return
	}
	items, err := h.svc.ListRules(r.Context(), projectID)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to load profit share rules")
		return
	}
	common.WriteJSON(w, http.StatusOK, items)
}

func (h *Handler) CreateRule(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseProjectID(w, r)
	if !ok {
		return
	}
	var req upsertRuleRequest
	if !common.DecodeAndValidate(w, r, h.validate, &req) {
		return
	}
	var recipientUserID *uuid.UUID
	if req.RecipientUserID != "" {
		id, ok := common.ParseUUIDValue(w, req.RecipientUserID, "recipient_user_id")
		if !ok {
			return
		}
		recipientUserID = &id
	}
	var (
		item sqlc.ProfitShareRule
		err  error
	)
	if req.PrependBeforeFirst {
		item, err = h.svc.CreateRulePrepend(r.Context(), sqlc.CreateProfitShareRuleParams{
			ProjectID:             projectID,
			RecipientUserID:       recipientUserID,
			RecipientName:         req.RecipientName,
			Responsibility:        req.Responsibility,
			CurrencyCode:          req.CurrencyCode,
			Percent:               req.Percent,
			Level:                 1,
			AppliesAfterMandatory: false,
		})
	} else {
		item, err = h.svc.CreateRule(r.Context(), sqlc.CreateProfitShareRuleParams{
			ProjectID:             projectID,
			RecipientUserID:       recipientUserID,
			RecipientName:         req.RecipientName,
			Responsibility:        req.Responsibility,
			CurrencyCode:          req.CurrencyCode,
			Percent:               req.Percent,
			Level:                 req.Level,
			AppliesAfterMandatory: req.AppliesAfterMandatory,
		})
	}
	if err != nil {
		common.WriteMappedError(w, err, http.StatusInternalServerError, "create_profit_share_rule_failed", "failed to create profit share rule",
			common.ErrorMapping{Err: ErrRulePercentOverflow, Status: http.StatusBadRequest, Code: "invalid_request"},
			common.ErrorMapping{Err: ErrInvalidRuleLevel, Status: http.StatusBadRequest, Code: "invalid_request"},
			common.ErrorMapping{Err: ErrInvalidRuleLevelSequence, Status: http.StatusBadRequest, Code: "invalid_request"},
			common.ErrorMapping{Err: ErrRuleNextLevelBlocked, Status: http.StatusBadRequest, Code: "invalid_request"},
			common.ErrorMapping{Err: ErrInvalidRuleCurrency, Status: http.StatusBadRequest, Code: "invalid_request"},
			common.ErrorMapping{Err: ErrInvalidRecipientUser, Status: http.StatusBadRequest, Code: "invalid_request"},
		)
		return
	}
	common.WriteJSON(w, http.StatusCreated, item)
}

func (h *Handler) UpdateRule(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseProjectID(w, r)
	if !ok {
		return
	}
	id, ok := common.ParseUUIDParam(w, r, "id", "rule")
	if !ok {
		return
	}
	var req upsertRuleRequest
	if !common.DecodeAndValidate(w, r, h.validate, &req) {
		return
	}
	var recipientUserID *uuid.UUID
	if req.RecipientUserID != "" {
		parsedID, ok := common.ParseUUIDValue(w, req.RecipientUserID, "recipient_user_id")
		if !ok {
			return
		}
		recipientUserID = &parsedID
	}
	item, err := h.svc.UpdateRule(r.Context(), sqlc.UpdateProfitShareRuleParams{
		ProjectID:             projectID,
		ID:                    id,
		RecipientUserID:       recipientUserID,
		RecipientName:         req.RecipientName,
		Responsibility:        req.Responsibility,
		CurrencyCode:          req.CurrencyCode,
		Percent:               req.Percent,
		Level:                 req.Level,
		AppliesAfterMandatory: req.AppliesAfterMandatory,
	})
	if err != nil {
		common.WriteMappedError(w, err, http.StatusInternalServerError, "update_profit_share_rule_failed", "failed to update profit share rule",
			common.ErrorMapping{Err: ErrRulePercentOverflow, Status: http.StatusBadRequest, Code: "invalid_request"},
			common.ErrorMapping{Err: ErrInvalidRuleLevel, Status: http.StatusBadRequest, Code: "invalid_request"},
			common.ErrorMapping{Err: ErrInvalidRuleLevelSequence, Status: http.StatusBadRequest, Code: "invalid_request"},
			common.ErrorMapping{Err: ErrRuleNextLevelBlocked, Status: http.StatusBadRequest, Code: "invalid_request"},
			common.ErrorMapping{Err: ErrInvalidRuleCurrency, Status: http.StatusBadRequest, Code: "invalid_request"},
			common.ErrorMapping{Err: ErrInvalidRecipientUser, Status: http.StatusBadRequest, Code: "invalid_request"},
		)
		return
	}
	common.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseProjectID(w, r)
	if !ok {
		return
	}
	id, ok := common.ParseUUIDParam(w, r, "id", "rule")
	if !ok {
		return
	}
	if err := h.svc.DeleteRule(r.Context(), sqlc.DeleteProfitShareRuleParams{ProjectID: projectID, ID: id}); err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to delete profit share rule")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) PayoutPreview(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseProjectID(w, r)
	if !ok {
		return
	}
	preview, err := h.svc.Preview(r.Context(), projectID)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to calculate payout preview")
		return
	}
	common.WriteJSON(w, http.StatusOK, preview)
}

func (h *Handler) MyIncome(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseProjectID(w, r)
	if !ok {
		return
	}
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	result, err := h.svc.MyIncome(r.Context(), projectID, user.ID)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to calculate my income")
		return
	}
	common.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) CalculatePayouts(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseProjectID(w, r)
	if !ok {
		return
	}
	payouts, preview, snapshot, err := h.svc.CalculateAndPersist(r.Context(), projectID)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to calculate payouts")
		return
	}
	snapshot.DownloadURL = fmt.Sprintf("/projects/%s/payout-snapshots/%s", projectID.String(), snapshot.FileName)
	common.WriteJSON(w, http.StatusCreated, map[string]any{
		"preview":  preview,
		"payouts":  payouts,
		"snapshot": snapshot,
	})
}

func (h *Handler) DownloadSnapshot(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseProjectID(w, r)
	if !ok {
		return
	}
	fileName := chi.URLParam(r, "fileName")
	filePath, err := h.svc.GetSnapshotPath(projectID, fileName)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidSnapshotFileName):
			common.WriteError(w, http.StatusBadRequest, "invalid snapshot file name")
			return
		case errors.Is(err, ErrSnapshotFileNotFound):
			common.WriteError(w, http.StatusNotFound, "snapshot file not found")
			return
		default:
			common.WriteError(w, http.StatusInternalServerError, "failed to open snapshot file")
			return
		}
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	http.ServeFile(w, r, filePath)
}

func parseProjectID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	return common.ParseUUIDParam(w, r, "projectId", "project")
}
