package pipeline

import (
	"context"
	"errors"
	"sort"
	"strings"

	"isbudget/backend/internal/db/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	BlockTypeTax                = "tax"
	BlockTypeActualExpense      = "actual_expense"
	BlockTypeCustomFee          = "custom_fee"
	BlockTypeStudioDistribution = "studio_distribution"
	BlockTypeAdminDistribution  = "admin_distribution"
	BlockTypeTeamDistribution   = "team_distribution"

	legacyBlockTypeTaxPercent        = "tax_percent"
	legacyBlockTypeFixedCost         = "fixed_cost"
	legacyBlockTypeStudioPercent     = "studio_percent"
	legacyBlockTypeAdminPercent      = "admin_percent"
	legacyBlockTypeTeamPoolPercent   = "team_pool_percent"
	legacyBlockTypeTeamPoolRemainder = "team_pool_remainder"

	ValueTypePercent = "percent"
	ValueTypeFixed   = "fixed"

	AppliesToGross          = "gross"
	AppliesToCurrentBalance = "current_balance"

	TeamDistributionModePercent       = "percent"
	TeamDistributionModeFullRemainder = "full_remainder"
)

var (
	ErrAllocationBlockNotFound       = errors.New("allocation block not found")
	ErrAllocationInvalidType         = errors.New("allocation block type is invalid")
	ErrAllocationInvalidValueType    = errors.New("allocation block value type is invalid")
	ErrAllocationInvalidAppliesTo    = errors.New("allocation block applies_to is invalid")
	ErrAllocationInvalidValue        = errors.New("allocation block value must be greater than or equal to zero")
	ErrAllocationInvalidOrderPayload = errors.New("invalid allocation block order payload")
	ErrAllocationInvalidLevelIndex   = errors.New("allocation block level_index must be greater than or equal to 1")
	ErrAllocationInvalidTeamMode     = errors.New("team distribution mode is invalid")
	ErrAllocationRemainderConflict   = errors.New("only one full remainder team distribution block is allowed per project")
)

type Service struct {
	repo Repository
}

type UpsertBlockInput struct {
	Name                 string
	Type                 string
	LevelIndex           int32
	Enabled              bool
	ValueType            string
	Value                float64
	AppliesTo            string
	TeamDistributionMode string
}

type ReorderItemInput struct {
	ID         uuid.UUID
	LevelIndex int32
	OrderIndex int32
}

type PreviewInput struct {
	GrossAmount *float64
}

type PreviewStep struct {
	BlockID              uuid.UUID `json:"block_id"`
	Name                 string    `json:"name"`
	Type                 string    `json:"type"`
	LevelIndex           int32     `json:"level_index"`
	OrderIndex           int32     `json:"order_index"`
	ValueType            string    `json:"value_type"`
	Value                float64   `json:"value"`
	AppliesTo            string    `json:"applies_to"`
	InputAmount          float64   `json:"input_amount"`
	AllocatedAmount      float64   `json:"allocated_amount"`
	RemainingAmount      float64   `json:"remaining_amount"`
	IsTeamPool           bool      `json:"is_team_pool"`
	TeamDistributionMode string    `json:"team_distribution_mode,omitempty"`
}

type PreviewLevel struct {
	LevelIndex      int32         `json:"level_index"`
	IncomingAmount  float64       `json:"incoming_amount"`
	TotalAllocated  float64       `json:"total_allocated"`
	RemainingAmount float64       `json:"remaining_amount"`
	Steps           []PreviewStep `json:"steps"`
}

type PreviewResponse struct {
	ProjectID                   uuid.UUID      `json:"project_id"`
	ProjectCurrencyCode         string         `json:"project_currency_code"`
	GrossAmount                 float64        `json:"gross_amount"`
	GrossAmountSource           string         `json:"gross_amount_source"`
	TotalDeductedBeforeTeamPool float64        `json:"total_deducted_before_team_pool"`
	FinalRemainingAmount        float64        `json:"final_remaining_amount"`
	FinalTeamPoolAmount         float64        `json:"final_team_pool_amount"`
	Levels                      []PreviewLevel `json:"levels"`
	Steps                       []PreviewStep  `json:"steps"`
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, projectID uuid.UUID) ([]sqlc.AllocationBlock, error) {
	return s.repo.ListAllocationBlocksByProject(ctx, projectID)
}

func (s *Service) Create(ctx context.Context, projectID uuid.UUID, input UpsertBlockInput) (sqlc.AllocationBlock, error) {
	normalized, err := s.normalizeInput(input)
	if err != nil {
		return sqlc.AllocationBlock{}, err
	}

	var created sqlc.AllocationBlock
	err = s.repo.WithTx(ctx, func(repo Repository) error {
		blocks, err := repo.ListAllocationBlocksByProject(ctx, projectID)
		if err != nil {
			return err
		}
		if hasFullRemainderConflict(blocks, normalized.Type, normalized.TeamDistributionMode, nil) {
			return ErrAllocationRemainderConflict
		}

		item, err := repo.CreateAllocationBlock(ctx, sqlc.CreateAllocationBlockParams{
			ProjectID:  projectID,
			Name:       normalized.Name,
			Type:       normalized.Type,
			LevelIndex: normalized.LevelIndex,
			OrderIndex: nextOrderIndexForLevel(blocks, normalized.LevelIndex),
			Enabled:    normalized.Enabled,
			ValueType:  normalized.ValueType,
			Value:      normalized.Value,
			AppliesTo:  normalized.AppliesTo,
		})
		if err != nil {
			return err
		}
		created = item
		return nil
	})
	if err != nil {
		return sqlc.AllocationBlock{}, err
	}
	return created, nil
}

func (s *Service) Update(ctx context.Context, projectID, blockID uuid.UUID, input UpsertBlockInput) (sqlc.AllocationBlock, error) {
	normalized, err := s.normalizeInput(input)
	if err != nil {
		return sqlc.AllocationBlock{}, err
	}

	var updated sqlc.AllocationBlock
	err = s.repo.WithTx(ctx, func(repo Repository) error {
		current, err := repo.GetAllocationBlockByProject(ctx, sqlc.GetAllocationBlockByProjectParams{
			ProjectID: projectID,
			ID:        blockID,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrAllocationBlockNotFound
			}
			return err
		}

		blocks, err := repo.ListAllocationBlocksByProject(ctx, projectID)
		if err != nil {
			return err
		}
		if hasFullRemainderConflict(blocks, normalized.Type, normalized.TeamDistributionMode, &current.ID) {
			return ErrAllocationRemainderConflict
		}

		targetLevel := current.LevelIndex
		if normalized.LevelIndex > 0 {
			targetLevel = normalized.LevelIndex
		}
		targetOrder := current.OrderIndex
		if current.LevelIndex != targetLevel {
			targetOrder = nextOrderIndexForLevel(blocks, targetLevel)
		}

		item, err := repo.UpdateAllocationBlock(ctx, sqlc.UpdateAllocationBlockParams{
			ProjectID:  projectID,
			ID:         blockID,
			Name:       normalized.Name,
			Type:       normalized.Type,
			LevelIndex: targetLevel,
			Enabled:    normalized.Enabled,
			ValueType:  normalized.ValueType,
			Value:      normalized.Value,
			AppliesTo:  normalized.AppliesTo,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrAllocationBlockNotFound
			}
			return err
		}
		if current.LevelIndex != targetLevel {
			if err := repo.UpdateAllocationBlockPlacement(ctx, sqlc.UpdateAllocationBlockPlacementParams{
				ProjectID:  projectID,
				ID:         blockID,
				LevelIndex: targetLevel,
				OrderIndex: targetOrder,
			}); err != nil {
				return err
			}
			item.LevelIndex = targetLevel
			item.OrderIndex = targetOrder
		}
		if err := compactPlacement(ctx, repo, projectID); err != nil {
			return err
		}
		updated = item
		return nil
	})
	if err != nil {
		return sqlc.AllocationBlock{}, err
	}
	return updated, nil
}

func (s *Service) Delete(ctx context.Context, projectID, blockID uuid.UUID) error {
	return s.repo.WithTx(ctx, func(repo Repository) error {
		rows, err := repo.DeleteAllocationBlock(ctx, sqlc.DeleteAllocationBlockParams{
			ProjectID: projectID,
			ID:        blockID,
		})
		if err != nil {
			return err
		}
		if rows == 0 {
			return ErrAllocationBlockNotFound
		}
		return compactPlacement(ctx, repo, projectID)
	})
}

func (s *Service) Reorder(ctx context.Context, projectID uuid.UUID, items []ReorderItemInput, legacyOrderedIDs []uuid.UUID) error {
	return s.repo.WithTx(ctx, func(repo Repository) error {
		blocks, err := repo.ListAllocationBlocksByProject(ctx, projectID)
		if err != nil {
			return err
		}

		if len(items) == 0 {
			items = legacyReorderItems(blocks, legacyOrderedIDs)
		}
		if !isValidPlacementPayload(blocks, items) {
			return ErrAllocationInvalidOrderPayload
		}

		for _, item := range items {
			if err := repo.UpdateAllocationBlockPlacement(ctx, sqlc.UpdateAllocationBlockPlacementParams{
				ProjectID:  projectID,
				ID:         item.ID,
				LevelIndex: item.LevelIndex,
				OrderIndex: item.OrderIndex,
			}); err != nil {
				return err
			}
		}
		return compactPlacement(ctx, repo, projectID)
	})
}

func (s *Service) Preview(ctx context.Context, projectID uuid.UUID, input PreviewInput) (PreviewResponse, error) {
	project, err := s.repo.GetProjectByID(ctx, projectID)
	if err != nil {
		return PreviewResponse{}, err
	}
	summary, err := s.repo.GetFinanceSummaryByProject(ctx, projectID)
	if err != nil {
		return PreviewResponse{}, err
	}
	blocks, err := s.repo.ListAllocationBlocksByProject(ctx, projectID)
	if err != nil {
		return PreviewResponse{}, err
	}

	grossAmount := summary.IncomeTotal
	grossSource := "project_income_total"
	if input.GrossAmount != nil {
		grossAmount = *input.GrossAmount
		grossSource = "manual_override"
	}

	return ComputePreview(projectID, project.CurrencyCode, grossAmount, grossSource, blocks), nil
}

func ComputePreview(projectID uuid.UUID, projectCurrencyCode string, grossAmount float64, grossSource string, blocks []sqlc.AllocationBlock) PreviewResponse {
	activeBlocks := make([]sqlc.AllocationBlock, 0, len(blocks))
	for _, block := range blocks {
		if block.Enabled {
			activeBlocks = append(activeBlocks, normalizeStoredBlock(block))
		}
	}
	sortBlocks(activeBlocks)

	remaining := clampAmount(grossAmount)
	teamPool := 0.0
	totalDeducted := 0.0
	steps := make([]PreviewStep, 0, len(activeBlocks))
	levels := make([]PreviewLevel, 0)

	for _, levelIndex := range collectLevelIndexes(activeBlocks) {
		levelBlocks := blocksForLevel(activeBlocks, levelIndex)
		levelIncoming := clampAmount(remaining)
		levelRemaining := levelIncoming
		levelSteps := make([]PreviewStep, 0, len(levelBlocks))
		levelAllocated := 0.0

		for _, block := range levelBlocks {
			baseAmount := levelIncoming
			if block.Type != BlockTypeTeamDistribution && block.LevelIndex <= 1 {
				baseAmount = clampAmount(grossAmount)
			}

			allocated := computeBlockAllocation(block, baseAmount, levelRemaining, levelIncoming)
			levelRemaining = clampAmount(levelRemaining - allocated)
			isTeamPool := block.Type == BlockTypeTeamDistribution
			if isTeamPool {
				teamPool += allocated
			} else {
				totalDeducted += allocated
			}
			levelAllocated += allocated

			step := PreviewStep{
				BlockID:              block.ID,
				Name:                 block.Name,
				Type:                 block.Type,
				LevelIndex:           block.LevelIndex,
				OrderIndex:           block.OrderIndex,
				ValueType:            block.ValueType,
				Value:                block.Value,
				AppliesTo:            block.AppliesTo,
				InputAmount:          clampAmount(baseAmount),
				AllocatedAmount:      allocated,
				RemainingAmount:      levelRemaining,
				IsTeamPool:           isTeamPool,
				TeamDistributionMode: teamDistributionMode(block),
			}
			steps = append(steps, step)
			levelSteps = append(levelSteps, step)
		}

		remaining = clampAmount(levelRemaining)
		levels = append(levels, PreviewLevel{
			LevelIndex:      levelIndex,
			IncomingAmount:  levelIncoming,
			TotalAllocated:  clampAmount(levelAllocated),
			RemainingAmount: remaining,
			Steps:           levelSteps,
		})
	}

	return PreviewResponse{
		ProjectID:                   projectID,
		ProjectCurrencyCode:         projectCurrencyCode,
		GrossAmount:                 clampAmount(grossAmount),
		GrossAmountSource:           grossSource,
		TotalDeductedBeforeTeamPool: clampAmount(totalDeducted),
		FinalRemainingAmount:        clampAmount(remaining),
		FinalTeamPoolAmount:         clampAmount(teamPool),
		Levels:                      levels,
		Steps:                       steps,
	}
}

func (s *Service) normalizeInput(input UpsertBlockInput) (UpsertBlockInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Type = strings.TrimSpace(input.Type)
	input.ValueType = strings.TrimSpace(input.ValueType)
	input.AppliesTo = strings.TrimSpace(input.AppliesTo)
	input.TeamDistributionMode = strings.TrimSpace(input.TeamDistributionMode)

	if input.LevelIndex <= 0 {
		input.LevelIndex = 1
	}
	if input.LevelIndex < 1 {
		return UpsertBlockInput{}, ErrAllocationInvalidLevelIndex
	}
	if input.Value < 0 {
		return UpsertBlockInput{}, ErrAllocationInvalidValue
	}

	input.Type = canonicalBlockType(input.Type)
	if !isValidBlockType(input.Type) {
		return UpsertBlockInput{}, ErrAllocationInvalidType
	}

	if input.Type == BlockTypeTeamDistribution {
		return normalizeTeamDistributionInput(input)
	}

	if !isValidValueType(input.ValueType) {
		return UpsertBlockInput{}, ErrAllocationInvalidValueType
	}
	if input.LevelIndex <= 1 {
		input.AppliesTo = AppliesToGross
	} else {
		input.AppliesTo = AppliesToCurrentBalance
	}
	return input, nil
}

func normalizeTeamDistributionInput(input UpsertBlockInput) (UpsertBlockInput, error) {
	mode := input.TeamDistributionMode
	if mode == "" {
		mode = inferTeamDistributionModeFromInput(input)
	}
	switch mode {
	case TeamDistributionModePercent:
		input.TeamDistributionMode = mode
		input.ValueType = ValueTypePercent
		input.AppliesTo = AppliesToCurrentBalance
		return input, nil
	case TeamDistributionModeFullRemainder:
		input.TeamDistributionMode = mode
		input.ValueType = ValueTypeFixed
		input.Value = 0
		input.AppliesTo = AppliesToCurrentBalance
		return input, nil
	default:
		return UpsertBlockInput{}, ErrAllocationInvalidTeamMode
	}
}

func compactPlacement(ctx context.Context, repo Repository, projectID uuid.UUID) error {
	blocks, err := repo.ListAllocationBlocksByProject(ctx, projectID)
	if err != nil {
		return err
	}
	sortBlocks(blocks)
	currentLevel := int32(0)
	levelOrder := int32(0)
	for _, block := range blocks {
		if block.LevelIndex != currentLevel {
			currentLevel = block.LevelIndex
			levelOrder = 1
		} else {
			levelOrder++
		}
		if block.OrderIndex == levelOrder {
			continue
		}
		if err := repo.UpdateAllocationBlockPlacement(ctx, sqlc.UpdateAllocationBlockPlacementParams{
			ProjectID:  projectID,
			ID:         block.ID,
			LevelIndex: currentLevel,
			OrderIndex: levelOrder,
		}); err != nil {
			return err
		}
	}
	return nil
}

func nextOrderIndexForLevel(blocks []sqlc.AllocationBlock, levelIndex int32) int32 {
	var max int32
	for _, block := range blocks {
		if block.LevelIndex != levelIndex {
			continue
		}
		if block.OrderIndex > max {
			max = block.OrderIndex
		}
	}
	return max + 1
}

func hasFullRemainderConflict(blocks []sqlc.AllocationBlock, nextType, nextMode string, excludeID *uuid.UUID) bool {
	if nextType != BlockTypeTeamDistribution || nextMode != TeamDistributionModeFullRemainder {
		return false
	}
	for _, block := range blocks {
		block = normalizeStoredBlock(block)
		if block.Type != BlockTypeTeamDistribution || teamDistributionMode(block) != TeamDistributionModeFullRemainder {
			continue
		}
		if excludeID != nil && block.ID == *excludeID {
			continue
		}
		return true
	}
	return false
}

func isValidPlacementPayload(blocks []sqlc.AllocationBlock, items []ReorderItemInput) bool {
	if len(blocks) != len(items) {
		return false
	}
	expected := make(map[uuid.UUID]struct{}, len(blocks))
	for _, block := range blocks {
		expected[block.ID] = struct{}{}
	}
	seen := make(map[uuid.UUID]struct{}, len(items))
	ordersPerLevel := make(map[int32]map[int32]struct{})
	for _, item := range items {
		if item.LevelIndex < 1 || item.OrderIndex < 1 {
			return false
		}
		if _, ok := expected[item.ID]; !ok {
			return false
		}
		if _, duplicate := seen[item.ID]; duplicate {
			return false
		}
		seen[item.ID] = struct{}{}
		if ordersPerLevel[item.LevelIndex] == nil {
			ordersPerLevel[item.LevelIndex] = map[int32]struct{}{}
		}
		if _, duplicate := ordersPerLevel[item.LevelIndex][item.OrderIndex]; duplicate {
			return false
		}
		ordersPerLevel[item.LevelIndex][item.OrderIndex] = struct{}{}
	}
	return true
}

func legacyReorderItems(blocks []sqlc.AllocationBlock, orderedIDs []uuid.UUID) []ReorderItemInput {
	if len(orderedIDs) == 0 {
		return nil
	}
	levelByID := make(map[uuid.UUID]int32, len(blocks))
	for _, block := range blocks {
		levelByID[block.ID] = block.LevelIndex
	}
	orders := make(map[int32]int32)
	items := make([]ReorderItemInput, 0, len(orderedIDs))
	for _, id := range orderedIDs {
		level := levelByID[id]
		orders[level]++
		items = append(items, ReorderItemInput{
			ID:         id,
			LevelIndex: level,
			OrderIndex: orders[level],
		})
	}
	return items
}

func canonicalBlockType(value string) string {
	switch value {
	case legacyBlockTypeTaxPercent:
		return BlockTypeTax
	case legacyBlockTypeFixedCost:
		return BlockTypeActualExpense
	case legacyBlockTypeStudioPercent:
		return BlockTypeStudioDistribution
	case legacyBlockTypeAdminPercent:
		return BlockTypeAdminDistribution
	case legacyBlockTypeTeamPoolPercent, legacyBlockTypeTeamPoolRemainder:
		return BlockTypeTeamDistribution
	default:
		return value
	}
}

func normalizeStoredBlock(block sqlc.AllocationBlock) sqlc.AllocationBlock {
	block.Type = canonicalBlockType(block.Type)
	if block.LevelIndex <= 0 {
		block.LevelIndex = 1
	}
	if block.Type == BlockTypeTeamDistribution {
		switch teamDistributionMode(block) {
		case TeamDistributionModeFullRemainder:
			block.ValueType = ValueTypeFixed
			block.Value = 0
			block.AppliesTo = AppliesToCurrentBalance
		default:
			block.ValueType = ValueTypePercent
			block.AppliesTo = AppliesToCurrentBalance
		}
		return block
	}
	if block.LevelIndex <= 1 {
		block.AppliesTo = AppliesToGross
	} else {
		block.AppliesTo = AppliesToCurrentBalance
	}
	return block
}

func teamDistributionMode(block sqlc.AllocationBlock) string {
	if block.Type != BlockTypeTeamDistribution && block.Type != legacyBlockTypeTeamPoolPercent && block.Type != legacyBlockTypeTeamPoolRemainder {
		return ""
	}
	if block.Type == legacyBlockTypeTeamPoolRemainder {
		return TeamDistributionModeFullRemainder
	}
	if block.ValueType == ValueTypeFixed && block.Value == 0 {
		return TeamDistributionModeFullRemainder
	}
	return TeamDistributionModePercent
}

func inferTeamDistributionModeFromInput(input UpsertBlockInput) string {
	switch input.Type {
	case legacyBlockTypeTeamPoolRemainder:
		return TeamDistributionModeFullRemainder
	case legacyBlockTypeTeamPoolPercent:
		return TeamDistributionModePercent
	}
	if input.ValueType == ValueTypeFixed && input.Value == 0 {
		return TeamDistributionModeFullRemainder
	}
	if input.ValueType == ValueTypePercent {
		return TeamDistributionModePercent
	}
	return input.TeamDistributionMode
}

func isValidBlockType(value string) bool {
	switch value {
	case BlockTypeTax, BlockTypeActualExpense, BlockTypeCustomFee, BlockTypeStudioDistribution, BlockTypeAdminDistribution, BlockTypeTeamDistribution:
		return true
	default:
		return false
	}
}

func isValidValueType(value string) bool {
	return value == ValueTypePercent || value == ValueTypeFixed
}

func isValidAppliesTo(value string) bool {
	return value == AppliesToGross || value == AppliesToCurrentBalance
}

func computeBlockAllocation(block sqlc.AllocationBlock, baseAmount, remainingInLevel, levelIncoming float64) float64 {
	if remainingInLevel <= 0 {
		return 0
	}
	if block.Type == BlockTypeTeamDistribution && teamDistributionMode(block) == TeamDistributionModeFullRemainder {
		return clampAmount(remainingInLevel)
	}

	amount := 0.0
	switch block.ValueType {
	case ValueTypePercent:
		amount = clampAmount(baseAmount * block.Value / 100)
	case ValueTypeFixed:
		amount = clampAmount(block.Value)
	default:
		amount = 0
	}
	if amount > levelIncoming {
		amount = levelIncoming
	}
	if amount > remainingInLevel {
		amount = remainingInLevel
	}
	return clampAmount(amount)
}

func sortBlocks(blocks []sqlc.AllocationBlock) {
	sort.SliceStable(blocks, func(i, j int) bool {
		if blocks[i].LevelIndex == blocks[j].LevelIndex {
			if blocks[i].OrderIndex == blocks[j].OrderIndex {
				return blocks[i].CreatedAt.Before(blocks[j].CreatedAt)
			}
			return blocks[i].OrderIndex < blocks[j].OrderIndex
		}
		return blocks[i].LevelIndex < blocks[j].LevelIndex
	})
}

func collectLevelIndexes(blocks []sqlc.AllocationBlock) []int32 {
	levels := make([]int32, 0)
	seen := map[int32]struct{}{}
	for _, block := range blocks {
		if _, ok := seen[block.LevelIndex]; ok {
			continue
		}
		seen[block.LevelIndex] = struct{}{}
		levels = append(levels, block.LevelIndex)
	}
	sort.Slice(levels, func(i, j int) bool { return levels[i] < levels[j] })
	return levels
}

func blocksForLevel(blocks []sqlc.AllocationBlock, levelIndex int32) []sqlc.AllocationBlock {
	items := make([]sqlc.AllocationBlock, 0)
	for _, block := range blocks {
		if block.LevelIndex == levelIndex {
			items = append(items, block)
		}
	}
	return items
}

func clampAmount(value float64) float64 {
	if value < 0 {
		return 0
	}
	return value
}
