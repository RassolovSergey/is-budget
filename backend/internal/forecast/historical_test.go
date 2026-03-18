package forecast

import (
	"math"
	"testing"
)

func TestNormalizeHistoricalCampaignInput_AllowsNegativeDailyIncome(t *testing.T) {
	req := UpsertHistoricalCampaignRequest{
		Name:         "Campaign A",
		Category:     "3D Sets",
		DurationDays: 3,
		CurrencyCode: "EUR",
		StartedAt:    "2026-02-01",
		EndedAt:      "2026-02-03",
		Days: []HistoricalCampaignDay{
			{DayNumber: 1, IncomeAmount: 100},
			{DayNumber: 2, IncomeAmount: -40},
			{DayNumber: 3, IncomeAmount: 20},
		},
	}

	normalized, err := normalizeHistoricalCampaignInput(req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(normalized.Days) != 3 {
		t.Fatalf("expected 3 days, got %d", len(normalized.Days))
	}
	if normalized.Days[1].IncomeAmount != -40 {
		t.Fatalf("expected day 2 income -40, got %v", normalized.Days[1].IncomeAmount)
	}
}

func TestNormalizeHistoricalCampaignInput_RejectsNaNIncome(t *testing.T) {
	req := UpsertHistoricalCampaignRequest{
		Name:         "Campaign B",
		Category:     "3D Sets",
		DurationDays: 3,
		CurrencyCode: "EUR",
		StartedAt:    "2026-02-01",
		EndedAt:      "2026-02-03",
		Days: []HistoricalCampaignDay{
			{DayNumber: 1, IncomeAmount: 100},
			{DayNumber: 2, IncomeAmount: math.NaN()},
			{DayNumber: 3, IncomeAmount: 20},
		},
	}

	_, err := normalizeHistoricalCampaignInput(req)
	if err == nil {
		t.Fatal("expected error for NaN income, got nil")
	}
}
