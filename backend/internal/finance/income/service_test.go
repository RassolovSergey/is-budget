package income

import (
	"testing"
	"time"

	"isbudget/backend/internal/db/sqlc"

	"github.com/google/uuid"
)

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

	upserts := buildDailyDeltaUpserts(projectID, 0, points)
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

	upserts := buildDailyDeltaUpserts(projectID, 0, points)
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

	upserts := buildDailyDeltaUpserts(projectID, 0, points)
	if len(upserts) != 1 {
		t.Fatalf("expected single upsert, got %d", len(upserts))
	}
	var _ sqlc.UpsertIncomeDailyParams = upserts[0]
}
