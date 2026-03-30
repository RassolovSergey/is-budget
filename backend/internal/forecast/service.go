package forecast

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"isbudget/backend/internal/db/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrInvalidHorizon           = errors.New("horizon_days must be between 1 and 30")
	ErrNoRemainingForecast      = errors.New("no remaining campaign days for forecast")
	ErrAnalyticsInvalidRequest  = errors.New("analytics service rejected request")
	ErrAnalyticsUnavailable     = errors.New("analytics service unavailable")
	ErrAnalyticsTimeout         = errors.New("analytics service timeout")
	ErrAnalyticsInvalidResponse = errors.New("analytics service invalid response")
)

const (
	maxForecastAmount           = 9999999999.99
	defaultForecastModelVersion = "crowdfunding-hybrid-v2"
)

type Service struct {
	repo   Repository
	client AnalyticsClient
}

func NewService(repo Repository, client AnalyticsClient) *Service {
	return &Service{repo: repo, client: client}
}

type RunResult struct {
	Run    sqlc.ForecastRun     `json:"run"`
	Points []sqlc.ForecastPoint `json:"points"`
}

type RunSummary struct {
	Run               sqlc.ForecastRun `json:"run"`
	ForecastTotalYhat float64          `json:"forecast_total_yhat"`
	ForecastTotalP10  float64          `json:"forecast_total_p10"`
	ForecastTotalP90  float64          `json:"forecast_total_p90"`
}

func (s *Service) Run(ctx context.Context, projectID uuid.UUID, horizon int) (RunResult, error) {
	input, err := s.prepareRunInput(ctx, projectID, horizon)
	if err != nil {
		return RunResult{}, err
	}

	analyticsResp, err := s.client.Forecast(ctx, buildAnalyticsRequest(input))
	if err != nil {
		return RunResult{}, err
	}

	modelVersion := input.ModelVersion
	if analyticsResp.ModelVersion != "" {
		modelVersion = analyticsResp.ModelVersion
	}

	pointParams, _, err := mapAnalyticsResponseToPointParams(analyticsResp)
	if err != nil {
		return RunResult{}, err
	}

	return s.storeForecastRun(ctx, input, modelVersion, pointParams)
}

func (s *Service) ListRuns(ctx context.Context, projectID uuid.UUID) ([]RunSummary, error) {
	return s.repo.ListForecastRunsByProject(ctx, projectID)
}

func (s *Service) GetByID(ctx context.Context, projectID, runID uuid.UUID) (RunResult, error) {
	run, err := s.repo.GetForecastRunByID(ctx, runID)
	if err != nil {
		return RunResult{}, err
	}
	if run.ProjectID != projectID {
		return RunResult{}, pgx.ErrNoRows
	}
	points, err := s.repo.ListForecastPointsByRun(ctx, run.ID)
	if err != nil {
		return RunResult{}, err
	}
	return RunResult{Run: run, Points: points}, nil
}

func (s *Service) GetLatest(ctx context.Context, projectID uuid.UUID) (RunResult, error) {
	run, err := s.repo.GetLatestForecastRunByProject(ctx, projectID)
	if err != nil {
		return RunResult{}, err
	}
	points, err := s.repo.ListForecastPointsByRun(ctx, run.ID)
	if err != nil {
		return RunResult{}, err
	}
	return RunResult{Run: run, Points: points}, nil
}

func (s *Service) DeleteRun(ctx context.Context, projectID, runID uuid.UUID) error {
	return s.repo.DeleteForecastRun(ctx, projectID, runID)
}

func (s *Service) ListHistoricalCampaignsByUser(ctx context.Context, ownerUserID uuid.UUID, category string) ([]HistoricalCampaign, error) {
	normalizedCategory := strings.TrimSpace(category)
	manualCampaigns, err := s.repo.ListHistoricalCampaignsByOwner(ctx, ownerUserID, normalizedCategory)
	if err != nil {
		return nil, err
	}
	completedCampaigns, err := s.repo.ListCompletedProjectHistoricalCampaignsByOwnerAndCategory(ctx, ownerUserID, normalizedCategory, uuid.Nil)
	if err != nil {
		return nil, err
	}

	merged := make([]HistoricalCampaign, 0, len(manualCampaigns)+len(completedCampaigns))
	merged = append(merged, manualCampaigns...)
	merged = append(merged, completedCampaigns...)
	sort.SliceStable(merged, func(i, j int) bool {
		return merged[i].CreatedAt.After(merged[j].CreatedAt)
	})
	return merged, nil
}

func (s *Service) CreateHistoricalCampaign(ctx context.Context, ownerUserID uuid.UUID, req UpsertHistoricalCampaignRequest) (HistoricalCampaign, error) {
	normalized, err := normalizeHistoricalCampaignInput(req)
	if err != nil {
		return HistoricalCampaign{}, err
	}
	return s.repo.CreateHistoricalCampaign(ctx, ownerUserID, normalized)
}

func (s *Service) UpdateHistoricalCampaign(ctx context.Context, ownerUserID, campaignID uuid.UUID, req UpsertHistoricalCampaignRequest) (HistoricalCampaign, error) {
	normalized, err := normalizeHistoricalCampaignInput(req)
	if err != nil {
		return HistoricalCampaign{}, err
	}
	return s.repo.UpdateHistoricalCampaign(ctx, ownerUserID, campaignID, normalized)
}

func (s *Service) DeleteHistoricalCampaign(ctx context.Context, ownerUserID, campaignID uuid.UUID) error {
	return s.repo.DeleteHistoricalCampaign(ctx, ownerUserID, campaignID)
}

func buildInputHashAndSeed(
	projectID uuid.UUID,
	horizon int,
	modelVersion string,
	history []HistoryPoint,
	historicalCampaigns []AnalyticsHistoricalCampaign,
	currentIncomeToDate float64,
	currentAdSpendToDate float64,
) (string, int64) {
	parts := make([]string, 0, len(history)+len(historicalCampaigns)+6)
	parts = append(parts, projectID.String(), fmt.Sprintf("%d", horizon), modelVersion)
	for _, point := range history {
		parts = append(parts, fmt.Sprintf("%s:%.2f", point.Date, point.Amount))
	}
	parts = append(parts, fmt.Sprintf("income_to_date:%.2f", currentIncomeToDate))
	parts = append(parts, fmt.Sprintf("ad_spend_to_date:%.2f", currentAdSpendToDate))
	for _, campaign := range historicalCampaigns {
		parts = append(parts, fmt.Sprintf("campaign:%s|%d|%s|%s|%s", campaign.Name, campaign.DurationDays, campaign.CurrencyCode, campaign.StartedAt, campaign.EndedAt))
		for _, day := range campaign.Days {
			parts = append(parts, fmt.Sprintf("day:%d:%.2f", day.DayNumber, day.IncomeAmount))
			for _, expense := range day.Expenses {
				parts = append(parts, fmt.Sprintf("expense:%d:%s:%.2f", day.DayNumber, strings.ToLower(expense.Category), expense.Amount))
			}
		}
	}
	raw := strings.Join(parts, "|")
	sum := sha256.Sum256([]byte(raw))
	hash := hex.EncodeToString(sum[:])
	seed := int64(binary.BigEndian.Uint64(sum[:8]))
	if seed < 0 {
		seed = -seed
	}
	if seed == 0 {
		seed = 42
	}
	return hash, seed
}

func mapHistoricalCampaignsToAnalytics(campaigns []HistoricalCampaign) []AnalyticsHistoricalCampaign {
	out := make([]AnalyticsHistoricalCampaign, 0, len(campaigns))
	for _, campaign := range campaigns {
		days := make([]AnalyticsHistoricalDay, 0, len(campaign.Days))
		for _, day := range campaign.Days {
			expenses := make([]AnalyticsHistoricalExpense, 0, len(day.Expenses))
			for _, expense := range day.Expenses {
				expenses = append(expenses, AnalyticsHistoricalExpense{
					Category: expense.Category,
					Amount:   round2(expense.Amount),
				})
			}
			days = append(days, AnalyticsHistoricalDay{
				DayNumber:    int(day.DayNumber),
				IncomeAmount: round2(day.IncomeAmount),
				Expenses:     expenses,
			})
		}
		out = append(out, AnalyticsHistoricalCampaign{
			Name:         campaign.Name,
			DurationDays: int(campaign.DurationDays),
			CurrencyCode: campaign.CurrencyCode,
			StartedAt:    campaign.StartedAt,
			EndedAt:      campaign.EndedAt,
			Days:         days,
		})
	}
	return out
}

func round2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}

func normalizeForecastAmount(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	if v < 0 {
		return 0
	}
	if v > maxForecastAmount {
		return maxForecastAmount
	}
	return round2(v)
}
