package projects

import (
	"testing"
	"time"

	"isbudget/backend/internal/db/sqlc"

	"github.com/google/uuid"
)

func TestBuildCategoryFolderSummariesCountsProjectsAndHistoricalCampaigns(t *testing.T) {
	now := time.Now().UTC()
	projects := []sqlc.ListProjectsForUserRow{
		{ID: uuid.New(), Category: "Launch", Status: ProjectStatusPreparation, CreatedAt: now, UpdatedAt: now},
		{ID: uuid.New(), Category: "Launch", Status: ProjectStatusActive, CreatedAt: now, UpdatedAt: now},
		{ID: uuid.New(), Category: "Launch", Status: ProjectStatusCompleted, CreatedAt: now, UpdatedAt: now},
		{ID: uuid.New(), Category: "", Status: ProjectStatusActive, CreatedAt: now, UpdatedAt: now},
	}
	historicalCounts := []historicalCampaignCategoryCount{
		{Category: "Launch", Count: 3},
		{Category: "", Count: 2},
	}

	got := buildCategoryFolderSummaries(projects, historicalCounts)
	if len(got) != 2 {
		t.Fatalf("expected 2 summaries, got %d", len(got))
	}

	byCategory := make(map[string]CategoryFolderSummary, len(got))
	for _, item := range got {
		byCategory[item.Category] = item
	}

	launch := byCategory["Launch"]
	if launch.ActiveProjectsCount != 2 || launch.CompletedProjectsCount != 1 || launch.HistoricalCampaignsCount != 3 {
		t.Fatalf("unexpected Launch summary: %+v", launch)
	}

	general := byCategory["Общее"]
	if general.ActiveProjectsCount != 1 || general.CompletedProjectsCount != 0 || general.HistoricalCampaignsCount != 2 {
		t.Fatalf("unexpected general summary: %+v", general)
	}
}

func TestBuildCategoryFolderSummariesIncludesHistoricalOnlyCategory(t *testing.T) {
	got := buildCategoryFolderSummaries(nil, []historicalCampaignCategoryCount{
		{Category: "Events", Count: 5},
	})

	if len(got) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(got))
	}
	if got[0].Category != "Events" {
		t.Fatalf("expected Events category, got %q", got[0].Category)
	}
	if got[0].ActiveProjectsCount != 0 || got[0].CompletedProjectsCount != 0 || got[0].HistoricalCampaignsCount != 5 {
		t.Fatalf("unexpected summary: %+v", got[0])
	}
}
