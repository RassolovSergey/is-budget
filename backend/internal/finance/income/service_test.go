package income

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"isbudget/backend/internal/db/sqlc"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func round2Test(value float64) float64 {
	return float64(int((value+0.005)*100)) / 100
}

func reconstructCampaignCumulative(
	start time.Time,
	duration int,
	rows []sqlc.UpsertIncomeDailyParams,
) []float64 {
	dailyByDate := make(map[string]float64, len(rows))
	for _, row := range rows {
		dailyByDate[row.IncomeDate.Format("2006-01-02")] += row.Amount
	}

	out := make([]float64, 0, duration)
	running := 0.0
	cursor := start
	for i := 0; i < duration; i += 1 {
		running = round2Test(running + dailyByDate[cursor.Format("2006-01-02")])
		out = append(out, running)
		cursor = cursor.AddDate(0, 0, 1)
	}
	return out
}

func TestNormalizeCumulativePointsDeduplicatesAndSorts(t *testing.T) {
	t1 := time.Date(2026, 2, 10, 14, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 2, 8, 9, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 2, 9, 9, 0, 0, 0, time.UTC)

	vA := 100.0
	vB := 200.0
	vC := 180.0
	points := []CumulativePointInput{
		{Date: t1, CumulativeAmount: &vA},
		{Date: t2, CumulativeAmount: &vB},
		// Same date as t1, last value should win.
		{Date: t1, CumulativeAmount: &vC},
		{Date: t3, CumulativeAmount: nil},
	}

	normalized := normalizeCumulativePoints(points)
	if len(normalized) != 3 {
		t.Fatalf("expected 3 normalized points, got %d", len(normalized))
	}
	if normalized[0].Date.Format("2006-01-02") != "2026-02-08" {
		t.Fatalf("expected first date 2026-02-08, got %s", normalized[0].Date.Format("2006-01-02"))
	}
	if normalized[1].Date.Format("2006-01-02") != "2026-02-09" {
		t.Fatalf("expected second date 2026-02-09, got %s", normalized[1].Date.Format("2006-01-02"))
	}
	if normalized[2].Date.Format("2006-01-02") != "2026-02-10" {
		t.Fatalf("expected third date 2026-02-10, got %s", normalized[2].Date.Format("2006-01-02"))
	}
	if normalized[2].CumulativeAmount == nil || *normalized[2].CumulativeAmount != 180 {
		t.Fatalf("expected deduped amount 180 on 2026-02-10, got %+v", normalized[2].CumulativeAmount)
	}
}

func TestBuildDailyDeltaUpsertsSupportsRefundDropsAndZeroDay(t *testing.T) {
	projectID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	v1 := 1200.0 // day1 cumulative
	v2 := 1400.0 // day2 cumulative (+200)
	v3 := 1360.0 // day3 cumulative (-40 refund)
	v4 := 1360.0 // day4 cumulative (+0, still persisted)

	points := []normalizedCumulativePoint{
		{Date: time.Date(2026, 2, 17, 0, 0, 0, 0, time.UTC), CumulativeAmount: &v1},
		{Date: time.Date(2026, 2, 18, 0, 0, 0, 0, time.UTC), CumulativeAmount: &v2},
		{Date: time.Date(2026, 2, 19, 0, 0, 0, 0, time.UTC), CumulativeAmount: &v3},
		{Date: time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC), CumulativeAmount: &v4},
	}

	upserts := buildDailyDeltaUpserts(projectID, points)
	if len(upserts) != 4 {
		t.Fatalf("expected 4 upserts, got %d", len(upserts))
	}

	want := []float64{1200, 200, -40, 0}
	for i := range want {
		if upserts[i].ProjectID != projectID {
			t.Fatalf("upsert[%d] has unexpected project id", i)
		}
		if upserts[i].Amount != want[i] {
			t.Fatalf("upsert[%d] expected amount %.2f, got %.2f", i, want[i], upserts[i].Amount)
		}
	}
}

func TestBuildDailyDeltaUpsertsSkipsClearedDay(t *testing.T) {
	projectID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	v1 := 300.0
	v2 := 450.0

	points := []normalizedCumulativePoint{
		{Date: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), CumulativeAmount: &v1},
		{Date: time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC), CumulativeAmount: nil},
		{Date: time.Date(2026, 2, 3, 0, 0, 0, 0, time.UTC), CumulativeAmount: &v2},
	}

	upserts := buildDailyDeltaUpserts(projectID, points)
	if len(upserts) != 2 {
		t.Fatalf("expected 2 persisted upserts (middle cleared), got %d", len(upserts))
	}

	if upserts[0].Amount != 300 {
		t.Fatalf("expected day1 delta 300, got %.2f", upserts[0].Amount)
	}
	// Day2 is cleared => no record, day3 compares to previous running(300): +150.
	if upserts[1].Amount != 150 {
		t.Fatalf("expected day3 delta 150, got %.2f", upserts[1].Amount)
	}
}

func TestBuildDailyDeltaUpsertsProducesSqlcParams(t *testing.T) {
	projectID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	v := 10.0
	points := []normalizedCumulativePoint{
		{Date: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), CumulativeAmount: &v},
	}

	upserts := buildDailyDeltaUpserts(projectID, points)
	if len(upserts) != 1 {
		t.Fatalf("expected single upsert, got %d", len(upserts))
	}
	var _ sqlc.UpsertIncomeDailyParams = upserts[0]
}

func TestBuildDailyDeltaUpsertsRoundTripsCampaignCumulativeSeries(t *testing.T) {
	projectID := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	v1 := 1000.0
	v2 := 2000.0
	v3 := 1800.0
	v4 := 2200.0
	points := []normalizedCumulativePoint{
		{Date: start, CumulativeAmount: &v1},
		{Date: start.AddDate(0, 0, 1), CumulativeAmount: &v2},
		{Date: start.AddDate(0, 0, 2), CumulativeAmount: &v3},
		{Date: start.AddDate(0, 0, 3), CumulativeAmount: &v4},
	}

	upserts := buildDailyDeltaUpserts(projectID, points)
	got := reconstructCampaignCumulative(start, 4, upserts)
	want := []float64{1000, 2000, 1800, 2200}

	if len(got) != len(want) {
		t.Fatalf("expected %d cumulative points, got %d", len(want), len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("cumulative[%d] expected %.2f, got %.2f", i, want[i], got[i])
		}
	}
}

func TestBuildDailyDeltaUpsertsFirstDayAlwaysStartsFromZeroBaseline(t *testing.T) {
	projectID := uuid.MustParse("55555555-5555-5555-5555-555555555555")
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	firstDay := 1000.0
	secondDay := 1200.0

	upserts := buildDailyDeltaUpserts(projectID, []normalizedCumulativePoint{
		{Date: start, CumulativeAmount: &firstDay},
		{Date: start.AddDate(0, 0, 1), CumulativeAmount: &secondDay},
	})

	if len(upserts) != 2 {
		t.Fatalf("expected 2 upserts, got %d", len(upserts))
	}
	if upserts[0].Amount != 1000 {
		t.Fatalf("expected day1 delta 1000 from zero baseline, got %.2f", upserts[0].Amount)
	}
}

func TestBuildDailyDeltaUpsertsKeepsRefundDeltaWithoutNegativeCumulativeInput(t *testing.T) {
	projectID := uuid.MustParse("66666666-6666-6666-6666-666666666666")
	start := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	v1 := 1000.0
	v2 := 800.0
	v3 := 1200.0
	points := []normalizedCumulativePoint{
		{Date: start, CumulativeAmount: &v1},
		{Date: start.AddDate(0, 0, 1), CumulativeAmount: &v2},
		{Date: start.AddDate(0, 0, 2), CumulativeAmount: &v3},
	}

	upserts := buildDailyDeltaUpserts(projectID, points)
	wantDaily := []float64{1000, -200, 400}
	if len(upserts) != len(wantDaily) {
		t.Fatalf("expected %d upserts, got %d", len(wantDaily), len(upserts))
	}
	for i := range wantDaily {
		if upserts[i].Amount != wantDaily[i] {
			t.Fatalf("daily[%d] expected %.2f, got %.2f", i, wantDaily[i], upserts[i].Amount)
		}
	}

	gotCumulative := reconstructCampaignCumulative(start, 3, upserts)
	wantCumulative := []float64{1000, 800, 1200}
	for i := range wantCumulative {
		if gotCumulative[i] != wantCumulative[i] {
			t.Fatalf("cumulative[%d] expected %.2f, got %.2f", i, wantCumulative[i], gotCumulative[i])
		}
		if gotCumulative[i] < 0 {
			t.Fatalf("cumulative[%d] should stay non-negative, got %.2f", i, gotCumulative[i])
		}
	}
}

func TestBuildDailyDeltaUpsertsIgnoresRowsOutsideCampaignRangeForDayOne(t *testing.T) {
	projectID := uuid.MustParse("77777777-7777-7777-7777-777777777777")
	start := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	staleBeforeCampaign := sqlc.UpsertIncomeDailyParams{
		ProjectID:  projectID,
		IncomeDate: start.AddDate(0, 0, -1),
		Amount:     2428,
	}
	v1 := 1000.0
	v2 := 2000.0
	v3 := 1800.0
	v4 := 2200.0
	newRows := buildDailyDeltaUpserts(projectID, []normalizedCumulativePoint{
		{Date: start, CumulativeAmount: &v1},
		{Date: start.AddDate(0, 0, 1), CumulativeAmount: &v2},
		{Date: start.AddDate(0, 0, 2), CumulativeAmount: &v3},
		{Date: start.AddDate(0, 0, 3), CumulativeAmount: &v4},
	})

	allRows := append([]sqlc.UpsertIncomeDailyParams{staleBeforeCampaign}, newRows...)
	got := reconstructCampaignCumulative(start, 4, allRows)
	want := []float64{1000, 2000, 1800, 2200}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("campaign cumulative[%d] expected %.2f, got %.2f", i, want[i], got[i])
		}
	}
}

func TestUpsertCumulativeRejectsNegativeCumulativeInput(t *testing.T) {
	handler := NewHandler(nil)
	projectID := uuid.NewString()
	req := httptest.NewRequest(http.MethodPut, "/api/projects/"+projectID+"/income-daily/cumulative", strings.NewReader(`{
  "points": [
    {"date": "2026-03-01", "cumulative_amount": -1}
  ]
}`))
	req.Header.Set("Content-Type", "application/json")

	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("projectId", projectID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	rec := httptest.NewRecorder()
	handler.UpsertCumulative(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for negative cumulative input, got %d", rec.Code)
	}

	var resp struct {
		Code  string `json:"code"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if resp.Code != "invalid_request" {
		t.Fatalf("expected invalid_request code, got %q", resp.Code)
	}
	if resp.Error == "" {
		t.Fatal("expected non-empty error message")
	}
}
