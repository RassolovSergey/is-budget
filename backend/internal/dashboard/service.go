package dashboard

import (
	"context"
	"errors"
	"time"

	"isbudget/backend/internal/db/sqlc"
	"isbudget/backend/internal/finance/payout"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Service struct {
	q         *sqlc.Queries
	payoutSvc *payout.Service
}

func NewService(q *sqlc.Queries, payoutSvc *payout.Service) *Service {
	return &Service{q: q, payoutSvc: payoutSvc}
}

type LastForecastRunSummary struct {
	ID           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	HorizonDays  int32     `json:"horizon_days"`
	ModelVersion string    `json:"model_version"`
	InputHash    string    `json:"input_hash"`
	Seed         int64     `json:"seed"`
}

type Response struct {
	DailyIncomeSeries   []sqlc.ActualIncomeDaily `json:"daily_income_series"`
	PlannedExpenses     []sqlc.PlannedExpense    `json:"planned_expenses"`
	ActualExpenses      []sqlc.ActualExpense     `json:"actual_expenses"`
	IncomeTotal         float64                  `json:"income_total"`
	ExpensesTotal       float64                  `json:"expenses_total"`
	NetProfitToDate     float64                  `json:"net_profit_to_date"`
	MandatoryTotal      float64                  `json:"mandatory_total"`
	MandatoryPaid       float64                  `json:"mandatory_paid"`
	MandatoryRemaining  float64                  `json:"mandatory_remaining"`
	AvailableBeforeTax  float64                  `json:"available_before_tax"`
	TaxPercentTotal     float64                  `json:"tax_percent_total"`
	TaxTotal            float64                  `json:"tax_total"`
	AvailableForSharing float64                  `json:"available_for_sharing"`
	PayoutPreview       []payout.PreviewItem     `json:"payout_preview"`
	LastForecastRun     *LastForecastRunSummary  `json:"last_forecast_run"`
}

func normalizeUTCDate(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func resolveDashboardIncomeRange(project sqlc.GetProjectByIDRow, days int, now time.Time) (time.Time, time.Time) {
	to := normalizeUTCDate(now.UTC())
	from := to.AddDate(0, 0, -(days - 1))

	if project.ActivatedAt == nil {
		return from, to
	}

	campaignStart := normalizeUTCDate(project.ActivatedAt.UTC())
	campaignEnd := campaignStart
	if project.DurationDays > 1 {
		campaignEnd = campaignStart.AddDate(0, 0, int(project.DurationDays)-1)
	}
	return campaignStart, campaignEnd
}

func (s *Service) Get(ctx context.Context, projectID uuid.UUID, days int) (Response, error) {
	if days <= 0 {
		days = 30
	}
	if days > 365 {
		days = 365
	}
	project, err := s.q.GetProjectByID(ctx, projectID)
	if err != nil {
		return Response{}, err
	}

	now := time.Now().UTC()
	incomeFrom, incomeTo := resolveDashboardIncomeRange(project, days, now)
	to := normalizeUTCDate(now)
	from := to.AddDate(0, 0, -(days - 1))

	incomeSeries, err := s.q.ListIncomeDailyByProjectRange(ctx, sqlc.ListIncomeDailyByProjectRangeParams{
		ProjectID: projectID,
		FromDate:  incomeFrom,
		ToDate:    incomeTo,
	})
	if err != nil {
		return Response{}, err
	}
	planned, err := s.q.ListPlannedExpensesByProject(ctx, projectID)
	if err != nil {
		return Response{}, err
	}
	actualExpenses, err := s.q.ListActualExpensesByProjectRange(ctx, sqlc.ListActualExpensesByProjectRangeParams{
		ProjectID: projectID,
		FromDate:  from,
		ToDate:    to,
	})
	if err != nil {
		return Response{}, err
	}

	incomeItems := make([]sqlc.ActualIncomeDaily, 0, len(incomeSeries))
	for _, row := range incomeSeries {
		incomeItems = append(incomeItems, sqlc.ActualIncomeDaily(row))
	}
	plannedItems := make([]sqlc.PlannedExpense, 0, len(planned))
	for _, row := range planned {
		plannedItems = append(plannedItems, sqlc.PlannedExpense(row))
	}
	actualExpenseItems := make([]sqlc.ActualExpense, 0, len(actualExpenses))
	for _, row := range actualExpenses {
		actualExpenseItems = append(actualExpenseItems, sqlc.ActualExpense(row))
	}

	preview, err := s.payoutSvc.Preview(ctx, projectID)
	if err != nil {
		return Response{}, err
	}

	var lastRun *LastForecastRunSummary
	run, err := s.q.GetLatestForecastRunByProject(ctx, projectID)
	if err == nil {
		lastRun = &LastForecastRunSummary{
			ID:           run.ID,
			CreatedAt:    run.CreatedAt,
			HorizonDays:  run.HorizonDays,
			ModelVersion: run.ModelVersion,
			InputHash:    run.InputHash,
			Seed:         run.Seed,
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return Response{}, err
	}

	return Response{
		DailyIncomeSeries:   incomeItems,
		PlannedExpenses:     plannedItems,
		ActualExpenses:      actualExpenseItems,
		IncomeTotal:         preview.Summary.IncomeTotal,
		ExpensesTotal:       preview.Summary.ExpensesTotal,
		NetProfitToDate:     preview.Summary.NetProfitToDate,
		MandatoryTotal:      preview.Summary.MandatoryTotal,
		MandatoryPaid:       preview.Summary.MandatoryPaid,
		MandatoryRemaining:  preview.Summary.MandatoryRemaining,
		AvailableBeforeTax:  preview.Summary.AvailableBeforeTax,
		TaxPercentTotal:     preview.Summary.TaxPercentTotal,
		TaxTotal:            preview.Summary.TaxTotal,
		AvailableForSharing: preview.Summary.AvailableForSharing,
		PayoutPreview:       preview.Items,
		LastForecastRun:     lastRun,
	}, nil
}
