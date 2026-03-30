package forecast

import (
	"context"
	"errors"
	"testing"
	"time"

	"isbudget/backend/internal/db/sqlc"

	"github.com/google/uuid"
)

func setForecastTodayForTest(t *testing.T, today time.Time) {
	t.Helper()
	prev := forecastTodayUTC
	forecastTodayUTC = func() time.Time {
		return normalizeUTCDate(today)
	}
	t.Cleanup(func() {
		forecastTodayUTC = prev
	})
}

type fakeAnalyticsClient struct {
	response AnalyticsResponse
	err      error
	lastReq  AnalyticsRequest
}

func (f *fakeAnalyticsClient) Forecast(ctx context.Context, req AnalyticsRequest) (AnalyticsResponse, error) {
	f.lastReq = req
	if f.err != nil {
		return AnalyticsResponse{}, f.err
	}
	return f.response, nil
}

type fakeRepo struct {
	history             []sqlc.ActualIncomeDaily
	actualExpenses      []sqlc.ActualExpense
	historicalCampaigns []HistoricalCampaign
	lastRun             sqlc.ForecastRun
	pointsByRun         map[uuid.UUID][]sqlc.ForecastPoint
}

func (f *fakeRepo) ListIncomeDailyByProject(ctx context.Context, projectID uuid.UUID) ([]sqlc.ActualIncomeDaily, error) {
	return f.history, nil
}

func (f *fakeRepo) StoreRunWithPoints(ctx context.Context, run sqlc.CreateForecastRunParams, points []sqlc.CreateForecastPointParams) (sqlc.ForecastRun, []sqlc.ForecastPoint, error) {
	runID := uuid.MustParse("aaaaaaaa-1111-1111-1111-aaaaaaaaaaaa")
	createdRun := sqlc.ForecastRun{
		ID:           runID,
		ProjectID:    run.ProjectID,
		CreatedAt:    run.CreatedAt,
		HorizonDays:  run.HorizonDays,
		ModelVersion: run.ModelVersion,
		InputHash:    run.InputHash,
		Seed:         run.Seed,
	}
	createdPoints := make([]sqlc.ForecastPoint, 0, len(points))
	for _, point := range points {
		createdPoints = append(createdPoints, sqlc.ForecastPoint{
			ID:           uuid.New(),
			RunID:        runID,
			ForecastDate: point.ForecastDate,
			Yhat:         point.Yhat,
			P10:          point.P10,
			P90:          point.P90,
		})
	}
	f.lastRun = createdRun
	if f.pointsByRun == nil {
		f.pointsByRun = map[uuid.UUID][]sqlc.ForecastPoint{}
	}
	f.pointsByRun[runID] = createdPoints
	return createdRun, createdPoints, nil
}

func (f *fakeRepo) ListActualExpensesByProject(ctx context.Context, projectID uuid.UUID) ([]sqlc.ActualExpense, error) {
	return f.actualExpenses, nil
}

func (f *fakeRepo) GetProjectByID(ctx context.Context, projectID uuid.UUID) (sqlc.GetProjectByIDRow, error) {
	return sqlc.GetProjectByIDRow{
		ID:           projectID,
		Name:         "Test Project",
		CurrencyCode: "EUR",
		DurationDays: 10,
	}, nil
}

func (f *fakeRepo) GetProjectOwnerAndCategoryAndDuration(ctx context.Context, projectID uuid.UUID) (uuid.UUID, string, int32, time.Time, error) {
	return uuid.MustParse("11111111-1111-1111-1111-111111111111"), "Общее", 10, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), nil
}

func (f *fakeRepo) ListForecastRunsByProject(ctx context.Context, projectID uuid.UUID) ([]RunSummary, error) {
	return nil, nil
}

func (f *fakeRepo) ListHistoricalCampaignsByOwner(ctx context.Context, ownerUserID uuid.UUID, category string) ([]HistoricalCampaign, error) {
	return f.historicalCampaigns, nil
}

func (f *fakeRepo) ListCompletedProjectHistoricalCampaignsByOwnerAndCategory(ctx context.Context, ownerUserID uuid.UUID, category string, excludeProjectID uuid.UUID) ([]HistoricalCampaign, error) {
	return nil, nil
}

func (f *fakeRepo) CreateHistoricalCampaign(ctx context.Context, ownerUserID uuid.UUID, req UpsertHistoricalCampaignRequest) (HistoricalCampaign, error) {
	return HistoricalCampaign{}, nil
}

func (f *fakeRepo) UpdateHistoricalCampaign(ctx context.Context, ownerUserID, campaignID uuid.UUID, req UpsertHistoricalCampaignRequest) (HistoricalCampaign, error) {
	return HistoricalCampaign{}, nil
}

func (f *fakeRepo) DeleteHistoricalCampaign(ctx context.Context, ownerUserID, campaignID uuid.UUID) error {
	return nil
}

func (f *fakeRepo) GetForecastRunByID(ctx context.Context, runID uuid.UUID) (sqlc.ForecastRun, error) {
	return f.lastRun, nil
}

func (f *fakeRepo) GetLatestForecastRunByProject(ctx context.Context, projectID uuid.UUID) (sqlc.ForecastRun, error) {
	return f.lastRun, nil
}

func (f *fakeRepo) ListForecastPointsByRun(ctx context.Context, runID uuid.UUID) ([]sqlc.ForecastPoint, error) {
	return f.pointsByRun[runID], nil
}

func (f *fakeRepo) DeleteForecastRun(ctx context.Context, projectID, runID uuid.UUID) error {
	return nil
}

func TestRunStoresAndLoadsForecastRun(t *testing.T) {
	setForecastTodayForTest(t, time.Date(2026, 2, 3, 0, 0, 0, 0, time.UTC))

	projectID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	repo := &fakeRepo{
		history: []sqlc.ActualIncomeDaily{
			{IncomeDate: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), Amount: 100},
			{IncomeDate: time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC), Amount: 150},
			{IncomeDate: time.Date(2026, 2, 3, 0, 0, 0, 0, time.UTC), Amount: 140},
		},
	}
	client := &fakeAnalyticsClient{response: AnalyticsResponse{
		ModelVersion: "v1",
		HorizonDays:  7,
		Forecast: []AnalyticsPoint{
			{Date: "2026-02-04", Yhat: 160, P10: 120, P90: 210},
			{Date: "2026-02-05", Yhat: 158, P10: 118, P90: 208},
		},
	}}
	svc := NewService(repo, client)

	created, err := svc.Run(context.Background(), projectID, 7)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if created.Run.ProjectID != projectID {
		t.Fatalf("stored run project mismatch")
	}
	if len(created.Points) != 2 {
		t.Fatalf("expected 2 forecast points, got %d", len(created.Points))
	}
	if client.lastReq.CurrentIncomeToDate != 390 {
		t.Fatalf("expected current_income_to_date=390, got %v", client.lastReq.CurrentIncomeToDate)
	}

	byID, err := svc.GetByID(context.Background(), projectID, created.Run.ID)
	if err != nil {
		t.Fatalf("get by id failed: %v", err)
	}
	if len(byID.Points) != 2 {
		t.Fatalf("expected 2 points from GetByID, got %d", len(byID.Points))
	}

	latest, err := svc.GetLatest(context.Background(), projectID)
	if err != nil {
		t.Fatalf("get latest failed: %v", err)
	}
	if latest.Run.ID != created.Run.ID {
		t.Fatalf("expected latest run id %s, got %s", created.Run.ID, latest.Run.ID)
	}
}

func TestRunReturnsNoRemainingForecastWhenCampaignFilled(t *testing.T) {
	setForecastTodayForTest(t, time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC))

	projectID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	repo := &fakeRepo{
		history: []sqlc.ActualIncomeDaily{
			{IncomeDate: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), Amount: 100},
			{IncomeDate: time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC), Amount: 150},
			{IncomeDate: time.Date(2026, 2, 3, 0, 0, 0, 0, time.UTC), Amount: 140},
			{IncomeDate: time.Date(2026, 2, 4, 0, 0, 0, 0, time.UTC), Amount: 140},
			{IncomeDate: time.Date(2026, 2, 5, 0, 0, 0, 0, time.UTC), Amount: 140},
			{IncomeDate: time.Date(2026, 2, 6, 0, 0, 0, 0, time.UTC), Amount: 140},
			{IncomeDate: time.Date(2026, 2, 7, 0, 0, 0, 0, time.UTC), Amount: 140},
			{IncomeDate: time.Date(2026, 2, 8, 0, 0, 0, 0, time.UTC), Amount: 140},
			{IncomeDate: time.Date(2026, 2, 9, 0, 0, 0, 0, time.UTC), Amount: 140},
			{IncomeDate: time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC), Amount: 140},
		},
	}
	client := &fakeAnalyticsClient{}
	svc := NewService(repo, client)

	_, err := svc.Run(context.Background(), projectID, 10)
	if !errors.Is(err, ErrNoRemainingForecast) {
		t.Fatalf("expected ErrNoRemainingForecast, got %v", err)
	}
}

func TestRunIgnoresIncomeOutsideCampaignWindow(t *testing.T) {
	setForecastTodayForTest(t, time.Date(2026, 2, 3, 0, 0, 0, 0, time.UTC))

	projectID := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	repo := &fakeRepo{
		history: []sqlc.ActualIncomeDaily{
			{IncomeDate: time.Date(2026, 1, 28, 0, 0, 0, 0, time.UTC), Amount: 700}, // stale row before campaign
			{IncomeDate: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), Amount: 100},
			{IncomeDate: time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC), Amount: 150},
			{IncomeDate: time.Date(2026, 2, 3, 0, 0, 0, 0, time.UTC), Amount: 140},
			{IncomeDate: time.Date(2026, 2, 12, 0, 0, 0, 0, time.UTC), Amount: 999}, // stale row after campaign
		},
	}
	client := &fakeAnalyticsClient{response: AnalyticsResponse{
		ModelVersion: "v1",
		HorizonDays:  7,
		Forecast: []AnalyticsPoint{
			{Date: "2026-02-04", Yhat: 160, P10: 120, P90: 210},
		},
	}}
	svc := NewService(repo, client)

	_, err := svc.Run(context.Background(), projectID, 10)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if client.lastReq.CurrentIncomeToDate != 390 {
		t.Fatalf("expected filtered current_income_to_date=390, got %v", client.lastReq.CurrentIncomeToDate)
	}
	if len(client.lastReq.History) != 3 {
		t.Fatalf("expected 3 observed calendar days, got %d", len(client.lastReq.History))
	}
	if client.lastReq.HorizonDays != 7 {
		t.Fatalf("expected horizon reduced to 7 remaining days, got %d", client.lastReq.HorizonDays)
	}
}

func TestRunBuildsDenseObservedHistoryAcrossCalendarGaps(t *testing.T) {
	setForecastTodayForTest(t, time.Date(2026, 2, 5, 0, 0, 0, 0, time.UTC))

	projectID := uuid.MustParse("88888888-8888-8888-8888-888888888888")
	repo := &fakeRepo{
		history: []sqlc.ActualIncomeDaily{
			{IncomeDate: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), Amount: 100},
			{IncomeDate: time.Date(2026, 2, 3, 0, 0, 0, 0, time.UTC), Amount: 40},
		},
	}
	client := &fakeAnalyticsClient{response: AnalyticsResponse{
		ModelVersion: "v1",
		HorizonDays:  5,
		Forecast: []AnalyticsPoint{
			{Date: "2026-02-06", Yhat: 50, P10: 40, P90: 60},
		},
	}}
	svc := NewService(repo, client)

	_, err := svc.Run(context.Background(), projectID, 7)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	if client.lastReq.HorizonDays != 7 {
		t.Fatalf("expected horizon reduced to 7 remaining days after latest actual entry, got %d", client.lastReq.HorizonDays)
	}
	if len(client.lastReq.History) != 3 {
		t.Fatalf("expected dense 3-day observed history, got %d points", len(client.lastReq.History))
	}

	want := []HistoryPoint{
		{Date: "2026-02-01", Amount: 100},
		{Date: "2026-02-02", Amount: 0},
		{Date: "2026-02-03", Amount: 40},
	}
	for i, point := range want {
		if client.lastReq.History[i] != point {
			t.Fatalf("history[%d] = %+v, want %+v", i, client.lastReq.History[i], point)
		}
	}
	if client.lastReq.CurrentIncomeToDate != 140 {
		t.Fatalf("expected current_income_to_date=140, got %v", client.lastReq.CurrentIncomeToDate)
	}
}

func TestRunUsesLatestActualEnteredDateInsteadOfTodayForObservedBoundary(t *testing.T) {
	setForecastTodayForTest(t, time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC))

	projectID := uuid.MustParse("99999999-9999-9999-9999-999999999999")
	repo := &fakeRepo{
		history: []sqlc.ActualIncomeDaily{
			{IncomeDate: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), Amount: 50},
			{IncomeDate: time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC), Amount: 50},
			{IncomeDate: time.Date(2026, 2, 3, 0, 0, 0, 0, time.UTC), Amount: 50},
			{IncomeDate: time.Date(2026, 2, 4, 0, 0, 0, 0, time.UTC), Amount: 50},
			{IncomeDate: time.Date(2026, 2, 5, 0, 0, 0, 0, time.UTC), Amount: 50},
		},
	}
	client := &fakeAnalyticsClient{response: AnalyticsResponse{
		ModelVersion: "v1",
		HorizonDays:  5,
		Forecast: []AnalyticsPoint{
			{Date: "2026-02-06", Yhat: 60, P10: 40, P90: 70},
		},
	}}
	svc := NewService(repo, client)

	_, err := svc.Run(context.Background(), projectID, 10)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	if len(client.lastReq.History) != 5 {
		t.Fatalf("expected 5 observed days from latest entered income date, got %d", len(client.lastReq.History))
	}
	if client.lastReq.CurrentIncomeToDate != 250 {
		t.Fatalf("expected current_income_to_date=250, got %v", client.lastReq.CurrentIncomeToDate)
	}
	if client.lastReq.HorizonDays != 5 {
		t.Fatalf("expected horizon reduced to 5 remaining days, got %d", client.lastReq.HorizonDays)
	}
	lastHistoryDate := client.lastReq.History[len(client.lastReq.History)-1].Date
	if lastHistoryDate != "2026-02-05" {
		t.Fatalf("expected latest observed history date 2026-02-05, got %s", lastHistoryDate)
	}
}

func TestRunFallsBackToTodayWhenNoActualIncomeRowsExist(t *testing.T) {
	setForecastTodayForTest(t, time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC))

	projectID := uuid.MustParse("12121212-1212-1212-1212-121212121212")
	repo := &fakeRepo{}
	client := &fakeAnalyticsClient{response: AnalyticsResponse{
		ModelVersion: "v1",
		HorizonDays:  8,
		Forecast: []AnalyticsPoint{
			{Date: "2026-02-03", Yhat: 10, P10: 8, P90: 14},
		},
	}}
	svc := NewService(repo, client)

	_, err := svc.Run(context.Background(), projectID, 10)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	if len(client.lastReq.History) != 2 {
		t.Fatalf("expected fallback observed days = 2 from current date, got %d", len(client.lastReq.History))
	}
	if client.lastReq.HorizonDays != 8 {
		t.Fatalf("expected horizon reduced to 8 remaining days, got %d", client.lastReq.HorizonDays)
	}
}

func TestRunReturnsAnalyticsInvalidResponseWhenDateIsMalformed(t *testing.T) {
	setForecastTodayForTest(t, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC))

	projectID := uuid.MustParse("55555555-5555-5555-5555-555555555555")
	repo := &fakeRepo{
		history: []sqlc.ActualIncomeDaily{{IncomeDate: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), Amount: 100}},
	}
	client := &fakeAnalyticsClient{response: AnalyticsResponse{
		ModelVersion: "v1",
		HorizonDays:  1,
		Forecast:     []AnalyticsPoint{{Date: "bad-date", Yhat: 10, P10: 8, P90: 12}},
	}}
	svc := NewService(repo, client)

	_, err := svc.Run(context.Background(), projectID, 1)
	if !errors.Is(err, ErrAnalyticsInvalidResponse) {
		t.Fatalf("expected ErrAnalyticsInvalidResponse, got %v", err)
	}
}

func TestRunClampsScenarioOrderingFromAnalyticsResponse(t *testing.T) {
	setForecastTodayForTest(t, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC))

	projectID := uuid.MustParse("77777777-7777-7777-7777-777777777777")
	repo := &fakeRepo{
		history: []sqlc.ActualIncomeDaily{{IncomeDate: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), Amount: 100}},
	}
	client := &fakeAnalyticsClient{response: AnalyticsResponse{
		ModelVersion: "v1",
		HorizonDays:  1,
		Forecast:     []AnalyticsPoint{{Date: "2026-02-02", Yhat: 10, P10: 15, P90: 8}},
	}}
	svc := NewService(repo, client)

	result, err := svc.Run(context.Background(), projectID, 1)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(result.Points) != 1 {
		t.Fatalf("points len = %d, want 1", len(result.Points))
	}
	if result.Points[0].P10 != result.Points[0].Yhat {
		t.Fatalf("expected P10 clamped to yhat, got p10=%v yhat=%v", result.Points[0].P10, result.Points[0].Yhat)
	}
	if result.Points[0].P90 != result.Points[0].Yhat {
		t.Fatalf("expected P90 clamped to yhat, got p90=%v yhat=%v", result.Points[0].P90, result.Points[0].Yhat)
	}
}

func TestRunReturnsAnalyticsClientErrorsUnchanged(t *testing.T) {
	setForecastTodayForTest(t, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC))

	projectID := uuid.MustParse("66666666-6666-6666-6666-666666666666")
	repo := &fakeRepo{
		history: []sqlc.ActualIncomeDaily{{IncomeDate: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), Amount: 100}},
	}
	client := &fakeAnalyticsClient{err: ErrAnalyticsUnavailable}
	svc := NewService(repo, client)

	_, err := svc.Run(context.Background(), projectID, 1)
	if !errors.Is(err, ErrAnalyticsUnavailable) {
		t.Fatalf("expected ErrAnalyticsUnavailable, got %v", err)
	}
}

func TestRunReturnsAnalyticsInvalidRequestUnchanged(t *testing.T) {
	setForecastTodayForTest(t, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC))

	projectID := uuid.MustParse("99999999-9999-9999-9999-999999999999")
	repo := &fakeRepo{
		history: []sqlc.ActualIncomeDaily{{IncomeDate: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), Amount: -25}},
	}
	client := &fakeAnalyticsClient{err: ErrAnalyticsInvalidRequest}
	svc := NewService(repo, client)

	_, err := svc.Run(context.Background(), projectID, 1)
	if !errors.Is(err, ErrAnalyticsInvalidRequest) {
		t.Fatalf("expected ErrAnalyticsInvalidRequest, got %v", err)
	}
}
