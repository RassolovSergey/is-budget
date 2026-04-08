package pipeline

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"isbudget/backend/internal/db/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type fakeRepo struct {
	project sqlc.Project
	summary sqlc.FinanceSummary
	blocks  []sqlc.AllocationBlock
}

func (f *fakeRepo) WithTx(ctx context.Context, fn func(repo Repository) error) error {
	return fn(f)
}

func (f *fakeRepo) GetProjectByID(ctx context.Context, id uuid.UUID) (sqlc.Project, error) {
	if f.project.ID != id {
		return sqlc.Project{}, pgx.ErrNoRows
	}
	return f.project, nil
}

func (f *fakeRepo) GetFinanceSummaryByProject(ctx context.Context, projectID uuid.UUID) (sqlc.FinanceSummary, error) {
	return f.summary, nil
}

func (f *fakeRepo) ListAllocationBlocksByProject(ctx context.Context, projectID uuid.UUID) ([]sqlc.AllocationBlock, error) {
	items := append([]sqlc.AllocationBlock(nil), f.blocks...)
	slices.SortFunc(items, func(a, b sqlc.AllocationBlock) int {
		if a.LevelIndex == b.LevelIndex {
			if a.OrderIndex == b.OrderIndex {
				return a.CreatedAt.Compare(b.CreatedAt)
			}
			if a.OrderIndex < b.OrderIndex {
				return -1
			}
			return 1
		}
		if a.LevelIndex < b.LevelIndex {
			return -1
		}
		return 1
	})
	return items, nil
}

func (f *fakeRepo) GetAllocationBlockByProject(ctx context.Context, arg sqlc.GetAllocationBlockByProjectParams) (sqlc.AllocationBlock, error) {
	for _, block := range f.blocks {
		if block.ProjectID == arg.ProjectID && block.ID == arg.ID {
			return block, nil
		}
	}
	return sqlc.AllocationBlock{}, pgx.ErrNoRows
}

func (f *fakeRepo) CreateAllocationBlock(ctx context.Context, arg sqlc.CreateAllocationBlockParams) (sqlc.AllocationBlock, error) {
	item := sqlc.AllocationBlock{
		ID:         uuid.New(),
		ProjectID:  arg.ProjectID,
		Name:       arg.Name,
		Type:       arg.Type,
		LevelIndex: arg.LevelIndex,
		OrderIndex: arg.OrderIndex,
		Enabled:    arg.Enabled,
		ValueType:  arg.ValueType,
		Value:      arg.Value,
		AppliesTo:  arg.AppliesTo,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	f.blocks = append(f.blocks, item)
	return item, nil
}

func (f *fakeRepo) UpdateAllocationBlock(ctx context.Context, arg sqlc.UpdateAllocationBlockParams) (sqlc.AllocationBlock, error) {
	for i, block := range f.blocks {
		if block.ProjectID == arg.ProjectID && block.ID == arg.ID {
			block.Name = arg.Name
			block.Type = arg.Type
			block.LevelIndex = arg.LevelIndex
			block.Enabled = arg.Enabled
			block.ValueType = arg.ValueType
			block.Value = arg.Value
			block.AppliesTo = arg.AppliesTo
			block.UpdatedAt = time.Now().UTC()
			f.blocks[i] = block
			return block, nil
		}
	}
	return sqlc.AllocationBlock{}, pgx.ErrNoRows
}

func (f *fakeRepo) DeleteAllocationBlock(ctx context.Context, arg sqlc.DeleteAllocationBlockParams) (int64, error) {
	for i, block := range f.blocks {
		if block.ProjectID == arg.ProjectID && block.ID == arg.ID {
			f.blocks = append(f.blocks[:i], f.blocks[i+1:]...)
			return 1, nil
		}
	}
	return 0, nil
}

func (f *fakeRepo) UpdateAllocationBlockPlacement(ctx context.Context, arg sqlc.UpdateAllocationBlockPlacementParams) error {
	for i, block := range f.blocks {
		if block.ProjectID == arg.ProjectID && block.ID == arg.ID {
			block.LevelIndex = arg.LevelIndex
			block.OrderIndex = arg.OrderIndex
			block.UpdatedAt = time.Now().UTC()
			f.blocks[i] = block
			return nil
		}
	}
	return pgx.ErrNoRows
}

func TestComputePreviewLevelBasedCalculation(t *testing.T) {
	projectID := uuid.New()
	blocks := []sqlc.AllocationBlock{
		makeBlock(projectID, 1, 1, BlockTypeTax, ValueTypePercent, 10, AppliesToGross, true),
		makeBlock(projectID, 2, 1, BlockTypeActualExpense, ValueTypeFixed, 200, AppliesToCurrentBalance, true),
		makeBlock(projectID, 3, 2, BlockTypeAdminDistribution, ValueTypePercent, 10, AppliesToCurrentBalance, true),
		makeBlock(projectID, 4, 3, BlockTypeTeamDistribution, ValueTypePercent, 50, AppliesToCurrentBalance, true),
		makeBlock(projectID, 5, 4, BlockTypeTeamDistribution, ValueTypeFixed, 0, AppliesToCurrentBalance, true),
	}

	preview := ComputePreview(projectID, "EUR", 1000, "manual_override", blocks)

	if len(preview.Levels) != 4 {
		t.Fatalf("expected 4 levels, got %d", len(preview.Levels))
	}
	if preview.Levels[0].IncomingAmount != 1000 || preview.Levels[0].RemainingAmount != 700 {
		t.Fatalf("unexpected level 1 summary: %+v", preview.Levels[0])
	}
	if preview.Levels[1].IncomingAmount != 700 || preview.Levels[1].RemainingAmount != 630 {
		t.Fatalf("unexpected level 2 summary: %+v", preview.Levels[1])
	}
	if preview.FinalTeamPoolAmount != 630 {
		t.Fatalf("expected 630 team pool, got %.2f", preview.FinalTeamPoolAmount)
	}
	if preview.FinalRemainingAmount != 0 {
		t.Fatalf("expected 0 remaining, got %.2f", preview.FinalRemainingAmount)
	}
	if preview.TotalDeductedBeforeTeamPool != 370 {
		t.Fatalf("expected 370 deducted before team pool, got %.2f", preview.TotalDeductedBeforeTeamPool)
	}
}

func TestComputePreviewLevelUsesSameIncomingAmountForCurrentBalanceBlocks(t *testing.T) {
	projectID := uuid.New()
	blocks := []sqlc.AllocationBlock{
		makeBlock(projectID, 1, 2, BlockTypeActualExpense, ValueTypeFixed, 100, AppliesToCurrentBalance, true),
		makeBlock(projectID, 2, 2, BlockTypeCustomFee, ValueTypePercent, 10, AppliesToCurrentBalance, true),
	}

	preview := ComputePreview(projectID, "EUR", 1000, "manual_override", blocks)
	if len(preview.Levels) != 1 {
		t.Fatalf("expected one level, got %d", len(preview.Levels))
	}
	level := preview.Levels[0]
	if level.IncomingAmount != 1000 {
		t.Fatalf("expected incoming level amount 1000, got %.2f", level.IncomingAmount)
	}
	if level.Steps[0].InputAmount != 1000 || level.Steps[1].InputAmount != 1000 {
		t.Fatalf("expected same input amount for level steps, got %+v", level.Steps)
	}
	if level.TotalAllocated != 200 || level.RemainingAmount != 800 {
		t.Fatalf("unexpected level allocation summary: %+v", level)
	}
}

func TestComputePreviewSkipsDisabledBlocks(t *testing.T) {
	projectID := uuid.New()
	blocks := []sqlc.AllocationBlock{
		makeBlock(projectID, 1, 1, BlockTypeTax, ValueTypePercent, 10, AppliesToGross, false),
		makeBlock(projectID, 2, 1, BlockTypeActualExpense, ValueTypeFixed, 80, AppliesToCurrentBalance, true),
	}

	preview := ComputePreview(projectID, "EUR", 500, "manual_override", blocks)
	if len(preview.Steps) != 1 {
		t.Fatalf("expected 1 active step, got %d", len(preview.Steps))
	}
	if preview.Steps[0].AllocatedAmount != 80 {
		t.Fatalf("expected only actual expense block to apply, got %+v", preview.Steps[0])
	}
}

func TestComputePreviewPreventsNegativeRemainingAmount(t *testing.T) {
	projectID := uuid.New()
	blocks := []sqlc.AllocationBlock{
		makeBlock(projectID, 1, 1, BlockTypeActualExpense, ValueTypeFixed, 150, AppliesToCurrentBalance, true),
	}

	preview := ComputePreview(projectID, "EUR", 100, "manual_override", blocks)
	if preview.Steps[0].AllocatedAmount != 100 {
		t.Fatalf("expected allocation to be capped at remaining amount, got %+v", preview.Steps[0])
	}
	if preview.FinalRemainingAmount != 0 {
		t.Fatalf("expected zero remaining, got %.2f", preview.FinalRemainingAmount)
	}
}

func TestComputePreviewTeamDistributionFullRemainderBehavior(t *testing.T) {
	projectID := uuid.New()
	blocks := []sqlc.AllocationBlock{
		makeBlock(projectID, 1, 1, BlockTypeCustomFee, ValueTypeFixed, 120, AppliesToCurrentBalance, true),
		makeBlock(projectID, 2, 2, BlockTypeTeamDistribution, ValueTypeFixed, 0, AppliesToCurrentBalance, true),
	}

	preview := ComputePreview(projectID, "EUR", 500, "manual_override", blocks)
	if preview.FinalTeamPoolAmount != 380 {
		t.Fatalf("expected remainder team pool 380, got %.2f", preview.FinalTeamPoolAmount)
	}
	if preview.Steps[1].AllocatedAmount != 380 {
		t.Fatalf("expected remainder step to capture all leftover, got %+v", preview.Steps[1])
	}
	if preview.FinalRemainingAmount != 0 {
		t.Fatalf("expected full remainder to zero out final balance, got %.2f", preview.FinalRemainingAmount)
	}
}

func TestReorderDeleteAndPreviewBehavior(t *testing.T) {
	projectID := uuid.New()
	blockA := makeBlock(projectID, 1, 1, BlockTypeActualExpense, ValueTypeFixed, 100, AppliesToCurrentBalance, true)
	blockB := makeBlock(projectID, 2, 2, BlockTypeTax, ValueTypePercent, 50, AppliesToGross, true)

	repo := &fakeRepo{
		project: sqlc.Project{ID: projectID, CurrencyCode: "EUR"},
		summary: sqlc.FinanceSummary{IncomeTotal: 1000},
		blocks:  []sqlc.AllocationBlock{blockA, blockB},
	}
	svc := NewService(repo)

	err := svc.Reorder(context.Background(), projectID, []ReorderItemInput{
		{ID: blockB.ID, LevelIndex: 1, OrderIndex: 1},
		{ID: blockA.ID, LevelIndex: 1, OrderIndex: 2},
	}, nil)
	if err != nil {
		t.Fatalf("reorder failed: %v", err)
	}

	blocks, err := svc.List(context.Background(), projectID)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if blocks[0].ID != blockB.ID || blocks[0].LevelIndex != 1 || blocks[0].OrderIndex != 1 {
		t.Fatalf("expected reordered block first, got %+v", blocks[0])
	}

	preview, err := svc.Preview(context.Background(), projectID, PreviewInput{})
	if err != nil {
		t.Fatalf("preview failed: %v", err)
	}
	if preview.Steps[0].BlockID != blockB.ID {
		t.Fatalf("expected preview to use reordered block first, got %+v", preview.Steps[0])
	}

	if err := svc.Delete(context.Background(), projectID, blockB.ID); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	blocks, err = svc.List(context.Background(), projectID)
	if err != nil {
		t.Fatalf("list after delete failed: %v", err)
	}
	if len(blocks) != 1 || blocks[0].OrderIndex != 1 {
		t.Fatalf("expected remaining block to be compacted to order 1, got %+v", blocks)
	}
}

func TestCreateRejectsSecondFullRemainderDistribution(t *testing.T) {
	projectID := uuid.New()
	repo := &fakeRepo{
		project: sqlc.Project{ID: projectID, CurrencyCode: "EUR"},
		blocks:  []sqlc.AllocationBlock{makeBlock(projectID, 1, 3, BlockTypeTeamDistribution, ValueTypeFixed, 0, AppliesToCurrentBalance, true)},
	}
	svc := NewService(repo)

	_, err := svc.Create(context.Background(), projectID, UpsertBlockInput{
		Name:                 "Распределение остатка 2",
		Type:                 BlockTypeTeamDistribution,
		LevelIndex:           4,
		Enabled:              true,
		ValueType:            ValueTypeFixed,
		Value:                0,
		AppliesTo:            AppliesToCurrentBalance,
		TeamDistributionMode: TeamDistributionModeFullRemainder,
	})
	if !errors.Is(err, ErrAllocationRemainderConflict) {
		t.Fatalf("expected remainder conflict, got %v", err)
	}
}

func TestNormalizeInputMapsLegacyTypes(t *testing.T) {
	svc := NewService(&fakeRepo{})

	got, err := svc.normalizeInput(UpsertBlockInput{
		Name:       "Legacy tax",
		Type:       legacyBlockTypeTaxPercent,
		LevelIndex: 1,
		Enabled:    true,
		ValueType:  ValueTypePercent,
		Value:      12,
		AppliesTo:  AppliesToGross,
	})
	if err != nil {
		t.Fatalf("normalizeInput returned error: %v", err)
	}
	if got.Type != BlockTypeTax {
		t.Fatalf("expected legacy tax to map to %q, got %q", BlockTypeTax, got.Type)
	}
}

func makeBlock(projectID uuid.UUID, orderIndex, levelIndex int32, blockType, valueType string, value float64, appliesTo string, enabled bool) sqlc.AllocationBlock {
	now := time.Now().UTC()
	return sqlc.AllocationBlock{
		ID:         uuid.New(),
		ProjectID:  projectID,
		Name:       blockType,
		Type:       blockType,
		LevelIndex: levelIndex,
		OrderIndex: orderIndex,
		Enabled:    enabled,
		ValueType:  valueType,
		Value:      value,
		AppliesTo:  appliesTo,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}
