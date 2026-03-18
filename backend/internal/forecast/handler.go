package forecast

import (
	"errors"
	"net/http"

	"isbudget/backend/internal/auth"
	"isbudget/backend/internal/common"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

type Handler struct {
	svc      *Service
	validate *validator.Validate
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc, validate: common.NewValidator()}
}

func (h *Handler) Run(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseProjectID(w, r)
	if !ok {
		return
	}
	var req runRequest
	if !common.DecodeAndValidate(w, r, h.validate, &req) {
		return
	}

	result, err := h.svc.Run(r.Context(), projectID, req.HorizonDays)
	if err != nil {
		if !errors.Is(err, ErrInvalidHorizon) && !errors.Is(err, ErrNoRemainingForecast) {
			log.Error().Err(err).Str("project_id", projectID.String()).Msg("failed to run forecast")
		}
		common.WriteMappedError(
			w,
			err,
			http.StatusInternalServerError,
			"forecast_run_failed",
			"failed to run forecast",
			common.ErrorMapping{Err: ErrInvalidHorizon, Status: http.StatusBadRequest, Code: "invalid_request"},
			common.ErrorMapping{Err: ErrNoRemainingForecast, Status: http.StatusBadRequest, Code: "invalid_request"},
			common.ErrorMapping{Err: ErrAnalyticsTimeout, Status: http.StatusGatewayTimeout, Code: "forecast_service_timeout", Message: "forecast service timeout"},
			common.ErrorMapping{Err: ErrAnalyticsUnavailable, Status: http.StatusBadGateway, Code: "forecast_service_unavailable", Message: "forecast service unavailable"},
			common.ErrorMapping{Err: ErrAnalyticsInvalidResponse, Status: http.StatusBadGateway, Code: "forecast_service_invalid_response", Message: "forecast service returned invalid response"},
		)
		return
	}
	common.WriteJSON(w, http.StatusCreated, result)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseProjectID(w, r)
	if !ok {
		return
	}
	runID, ok := common.ParseUUIDParam(w, r, "runId", "run")
	if !ok {
		return
	}
	result, err := h.svc.GetByID(r.Context(), projectID, runID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			common.WriteError(w, http.StatusNotFound, "forecast run not found")
			return
		}
		common.WriteError(w, http.StatusInternalServerError, "failed to load forecast run")
		return
	}
	common.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) DeleteRun(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseProjectID(w, r)
	if !ok {
		return
	}
	runID, ok := common.ParseUUIDParam(w, r, "runId", "run")
	if !ok {
		return
	}
	if err := h.svc.DeleteRun(r.Context(), projectID, runID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			common.WriteError(w, http.StatusNotFound, "forecast run not found")
			return
		}
		log.Error().Err(err).Str("project_id", projectID.String()).Str("run_id", runID.String()).Msg("failed to delete forecast run")
		common.WriteError(w, http.StatusInternalServerError, "failed to delete forecast run")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) DownloadRunReport(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseProjectID(w, r)
	if !ok {
		return
	}
	runID, ok := common.ParseUUIDParam(w, r, "runId", "run")
	if !ok {
		return
	}

	data, fileName, err := h.svc.BuildRunReportPDF(r.Context(), projectID, runID)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			log.Error().Err(err).Str("project_id", projectID.String()).Str("run_id", runID.String()).Msg("failed to build forecast report")
		}
		common.WriteMappedError(w, err, http.StatusInternalServerError, "forecast_report_failed", "failed to build forecast report", common.ErrorMapping{Err: pgx.ErrNoRows, Status: http.StatusNotFound, Code: "not_found", Message: "forecast run not found"})
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+fileName+"\"")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (h *Handler) ListRuns(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseProjectID(w, r)
	if !ok {
		return
	}
	items, err := h.svc.ListRuns(r.Context(), projectID)
	if err != nil {
		log.Error().Err(err).Str("project_id", projectID.String()).Msg("failed to list forecast runs")
		common.WriteError(w, http.StatusInternalServerError, "failed to list forecast runs")
		return
	}
	common.WriteJSON(w, http.StatusOK, items)
}

func (h *Handler) GetLatest(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseProjectID(w, r)
	if !ok {
		return
	}
	result, err := h.svc.GetLatest(r.Context(), projectID)
	if err != nil {
		common.WriteMappedError(w, err, http.StatusInternalServerError, "forecast_load_failed", "failed to load latest forecast", common.ErrorMapping{Err: pgx.ErrNoRows, Status: http.StatusNotFound, Code: "not_found", Message: "no forecast run for project"})
		return
	}
	common.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) ListHistoricalCampaignsForUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := parseAuthUserID(w, r)
	if !ok {
		return
	}
	category := r.URL.Query().Get("category")
	items, err := h.svc.ListHistoricalCampaignsByUser(r.Context(), userID, category)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID.String()).Msg("failed to list historical campaigns")
		common.WriteError(w, http.StatusInternalServerError, "failed to list historical campaigns")
		return
	}
	common.WriteJSON(w, http.StatusOK, items)
}

func (h *Handler) CreateHistoricalCampaignForUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := parseAuthUserID(w, r)
	if !ok {
		return
	}
	var req upsertHistoricalCampaignRequest
	if err := common.DecodeJSON(r, &req); err != nil {
		common.WriteDecodeError(w, err)
		return
	}

	item, err := h.svc.CreateHistoricalCampaign(r.Context(), userID, mapUpsertHistoricalCampaignRequest(req))
	if err != nil {
		if !isHistoricalCampaignValidationError(err) {
			log.Error().Err(err).Str("user_id", userID.String()).Msg("failed to create historical campaign")
		}
		common.WriteMappedError(w, err, http.StatusInternalServerError, "historical_campaign_create_failed", "failed to create historical campaign", common.ErrorMapping{Err: ErrInvalidHistoricalCampaignName, Status: http.StatusBadRequest, Code: "invalid_request"}, common.ErrorMapping{Err: ErrInvalidHistoricalCampaignCategory, Status: http.StatusBadRequest, Code: "invalid_request"}, common.ErrorMapping{Err: ErrInvalidHistoricalCampaignDuration, Status: http.StatusBadRequest, Code: "invalid_request"}, common.ErrorMapping{Err: ErrInvalidHistoricalCampaignCurrency, Status: http.StatusBadRequest, Code: "invalid_request"}, common.ErrorMapping{Err: ErrInvalidHistoricalCampaignStartDate, Status: http.StatusBadRequest, Code: "invalid_request"}, common.ErrorMapping{Err: ErrInvalidHistoricalCampaignEndDate, Status: http.StatusBadRequest, Code: "invalid_request"}, common.ErrorMapping{Err: ErrInvalidHistoricalCampaignPeriod, Status: http.StatusBadRequest, Code: "invalid_request"}, common.ErrorMapping{Err: ErrInvalidHistoricalCampaignDateDuration, Status: http.StatusBadRequest, Code: "invalid_request"}, common.ErrorMapping{Err: ErrInvalidHistoricalCampaignDays, Status: http.StatusBadRequest, Code: "invalid_request"}, common.ErrorMapping{Err: ErrInvalidHistoricalCampaignDayNumber, Status: http.StatusBadRequest, Code: "invalid_request"}, common.ErrorMapping{Err: ErrInvalidHistoricalCampaignDayDuplicate, Status: http.StatusBadRequest, Code: "invalid_request"}, common.ErrorMapping{Err: ErrInvalidHistoricalCampaignIncome, Status: http.StatusBadRequest, Code: "invalid_request"}, common.ErrorMapping{Err: ErrInvalidHistoricalCampaignExpense, Status: http.StatusBadRequest, Code: "invalid_request"})
		return
	}
	common.WriteJSON(w, http.StatusCreated, item)
}

func (h *Handler) UpdateHistoricalCampaignForUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := parseAuthUserID(w, r)
	if !ok {
		return
	}
	campaignID, ok := common.ParseUUIDParam(w, r, "campaignId", "campaign")
	if !ok {
		return
	}
	var req upsertHistoricalCampaignRequest
	if err := common.DecodeJSON(r, &req); err != nil {
		common.WriteDecodeError(w, err)
		return
	}

	item, err := h.svc.UpdateHistoricalCampaign(r.Context(), userID, campaignID, mapUpsertHistoricalCampaignRequest(req))
	if err != nil {
		if !isHistoricalCampaignValidationError(err) && !errors.Is(err, pgx.ErrNoRows) {
			log.Error().Err(err).Str("user_id", userID.String()).Str("campaign_id", campaignID.String()).Msg("failed to update historical campaign")
		}
		common.WriteMappedError(w, err, http.StatusInternalServerError, "historical_campaign_update_failed", "failed to update historical campaign", common.ErrorMapping{Err: pgx.ErrNoRows, Status: http.StatusNotFound, Code: "not_found", Message: "historical campaign not found"}, common.ErrorMapping{Err: ErrInvalidHistoricalCampaignName, Status: http.StatusBadRequest, Code: "invalid_request"}, common.ErrorMapping{Err: ErrInvalidHistoricalCampaignCategory, Status: http.StatusBadRequest, Code: "invalid_request"}, common.ErrorMapping{Err: ErrInvalidHistoricalCampaignDuration, Status: http.StatusBadRequest, Code: "invalid_request"}, common.ErrorMapping{Err: ErrInvalidHistoricalCampaignCurrency, Status: http.StatusBadRequest, Code: "invalid_request"}, common.ErrorMapping{Err: ErrInvalidHistoricalCampaignStartDate, Status: http.StatusBadRequest, Code: "invalid_request"}, common.ErrorMapping{Err: ErrInvalidHistoricalCampaignEndDate, Status: http.StatusBadRequest, Code: "invalid_request"}, common.ErrorMapping{Err: ErrInvalidHistoricalCampaignPeriod, Status: http.StatusBadRequest, Code: "invalid_request"}, common.ErrorMapping{Err: ErrInvalidHistoricalCampaignDateDuration, Status: http.StatusBadRequest, Code: "invalid_request"}, common.ErrorMapping{Err: ErrInvalidHistoricalCampaignDays, Status: http.StatusBadRequest, Code: "invalid_request"}, common.ErrorMapping{Err: ErrInvalidHistoricalCampaignDayNumber, Status: http.StatusBadRequest, Code: "invalid_request"}, common.ErrorMapping{Err: ErrInvalidHistoricalCampaignDayDuplicate, Status: http.StatusBadRequest, Code: "invalid_request"}, common.ErrorMapping{Err: ErrInvalidHistoricalCampaignIncome, Status: http.StatusBadRequest, Code: "invalid_request"}, common.ErrorMapping{Err: ErrInvalidHistoricalCampaignExpense, Status: http.StatusBadRequest, Code: "invalid_request"})
		return
	}
	common.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) DeleteHistoricalCampaignForUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := parseAuthUserID(w, r)
	if !ok {
		return
	}
	campaignID, ok := common.ParseUUIDParam(w, r, "campaignId", "campaign")
	if !ok {
		return
	}
	if err := h.svc.DeleteHistoricalCampaign(r.Context(), userID, campaignID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			common.WriteError(w, http.StatusNotFound, "historical campaign not found")
			return
		}
		log.Error().Err(err).Str("user_id", userID.String()).Str("campaign_id", campaignID.String()).Msg("failed to delete historical campaign")
		common.WriteError(w, http.StatusInternalServerError, "failed to delete historical campaign")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func mapUpsertHistoricalCampaignRequest(req upsertHistoricalCampaignRequest) UpsertHistoricalCampaignRequest {
	out := UpsertHistoricalCampaignRequest{
		Name:         req.Name,
		Category:     req.Category,
		DurationDays: req.DurationDays,
		CurrencyCode: req.CurrencyCode,
		StartedAt:    req.StartedAt,
		EndedAt:      req.EndedAt,
		Days:         make([]HistoricalCampaignDay, 0, len(req.Days)),
	}
	for _, day := range req.Days {
		mappedDay := HistoricalCampaignDay{
			DayNumber:    day.DayNumber,
			IncomeAmount: day.IncomeAmount,
			Expenses:     make([]HistoricalCampaignExpense, 0, len(day.Expenses)),
		}
		for _, expense := range day.Expenses {
			mappedDay.Expenses = append(mappedDay.Expenses, HistoricalCampaignExpense{
				Category: expense.Category,
				Amount:   expense.Amount,
			})
		}
		out.Days = append(out.Days, mappedDay)
	}
	return out
}

func isHistoricalCampaignValidationError(err error) bool {
	return errors.Is(err, ErrInvalidHistoricalCampaignName) ||
		errors.Is(err, ErrInvalidHistoricalCampaignCategory) ||
		errors.Is(err, ErrInvalidHistoricalCampaignDuration) ||
		errors.Is(err, ErrInvalidHistoricalCampaignCurrency) ||
		errors.Is(err, ErrInvalidHistoricalCampaignStartDate) ||
		errors.Is(err, ErrInvalidHistoricalCampaignEndDate) ||
		errors.Is(err, ErrInvalidHistoricalCampaignPeriod) ||
		errors.Is(err, ErrInvalidHistoricalCampaignDateDuration) ||
		errors.Is(err, ErrInvalidHistoricalCampaignDays) ||
		errors.Is(err, ErrInvalidHistoricalCampaignDayNumber) ||
		errors.Is(err, ErrInvalidHistoricalCampaignDayDuplicate) ||
		errors.Is(err, ErrInvalidHistoricalCampaignIncome) ||
		errors.Is(err, ErrInvalidHistoricalCampaignExpense)
}

func parseProjectID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	return common.ParseUUIDParam(w, r, "projectId", "project")
}

func parseAuthUserID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return uuid.Nil, false
	}
	return user.ID, true
}
