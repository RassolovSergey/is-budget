package taxes

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"isbudget/backend/internal/db/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInvalidTaxOrder         = errors.New("invalid tax order payload")
	ErrInvalidTaxLevel         = errors.New("tax level must be greater than or equal to 1")
	ErrInvalidTaxLevelSequence = errors.New("cannot skip tax levels: create missing intermediate levels first")
	ErrTaxLevelPercentOverflow = errors.New("sum of tax percents for a level cannot exceed 100")
)

type Service struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

func NewService(q *sqlc.Queries, pool *pgxpool.Pool) *Service {
	return &Service{q: q, pool: pool}
}

func (s *Service) List(ctx context.Context, projectID uuid.UUID) ([]sqlc.ProjectTaxRule, error) {
	rows, err := s.q.ListTaxRulesByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	rules := make([]sqlc.ProjectTaxRule, 0, len(rows))
	for _, row := range rows {
		rules = append(rules, mapTaxRuleRow(row))
	}
	return rules, nil
}

func (s *Service) Create(ctx context.Context, req sqlc.CreateTaxRuleParams) (sqlc.ProjectTaxRule, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.Level < 1 {
		return sqlc.ProjectTaxRule{}, ErrInvalidTaxLevel
	}
	if err := s.validateLevelSequence(ctx, req.ProjectID, req.Level, nil); err != nil {
		return sqlc.ProjectTaxRule{}, err
	}
	if err := s.validateLevelPercentBudget(ctx, req.ProjectID, req.Level, req.Percent, nil); err != nil {
		return sqlc.ProjectTaxRule{}, err
	}
	created, err := s.q.CreateTaxRule(ctx, req)
	if err != nil {
		return sqlc.ProjectTaxRule{}, err
	}
	if err := s.normalizeLevels(ctx, req.ProjectID); err != nil {
		return sqlc.ProjectTaxRule{}, err
	}
	return mapCreateTaxRuleRow(created), nil
}

func (s *Service) CreatePrepend(ctx context.Context, req sqlc.CreateTaxRuleParams) (sqlc.ProjectTaxRule, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Level = 1
	if req.Level < 1 {
		return sqlc.ProjectTaxRule{}, ErrInvalidTaxLevel
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return sqlc.ProjectTaxRule{}, err
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)
	rows, err := qtx.ListTaxRulesByProject(ctx, req.ProjectID)
	if err != nil {
		return sqlc.ProjectTaxRule{}, err
	}
	rules := make([]sqlc.ProjectTaxRule, 0, len(rows))
	for _, row := range rows {
		rules = append(rules, mapTaxRuleRow(row))
	}

	shifted := shiftTaxRulesForPrepend(rules)
	tempBase := int32(len(shifted) + 1000)
	for idx, rule := range shifted {
		if err := qtx.UpdateTaxRuleLevel(ctx, sqlc.UpdateTaxRuleLevelParams{
			ProjectID: req.ProjectID,
			ID:        rule.ID,
			Level:     tempBase + int32(idx),
		}); err != nil {
			return sqlc.ProjectTaxRule{}, err
		}
	}
	for _, rule := range shifted {
		if err := qtx.UpdateTaxRuleLevel(ctx, sqlc.UpdateTaxRuleLevelParams{
			ProjectID: req.ProjectID,
			ID:        rule.ID,
			Level:     rule.Level,
		}); err != nil {
			return sqlc.ProjectTaxRule{}, err
		}
	}

	created, err := qtx.CreateTaxRule(ctx, req)
	if err != nil {
		return sqlc.ProjectTaxRule{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return sqlc.ProjectTaxRule{}, err
	}
	return mapCreateTaxRuleRow(created), nil
}

func (s *Service) Update(ctx context.Context, req sqlc.UpdateTaxRuleParams) (sqlc.ProjectTaxRule, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.Level < 1 {
		return sqlc.ProjectTaxRule{}, ErrInvalidTaxLevel
	}
	if err := s.validateLevelSequence(ctx, req.ProjectID, req.Level, &req.ID); err != nil {
		return sqlc.ProjectTaxRule{}, err
	}
	if err := s.validateLevelPercentBudget(ctx, req.ProjectID, req.Level, req.Percent, &req.ID); err != nil {
		return sqlc.ProjectTaxRule{}, err
	}
	updated, err := s.q.UpdateTaxRule(ctx, req)
	if err != nil {
		return sqlc.ProjectTaxRule{}, err
	}
	if err := s.normalizeLevels(ctx, req.ProjectID); err != nil {
		return sqlc.ProjectTaxRule{}, err
	}
	return mapUpdateTaxRuleRow(updated), nil
}

func (s *Service) Delete(ctx context.Context, req sqlc.DeleteTaxRuleParams) error {
	if err := s.q.DeleteTaxRule(ctx, req); err != nil {
		return err
	}
	return s.normalizeLevels(ctx, req.ProjectID)
}

func (s *Service) Reorder(ctx context.Context, projectID uuid.UUID, orderedIDs []uuid.UUID) error {
	rows, err := s.q.ListTaxRulesByProject(ctx, projectID)
	if err != nil {
		return err
	}
	rules := make([]sqlc.ProjectTaxRule, 0, len(rows))
	for _, row := range rows {
		rules = append(rules, mapTaxRuleRow(row))
	}

	if len(orderedIDs) != len(rules) {
		return ErrInvalidTaxOrder
	}

	seen := make(map[uuid.UUID]struct{}, len(orderedIDs))
	exists := make(map[uuid.UUID]struct{}, len(rules))
	for _, rule := range rules {
		exists[rule.ID] = struct{}{}
	}
	for _, id := range orderedIDs {
		if _, ok := exists[id]; !ok {
			return ErrInvalidTaxOrder
		}
		if _, duplicate := seen[id]; duplicate {
			return ErrInvalidTaxOrder
		}
		seen[id] = struct{}{}
	}
	return s.persistLevels(ctx, projectID, orderedIDs)
}

func (s *Service) normalizeLevels(ctx context.Context, projectID uuid.UUID) error {
	rows, err := s.q.ListTaxRulesByProject(ctx, projectID)
	if err != nil {
		return err
	}
	rules := make([]sqlc.ProjectTaxRule, 0, len(rows))
	for _, row := range rows {
		rules = append(rules, mapTaxRuleRow(row))
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)
	levelMap := make(map[int32]int32, len(rules))
	nextLevel := int32(1)

	for _, rule := range rules {
		mapped, ok := levelMap[rule.Level]
		if !ok {
			mapped = nextLevel
			levelMap[rule.Level] = mapped
			nextLevel++
		}
		if mapped != rule.Level {
			if err := qtx.UpdateTaxRuleLevel(ctx, sqlc.UpdateTaxRuleLevelParams{
				ProjectID: projectID,
				ID:        rule.ID,
				Level:     mapped,
			}); err != nil {
				return err
			}
		}
	}

	return tx.Commit(ctx)
}

func (s *Service) validateLevelSequence(ctx context.Context, projectID uuid.UUID, level int32, excludeID *uuid.UUID) error {
	rows, err := s.q.ListTaxRulesByProject(ctx, projectID)
	if err != nil {
		return err
	}
	rules := make([]sqlc.ProjectTaxRule, 0, len(rows))
	for _, row := range rows {
		rules = append(rules, mapTaxRuleRow(row))
	}

	maxLevel := int32(0)
	for _, rule := range rules {
		if excludeID != nil && rule.ID == *excludeID {
			continue
		}
		if rule.Level > maxLevel {
			maxLevel = rule.Level
		}
	}

	if level > maxLevel+1 {
		return ErrInvalidTaxLevelSequence
	}
	return nil
}

func (s *Service) validateLevelPercentBudget(ctx context.Context, projectID uuid.UUID, level int32, percent float64, excludeID *uuid.UUID) error {
	rules, err := s.q.ListTaxRulesByProject(ctx, projectID)
	if err != nil {
		return err
	}

	sum := 0.0
	for _, rule := range rules {
		if excludeID != nil && rule.ID == *excludeID {
			continue
		}
		if rule.Level == level {
			sum += rule.Percent
		}
	}

	if sum+percent > 100.000001 {
		return fmt.Errorf("%w (уровень %d)", ErrTaxLevelPercentOverflow, level)
	}
	return nil
}

func (s *Service) persistLevels(ctx context.Context, projectID uuid.UUID, orderedIDs []uuid.UUID) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)
	tempBase := int32(len(orderedIDs) + 1000)
	for idx, id := range orderedIDs {
		if err := qtx.UpdateTaxRuleLevel(ctx, sqlc.UpdateTaxRuleLevelParams{
			ProjectID: projectID,
			ID:        id,
			Level:     tempBase + int32(idx),
		}); err != nil {
			return err
		}
	}

	for idx, id := range orderedIDs {
		if err := qtx.UpdateTaxRuleLevel(ctx, sqlc.UpdateTaxRuleLevelParams{
			ProjectID: projectID,
			ID:        id,
			Level:     int32(idx + 1),
		}); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func mapTaxRuleRow(row sqlc.ListTaxRulesByProjectRow) sqlc.ProjectTaxRule {
	return sqlc.ProjectTaxRule{
		ID:        row.ID,
		ProjectID: row.ProjectID,
		Name:      row.Name,
		Percent:   row.Percent,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
		Level:     row.Level,
	}
}

func mapCreateTaxRuleRow(row sqlc.CreateTaxRuleRow) sqlc.ProjectTaxRule {
	return sqlc.ProjectTaxRule{
		ID:        row.ID,
		ProjectID: row.ProjectID,
		Name:      row.Name,
		Percent:   row.Percent,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
		Level:     row.Level,
	}
}

func mapUpdateTaxRuleRow(row sqlc.UpdateTaxRuleRow) sqlc.ProjectTaxRule {
	return sqlc.ProjectTaxRule{
		ID:        row.ID,
		ProjectID: row.ProjectID,
		Name:      row.Name,
		Percent:   row.Percent,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
		Level:     row.Level,
	}
}

func shiftTaxRulesForPrepend(rules []sqlc.ProjectTaxRule) []sqlc.ProjectTaxRule {
	shifted := make([]sqlc.ProjectTaxRule, 0, len(rules))
	for _, rule := range rules {
		next := rule
		next.Level = rule.Level + 1
		shifted = append(shifted, next)
	}
	return shifted
}
