package income

import (
	"context"
	"sort"
	"time"

	"isbudget/backend/internal/db/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

func NewService(q *sqlc.Queries, pool *pgxpool.Pool) *Service {
	return &Service{q: q, pool: pool}
}

func (s *Service) ListRange(ctx context.Context, projectID uuid.UUID, from, to time.Time) ([]sqlc.ActualIncomeDaily, error) {
	rows, err := s.q.ListIncomeDailyByProjectRange(ctx, sqlc.ListIncomeDailyByProjectRangeParams{
		ProjectID: projectID,
		FromDate:  from,
		ToDate:    to,
	})
	if err != nil {
		return nil, err
	}
	items := make([]sqlc.ActualIncomeDaily, 0, len(rows))
	for _, row := range rows {
		items = append(items, sqlc.ActualIncomeDaily(row))
	}
	return items, nil
}

func (s *Service) ListAll(ctx context.Context, projectID uuid.UUID) ([]sqlc.ActualIncomeDaily, error) {
	rows, err := s.q.ListIncomeDailyByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	items := make([]sqlc.ActualIncomeDaily, 0, len(rows))
	for _, row := range rows {
		items = append(items, sqlc.ActualIncomeDaily(row))
	}
	return items, nil
}

func (s *Service) Upsert(ctx context.Context, projectID uuid.UUID, date time.Time, amount float64) (sqlc.ActualIncomeDaily, error) {
	row, err := s.q.UpsertIncomeDaily(ctx, sqlc.UpsertIncomeDailyParams{
		ProjectID:  projectID,
		IncomeDate: date,
		Amount:     amount,
	})
	if err != nil {
		return sqlc.ActualIncomeDaily{}, err
	}
	return sqlc.ActualIncomeDaily(row), nil
}

func (s *Service) Delete(ctx context.Context, projectID uuid.UUID, date time.Time) error {
	return s.q.DeleteIncomeDaily(ctx, sqlc.DeleteIncomeDailyParams{ProjectID: projectID, IncomeDate: date})
}

type CumulativePointInput struct {
	Date             time.Time
	CumulativeAmount *float64
}

func (s *Service) UpsertCumulative(ctx context.Context, projectID uuid.UUID, points []CumulativePointInput) ([]sqlc.ActualIncomeDaily, error) {
	normalizedPoints := normalizeCumulativePoints(points)
	if len(normalizedPoints) == 0 {
		return []sqlc.ActualIncomeDaily{}, nil
	}

	targetDates := make([]time.Time, 0, len(normalizedPoints))
	for _, point := range normalizedPoints {
		targetDates = append(targetDates, point.Date)
	}
	firstDate := targetDates[0]
	lastDate := targetDates[len(targetDates)-1]

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	qtx := s.q.WithTx(tx)

	// The cumulative UI submits an authoritative campaign-scoped series.
	// Old rows before the first submitted day are stale residues and must not
	// shift the day-1 baseline away from zero.
	if _, err := tx.Exec(ctx, `
DELETE FROM actual_income_daily
WHERE project_id = $1
  AND income_date < $2
`, projectID, firstDate); err != nil {
		return nil, err
	}

	deleteDates := make([]time.Time, 0, len(targetDates))
	for _, date := range targetDates {
		deleteDates = append(deleteDates, date)
	}
	if err := qtx.DeleteIncomeDailyByProjectDates(ctx, sqlc.DeleteIncomeDailyByProjectDatesParams{
		ProjectID: projectID,
		Column2:   deleteDates,
	}); err != nil {
		return nil, err
	}

	upserts := buildDailyDeltaUpserts(projectID, normalizedPoints)
	for _, upsert := range upserts {
		if _, err := qtx.UpsertIncomeDaily(ctx, upsert); err != nil {
			return nil, err
		}
	}

	updatedRows, err := qtx.ListIncomeDailyByProjectRange(ctx, sqlc.ListIncomeDailyByProjectRangeParams{
		ProjectID: projectID,
		FromDate:  firstDate,
		ToDate:    lastDate,
	})
	if err != nil {
		return nil, err
	}
	updated := make([]sqlc.ActualIncomeDaily, 0, len(updatedRows))
	for _, row := range updatedRows {
		updated = append(updated, sqlc.ActualIncomeDaily(row))
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return updated, nil
}

type normalizedCumulativePoint struct {
	Date             time.Time
	CumulativeAmount *float64
}

func normalizeCumulativePoints(points []CumulativePointInput) []normalizedCumulativePoint {
	if len(points) == 0 {
		return nil
	}
	byDate := make(map[string]normalizedCumulativePoint, len(points))
	for _, point := range points {
		date := time.Date(point.Date.Year(), point.Date.Month(), point.Date.Day(), 0, 0, 0, 0, time.UTC)
		key := date.Format("2006-01-02")
		byDate[key] = normalizedCumulativePoint{
			Date:             date,
			CumulativeAmount: point.CumulativeAmount,
		}
	}
	out := make([]normalizedCumulativePoint, 0, len(byDate))
	for _, point := range byDate {
		out = append(out, point)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date.Before(out[j].Date) })
	return out
}

func buildDailyDeltaUpserts(projectID uuid.UUID, points []normalizedCumulativePoint) []sqlc.UpsertIncomeDailyParams {
	if len(points) == 0 {
		return nil
	}
	running := 0.0
	out := make([]sqlc.UpsertIncomeDailyParams, 0, len(points))
	for _, point := range points {
		targetCumulative := running
		shouldPersistDay := false
		if point.CumulativeAmount != nil {
			targetCumulative = *point.CumulativeAmount
			shouldPersistDay = true
		}

		dayDelta := targetCumulative - running
		if shouldPersistDay {
			out = append(out, sqlc.UpsertIncomeDailyParams{
				ProjectID:  projectID,
				IncomeDate: point.Date,
				Amount:     dayDelta,
			})
		}
		running = targetCumulative
	}
	return out
}
