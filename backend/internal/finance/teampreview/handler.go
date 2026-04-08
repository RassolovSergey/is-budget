package teampreview

import (
	"net/http"
	"strconv"

	"isbudget/backend/internal/auth"
	"isbudget/backend/internal/common"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Preview(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	projectID, ok := common.ParseUUIDParam(w, r, "projectId", "project")
	if !ok {
		return
	}

	var grossOverride *float64
	rawGross := r.URL.Query().Get("gross_amount")
	if rawGross != "" {
		value, err := strconv.ParseFloat(rawGross, 64)
		if err != nil || value < 0 {
			common.WriteError(w, http.StatusBadRequest, "gross_amount must be a number 0 or greater")
			return
		}
		grossOverride = &value
	}

	item, err := h.svc.Preview(r.Context(), user.ID, projectID, grossOverride)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to build team payout preview")
		return
	}
	common.WriteJSON(w, http.StatusOK, item)
}
