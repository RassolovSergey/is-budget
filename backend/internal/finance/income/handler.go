package income

import (
	"net/http"
	"time"

	"isbudget/backend/internal/common"

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

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseProjectID(w, r)
	if !ok {
		return
	}
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	var from, to time.Time
	if fromStr == "" || toStr == "" {
		to = time.Now().UTC()
		to = time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)
		from = to.AddDate(0, 0, -29)
	} else {
		var ok bool
		from, ok = common.ParseDateValue(w, fromStr, "from")
		if !ok {
			return
		}
		to, ok = common.ParseDateValue(w, toStr, "to")
		if !ok {
			return
		}
	}

	items, err := h.svc.ListRange(r.Context(), projectID, from, to)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to load income records")
		return
	}
	common.WriteJSON(w, http.StatusOK, items)
}

func (h *Handler) Upsert(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseProjectID(w, r)
	if !ok {
		return
	}
	var req upsertRequest
	if !common.DecodeAndValidate(w, r, h.validate, &req) {
		return
	}
	date, ok := common.ParseDateValue(w, req.Date, "date")
	if !ok {
		return
	}
	item, err := h.svc.Upsert(r.Context(), projectID, date, req.Amount)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to upsert income record")
		return
	}
	common.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseProjectID(w, r)
	if !ok {
		return
	}
	date, ok := common.ParseDateValue(w, chi.URLParam(r, "date"), "date")
	if !ok {
		return
	}
	if err := h.svc.Delete(r.Context(), projectID, date); err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to delete income record")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) UpsertCumulative(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseProjectID(w, r)
	if !ok {
		return
	}

	var req upsertCumulativeRequest
	if !common.DecodeAndValidate(w, r, h.validate, &req) {
		return
	}

	points := make([]CumulativePointInput, 0, len(req.Points))
	for _, point := range req.Points {
		date, ok := common.ParseDateValue(w, point.Date, "date")
		if !ok {
			return
		}
		if point.CumulativeAmount != nil && *point.CumulativeAmount < 0 {
			common.WriteErrorWithCode(w, http.StatusBadRequest, "invalid_request", "cumulative_amount must be greater than or equal to 0")
			return
		}
		points = append(points, CumulativePointInput{
			Date:             date,
			CumulativeAmount: point.CumulativeAmount,
		})
	}

	items, err := h.svc.UpsertCumulative(r.Context(), projectID, points)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "failed to upsert cumulative income series")
		return
	}
	common.WriteJSON(w, http.StatusOK, items)
}

func parseProjectID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	return common.ParseUUIDParam(w, r, "projectId", "project")
}
