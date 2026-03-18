package planned_expenses

import (
	"context"
	"strings"

	"isbudget/backend/internal/db/sqlc"

	"github.com/google/uuid"
)

type Service struct {
	q *sqlc.Queries
}

func NewService(q *sqlc.Queries) *Service {
	return &Service{q: q}
}

func (s *Service) List(ctx context.Context, projectID uuid.UUID) ([]sqlc.PlannedExpense, error) {
	rows, err := s.q.ListPlannedExpensesByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	items := make([]sqlc.PlannedExpense, 0, len(rows))
	for _, row := range rows {
		items = append(items, sqlc.PlannedExpense(row))
	}
	return items, nil
}

func (s *Service) Create(ctx context.Context, req sqlc.CreatePlannedExpenseParams) (sqlc.PlannedExpense, error) {
	req.Category = strings.TrimSpace(req.Category)
	req.Note = strings.TrimSpace(req.Note)
	req.Title = plannedExpenseTitle(req.Category)
	row, err := s.q.CreatePlannedExpense(ctx, req)
	if err != nil {
		return sqlc.PlannedExpense{}, err
	}
	return sqlc.PlannedExpense(row), nil
}

func (s *Service) Update(ctx context.Context, req sqlc.UpdatePlannedExpenseParams) (sqlc.PlannedExpense, error) {
	req.Category = strings.TrimSpace(req.Category)
	req.Note = strings.TrimSpace(req.Note)
	req.Title = plannedExpenseTitle(req.Category)
	row, err := s.q.UpdatePlannedExpense(ctx, req)
	if err != nil {
		return sqlc.PlannedExpense{}, err
	}
	return sqlc.PlannedExpense(row), nil
}

func (s *Service) Delete(ctx context.Context, req sqlc.DeletePlannedExpenseParams) error {
	return s.q.DeletePlannedExpense(ctx, req)
}

func plannedExpenseTitle(category string) string {
	if category == "" {
		return "Плановый расход"
	}
	return "Категория: " + category
}
