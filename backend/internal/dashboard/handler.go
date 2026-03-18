package dashboard

import (
	"net/http"
	"strconv"

	"isbudget/backend/internal/common"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(chi.URLParam(r, "projectId"))
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	days := 30
	if v := r.URL.Query().Get("days"); v != "" {
		d, err := strconv.Atoi(v)
		if err != nil || d < 1 || d > 365 {
			common.WriteError(w, http.StatusBadRequest, "days must be between 1 and 365")
			return
		}
		days = d
	}

	resp, err := h.svc.Get(r.Context(), projectID, days)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to load dashboard")
		return
	}
	common.WriteJSON(w, http.StatusOK, resp)
}
