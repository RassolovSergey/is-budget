package taxes

import (
	"testing"

	"isbudget/backend/internal/db/sqlc"

	"github.com/google/uuid"
)

func TestShiftTaxRulesForPrependShiftsAllExistingLevelsByOne(t *testing.T) {
	projectID := uuid.New()
	rules := []sqlc.ProjectTaxRule{
		{ID: uuid.New(), ProjectID: projectID, Name: "Level1", Percent: 10, Level: 1},
		{ID: uuid.New(), ProjectID: projectID, Name: "Level2", Percent: 12, Level: 2},
		{ID: uuid.New(), ProjectID: projectID, Name: "Level3", Percent: 15, Level: 3},
	}

	shifted := shiftTaxRulesForPrepend(rules)
	if len(shifted) != 3 {
		t.Fatalf("expected 3 shifted rules, got %d", len(shifted))
	}
	if shifted[0].Level != 2 || shifted[1].Level != 3 || shifted[2].Level != 4 {
		t.Fatalf("expected shifted levels 2,3,4 got %d,%d,%d", shifted[0].Level, shifted[1].Level, shifted[2].Level)
	}
}

func TestShiftTaxRulesForPrependLeavesEmptySliceUntouched(t *testing.T) {
	shifted := shiftTaxRulesForPrepend(nil)
	if len(shifted) != 0 {
		t.Fatalf("expected empty shifted slice, got %d", len(shifted))
	}
}
