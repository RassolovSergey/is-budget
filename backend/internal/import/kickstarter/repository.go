package kickstarter

import (
	"context"
	"time"

	"isbudget/backend/internal/db/sqlc"

	"github.com/google/uuid"
)

type Repository interface {
	GetProjectMemberByUser(ctx context.Context, arg sqlc.GetProjectMemberByUserParams) (sqlc.ProjectMember, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (sqlc.GetUserByIDRow, error)
	CreateImportRun(ctx context.Context, arg sqlc.CreateImportRunParams) (sqlc.ImportRun, error)
	UpdateImportRunResult(ctx context.Context, arg sqlc.UpdateImportRunResultParams) (sqlc.ImportRun, error)
	CreateImportRecord(ctx context.Context, arg sqlc.CreateImportRecordParams) (sqlc.ImportRecord, error)
	ListImportRunsByProject(ctx context.Context, projectID uuid.UUID) ([]sqlc.ImportRun, error)
	GetImportRunByID(ctx context.Context, id uuid.UUID) (sqlc.ImportRun, error)
	ListImportRecordsByRun(ctx context.Context, runID uuid.UUID) ([]sqlc.ImportRecord, error)
	ListIncomeDailyByProject(ctx context.Context, projectID uuid.UUID) ([]sqlc.ListIncomeDailyByProjectRow, error)
	UpsertIncomeDaily(ctx context.Context, arg sqlc.UpsertIncomeDailyParams) (sqlc.UpsertIncomeDailyRow, error)
}

type PGRepository struct {
	q *sqlc.Queries
}

func NewPGRepository(q *sqlc.Queries) *PGRepository {
	return &PGRepository{q: q}
}

func (r *PGRepository) GetProjectMemberByUser(ctx context.Context, arg sqlc.GetProjectMemberByUserParams) (sqlc.ProjectMember, error) {
	return r.q.GetProjectMemberByUser(ctx, arg)
}

func (r *PGRepository) GetUserByID(ctx context.Context, id uuid.UUID) (sqlc.GetUserByIDRow, error) {
	return r.q.GetUserByID(ctx, id)
}

func (r *PGRepository) CreateImportRun(ctx context.Context, arg sqlc.CreateImportRunParams) (sqlc.ImportRun, error) {
	row, err := r.q.CreateImportRun(ctx, arg)
	if err != nil {
		return sqlc.ImportRun{}, err
	}
	return sqlc.ImportRun(row), nil
}

func (r *PGRepository) UpdateImportRunResult(ctx context.Context, arg sqlc.UpdateImportRunResultParams) (sqlc.ImportRun, error) {
	row, err := r.q.UpdateImportRunResult(ctx, arg)
	if err != nil {
		return sqlc.ImportRun{}, err
	}
	return sqlc.ImportRun(row), nil
}

func (r *PGRepository) CreateImportRecord(ctx context.Context, arg sqlc.CreateImportRecordParams) (sqlc.ImportRecord, error) {
	row, err := r.q.CreateImportRecord(ctx, arg)
	if err != nil {
		return sqlc.ImportRecord{}, err
	}
	return sqlc.ImportRecord(row), nil
}

func (r *PGRepository) ListImportRunsByProject(ctx context.Context, projectID uuid.UUID) ([]sqlc.ImportRun, error) {
	rows, err := r.q.ListImportRunsByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	items := make([]sqlc.ImportRun, 0, len(rows))
	for _, row := range rows {
		items = append(items, sqlc.ImportRun(row))
	}
	return items, nil
}

func (r *PGRepository) GetImportRunByID(ctx context.Context, id uuid.UUID) (sqlc.ImportRun, error) {
	row, err := r.q.GetImportRunByID(ctx, id)
	if err != nil {
		return sqlc.ImportRun{}, err
	}
	return sqlc.ImportRun(row), nil
}

func (r *PGRepository) ListImportRecordsByRun(ctx context.Context, runID uuid.UUID) ([]sqlc.ImportRecord, error) {
	rows, err := r.q.ListImportRecordsByRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	items := make([]sqlc.ImportRecord, 0, len(rows))
	for _, row := range rows {
		items = append(items, sqlc.ImportRecord(row))
	}
	return items, nil
}

func (r *PGRepository) ListIncomeDailyByProject(ctx context.Context, projectID uuid.UUID) ([]sqlc.ListIncomeDailyByProjectRow, error) {
	return r.q.ListIncomeDailyByProject(ctx, projectID)
}

func (r *PGRepository) UpsertIncomeDaily(ctx context.Context, arg sqlc.UpsertIncomeDailyParams) (sqlc.UpsertIncomeDailyRow, error) {
	return r.q.UpsertIncomeDaily(ctx, arg)
}

func normalizeDate(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
}
