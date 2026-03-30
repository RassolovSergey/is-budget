package dashboard

import (
	"testing"
	"time"

	"isbudget/backend/internal/db/sqlc"
)

func TestResolveDashboardIncomeRangeUsesFullCampaignWindowForActiveProject(t *testing.T) {
	activatedAt := time.Date(2026, 2, 1, 14, 0, 0, 0, time.UTC)
	project := sqlc.GetProjectByIDRow{
		DurationDays: 10,
		ActivatedAt:  &activatedAt,
	}

	from, to := resolveDashboardIncomeRange(project, 30, time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC))
	if from.Format("2006-01-02") != "2026-02-01" {
		t.Fatalf("expected campaign range start 2026-02-01, got %s", from.Format("2006-01-02"))
	}
	if to.Format("2006-01-02") != "2026-02-10" {
		t.Fatalf("expected campaign range end 2026-02-10, got %s", to.Format("2006-01-02"))
	}
}

func TestResolveDashboardIncomeRangeFallsBackToRollingWindowWithoutIncomeRows(t *testing.T) {
	project := sqlc.GetProjectByIDRow{}

	from, to := resolveDashboardIncomeRange(project, 30, time.Date(2026, 2, 2, 12, 0, 0, 0, time.UTC))
	if to.Format("2006-01-02") != "2026-02-02" {
		t.Fatalf("expected rolling window end 2026-02-02, got %s", to.Format("2006-01-02"))
	}
	if from.Format("2006-01-02") != "2026-01-04" {
		t.Fatalf("expected rolling window start 2026-01-04, got %s", from.Format("2006-01-02"))
	}
}
