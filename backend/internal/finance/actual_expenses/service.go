package actual_expenses

import (
	"context"
	"strings"
	"time"

	"isbudget/backend/internal/db/sqlc"

	"github.com/google/uuid"
)

type Service struct {
	q *sqlc.Queries
}

func NewService(q *sqlc.Queries) *Service {
	return &Service{q: q}
}

func (s *Service) ListRange(ctx context.Context, projectID uuid.UUID, from, to time.Time) ([]sqlc.ActualExpense, error) {
	rows, err := s.q.ListActualExpensesByProjectRange(ctx, sqlc.ListActualExpensesByProjectRangeParams{
		ProjectID: projectID,
		FromDate:  from,
		ToDate:    to,
	})
	if err != nil {
		return nil, err
	}
	items := make([]sqlc.ActualExpense, 0, len(rows))
	for _, row := range rows {
		items = append(items, sqlc.ActualExpense(row))
	}
	return items, nil
}

func (s *Service) ListAll(ctx context.Context, projectID uuid.UUID) ([]sqlc.ActualExpense, error) {
	rows, err := s.q.ListActualExpensesByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	items := make([]sqlc.ActualExpense, 0, len(rows))
	for _, row := range rows {
		items = append(items, sqlc.ActualExpense(row))
	}
	return items, nil
}

func (s *Service) Create(ctx context.Context, req sqlc.CreateActualExpenseParams) (sqlc.ActualExpense, error) {
	req.Title = strings.TrimSpace(req.Title)
	req.Category = strings.TrimSpace(req.Category)
	req.Note = strings.TrimSpace(req.Note)
	row, err := s.q.CreateActualExpense(ctx, req)
	if err != nil {
		return sqlc.ActualExpense{}, err
	}
	return sqlc.ActualExpense(row), nil
}

func (s *Service) Update(ctx context.Context, req sqlc.UpdateActualExpenseParams) (sqlc.ActualExpense, error) {
	req.Title = strings.TrimSpace(req.Title)
	req.Category = strings.TrimSpace(req.Category)
	req.Note = strings.TrimSpace(req.Note)
	row, err := s.q.UpdateActualExpense(ctx, req)
	if err != nil {
		return sqlc.ActualExpense{}, err
	}
	return sqlc.ActualExpense(row), nil
}

func (s *Service) Delete(ctx context.Context, req sqlc.DeleteActualExpenseParams) error {
	return s.q.DeleteActualExpense(ctx, req)
}
