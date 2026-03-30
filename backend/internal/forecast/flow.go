package forecast

import (
	"context"
	"fmt"
	"time"

	"isbudget/backend/internal/common"
	"isbudget/backend/internal/db/sqlc"

	"github.com/google/uuid"
)

type preparedRunInput struct {
	ProjectID            uuid.UUID
	HorizonDays          int
	ModelVersion         string
	History              []HistoryPoint
	HistoricalCampaigns  []AnalyticsHistoricalCampaign
	CurrentIncomeToDate  float64
	CurrentAdSpendToDate float64
	InputHash            string
	Seed                 int64
}

var forecastTodayUTC = func() time.Time {
	now := time.Now().UTC()
	return normalizeUTCDate(now)
}

func (s *Service) prepareRunInput(ctx context.Context, projectID uuid.UUID, horizon int) (preparedRunInput, error) {
	// Сначала собираем весь наблюдаемый контекст проекта, чтобы analytics-service получал один стабильный input.
	ownerUserID, category, durationDays, campaignStartDate, err := s.repo.GetProjectOwnerAndCategoryAndDuration(ctx, projectID)
	if err != nil {
		return preparedRunInput{}, err
	}
	campaignEndDate := campaignStartDate
	if durationDays > 1 {
		campaignEndDate = campaignStartDate.AddDate(0, 0, int(durationDays)-1)
	}

	historyRows, err := s.repo.ListIncomeDailyByProject(ctx, projectID)
	if err != nil {
		return preparedRunInput{}, err
	}
	observedThroughDate := resolveObservedThroughDate(historyRows, campaignStartDate, campaignEndDate, forecastTodayUTC())
	history, currentIncomeTotal := buildObservedHistory(historyRows, campaignStartDate, observedThroughDate)
	observedDays := countObservedCampaignDays(campaignStartDate, observedThroughDate)

	actualExpenses, err := s.repo.ListActualExpensesByProject(ctx, projectID)
	if err != nil {
		return preparedRunInput{}, err
	}
	currentAdSpend := calculateAdSpend(actualExpenses)

	resolvedHorizon, err := resolveForecastHorizon(durationDays, observedDays, horizon)
	if err != nil {
		return preparedRunInput{}, err
	}

	historicalCampaigns, err := s.loadAnalyticsHistoricalCampaigns(ctx, ownerUserID, category, projectID)
	if err != nil {
		return preparedRunInput{}, err
	}

	modelVersion := defaultForecastModelVersion
	inputHash, seed := buildInputHashAndSeed(projectID, resolvedHorizon, modelVersion, history, historicalCampaigns, currentIncomeTotal, currentAdSpend)

	return preparedRunInput{
		ProjectID:            projectID,
		HorizonDays:          resolvedHorizon,
		ModelVersion:         modelVersion,
		History:              history,
		HistoricalCampaigns:  historicalCampaigns,
		CurrentIncomeToDate:  round2(currentIncomeTotal),
		CurrentAdSpendToDate: round2(currentAdSpend),
		InputHash:            inputHash,
		Seed:                 seed,
	}, nil
}

func normalizeUTCDate(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func countObservedCampaignDays(campaignStartDate, observedThroughDate time.Time) int {
	startDate := normalizeUTCDate(campaignStartDate)
	endDate := normalizeUTCDate(observedThroughDate)
	if endDate.Before(startDate) {
		return 0
	}
	return int(endDate.Sub(startDate).Hours()/24) + 1
}

func resolveObservedThroughDate(rows []sqlc.ActualIncomeDaily, campaignStartDate, campaignEndDate, fallbackDate time.Time) time.Time {
	startDate := normalizeUTCDate(campaignStartDate)
	endDate := normalizeUTCDate(campaignEndDate)

	var latestActualDate time.Time
	for _, row := range rows {
		incomeDate := normalizeUTCDate(row.IncomeDate)
		if incomeDate.Before(startDate) || incomeDate.After(endDate) {
			continue
		}
		if latestActualDate.IsZero() || incomeDate.After(latestActualDate) {
			latestActualDate = incomeDate
		}
	}
	if !latestActualDate.IsZero() {
		return latestActualDate
	}

	observedThroughDate := normalizeUTCDate(fallbackDate)
	if observedThroughDate.After(endDate) {
		observedThroughDate = endDate
	}
	if observedThroughDate.Before(startDate) {
		observedThroughDate = startDate
	}
	return observedThroughDate
}

func buildObservedHistory(rows []sqlc.ActualIncomeDaily, campaignStartDate, observedThroughDate time.Time) ([]HistoryPoint, float64) {
	startDate := normalizeUTCDate(campaignStartDate)
	endDate := normalizeUTCDate(observedThroughDate)
	if endDate.Before(startDate) {
		return []HistoryPoint{}, 0
	}

	amountByDate := make(map[string]float64, len(rows))
	currentIncomeTotal := 0.0
	for _, row := range rows {
		incomeDate := normalizeUTCDate(row.IncomeDate)
		if incomeDate.Before(startDate) || incomeDate.After(endDate) {
			continue
		}
		amountByDate[incomeDate.Format(common.DateLayout)] += row.Amount
		currentIncomeTotal += row.Amount
	}

	history := make([]HistoryPoint, 0, countObservedCampaignDays(startDate, endDate))
	for cursor := startDate; !cursor.After(endDate); cursor = cursor.AddDate(0, 0, 1) {
		dateKey := cursor.Format(common.DateLayout)
		history = append(history, HistoryPoint{
			Date:   dateKey,
			Amount: amountByDate[dateKey],
		})
	}

	return history, currentIncomeTotal
}

func calculateAdSpend(expenses []sqlc.ActualExpense) float64 {
	currentAdSpend := 0.0
	for _, expense := range expenses {
		if isAdCategory(expense.Category) {
			currentAdSpend += expense.Amount
		}
	}
	return currentAdSpend
}

func resolveForecastHorizon(durationDays int32, observedDays, requestedHorizon int) (int, error) {
	horizon := requestedHorizon
	if durationDays > 0 {
		remainingDays := int(durationDays) - observedDays
		if remainingDays <= 0 {
			return 0, ErrNoRemainingForecast
		}
		if horizon > remainingDays {
			horizon = remainingDays
		}
	}
	if horizon < 1 || horizon > 30 {
		return 0, ErrInvalidHorizon
	}
	return horizon, nil
}

func (s *Service) loadAnalyticsHistoricalCampaigns(ctx context.Context, ownerUserID uuid.UUID, category string, projectID uuid.UUID) ([]AnalyticsHistoricalCampaign, error) {
	// История для модели складывается из ручного архива и завершённых проектов той же категории.
	manualHistoricalCampaigns, err := s.repo.ListHistoricalCampaignsByOwner(ctx, ownerUserID, category)
	if err != nil {
		return nil, err
	}
	completedHistoricalCampaigns, err := s.repo.ListCompletedProjectHistoricalCampaignsByOwnerAndCategory(ctx, ownerUserID, category, projectID)
	if err != nil {
		return nil, err
	}
	historicalCampaigns := make([]HistoricalCampaign, 0, len(manualHistoricalCampaigns)+len(completedHistoricalCampaigns))
	historicalCampaigns = append(historicalCampaigns, manualHistoricalCampaigns...)
	historicalCampaigns = append(historicalCampaigns, completedHistoricalCampaigns...)
	return mapHistoricalCampaignsToAnalytics(historicalCampaigns), nil
}

func buildAnalyticsRequest(input preparedRunInput) AnalyticsRequest {
	return AnalyticsRequest{
		ProjectID:            input.ProjectID.String(),
		HorizonDays:          input.HorizonDays,
		History:              input.History,
		HistoricalCampaigns:  input.HistoricalCampaigns,
		CurrentIncomeToDate:  input.CurrentIncomeToDate,
		CurrentAdSpendToDate: input.CurrentAdSpendToDate,
		ModelVersion:         input.ModelVersion,
		Seed:                 input.Seed,
	}
}

func mapAnalyticsResponseToPointParams(resp AnalyticsResponse) ([]sqlc.CreateForecastPointParams, string, error) {
	modelVersion := resp.ModelVersion
	pointParams := make([]sqlc.CreateForecastPointParams, 0, len(resp.Forecast))
	for _, point := range resp.Forecast {
		forecastDate, err := common.ParseDate(point.Date)
		if err != nil {
			return nil, modelVersion, fmt.Errorf("%w: invalid forecast date %q", ErrAnalyticsInvalidResponse, point.Date)
		}
		yhat := normalizeForecastAmount(point.Yhat)
		p10 := normalizeForecastAmount(point.P10)
		p90 := normalizeForecastAmount(point.P90)
		// Даже при неидеальном ответе интеграции сохраняем инвариант сценариев для backend и frontend.
		if p10 > yhat {
			p10 = yhat
		}
		if p90 < yhat {
			p90 = yhat
		}
		pointParams = append(pointParams, sqlc.CreateForecastPointParams{
			ForecastDate: forecastDate,
			Yhat:         yhat,
			P10:          p10,
			P90:          p90,
		})
	}
	return pointParams, modelVersion, nil
}

func (s *Service) storeForecastRun(ctx context.Context, input preparedRunInput, modelVersion string, pointParams []sqlc.CreateForecastPointParams) (RunResult, error) {
	run, points, err := s.repo.StoreRunWithPoints(ctx, sqlc.CreateForecastRunParams{
		ProjectID:    input.ProjectID,
		CreatedAt:    time.Now().UTC(),
		HorizonDays:  int32(input.HorizonDays),
		ModelVersion: modelVersion,
		InputHash:    input.InputHash,
		Seed:         input.Seed,
	}, pointParams)
	if err != nil {
		return RunResult{}, err
	}
	return RunResult{Run: run, Points: points}, nil
}
