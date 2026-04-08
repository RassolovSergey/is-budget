package pipeline

import (
	"context"

	"isbudget/backend/internal/db/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	WithTx(ctx context.Context, fn func(repo Repository) error) error
	GetProjectByID(ctx context.Context, id uuid.UUID) (sqlc.Project, error)
	GetFinanceSummaryByProject(ctx context.Context, projectID uuid.UUID) (sqlc.FinanceSummary, error)
	ListAllocationBlocksByProject(ctx context.Context, projectID uuid.UUID) ([]sqlc.AllocationBlock, error)
	GetAllocationBlockByProject(ctx context.Context, arg sqlc.GetAllocationBlockByProjectParams) (sqlc.AllocationBlock, error)
	CreateAllocationBlock(ctx context.Context, arg sqlc.CreateAllocationBlockParams) (sqlc.AllocationBlock, error)
	UpdateAllocationBlock(ctx context.Context, arg sqlc.UpdateAllocationBlockParams) (sqlc.AllocationBlock, error)
	DeleteAllocationBlock(ctx context.Context, arg sqlc.DeleteAllocationBlockParams) (int64, error)
	UpdateAllocationBlockPlacement(ctx context.Context, arg sqlc.UpdateAllocationBlockPlacementParams) error
}

type PGRepository struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
	tx   pgx.Tx
}

func NewPGRepository(q *sqlc.Queries, pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{q: q, pool: pool}
}

func (r *PGRepository) queries() *sqlc.Queries {
	if r.tx != nil {
		return r.q.WithTx(r.tx)
	}
	return r.q
}

func (r *PGRepository) WithTx(ctx context.Context, fn func(repo Repository) error) error {
	if r.tx != nil {
		return fn(r)
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	repo := &PGRepository{q: r.q, pool: r.pool, tx: tx}
	if err := fn(repo); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) GetProjectByID(ctx context.Context, id uuid.UUID) (sqlc.Project, error) {
	row, err := r.queries().GetProjectByID(ctx, id)
	if err != nil {
		return sqlc.Project{}, err
	}
	return sqlc.Project{
		ID:                    row.ID,
		Name:                  row.Name,
		Description:           row.Description,
		CreatedBy:             row.CreatedBy,
		CreatedAt:             row.CreatedAt,
		UpdatedAt:             row.UpdatedAt,
		DurationDays:          row.DurationDays,
		Status:                row.Status,
		ActivatedAt:           row.ActivatedAt,
		CurrencyCode:          row.CurrencyCode,
		Category:              row.Category,
		SecondaryCurrencyCode: row.SecondaryCurrencyCode,
	}, nil
}

func (r *PGRepository) GetFinanceSummaryByProject(ctx context.Context, projectID uuid.UUID) (sqlc.FinanceSummary, error) {
	return r.queries().GetFinanceSummaryByProject(ctx, projectID)
}

func (r *PGRepository) ListAllocationBlocksByProject(ctx context.Context, projectID uuid.UUID) ([]sqlc.AllocationBlock, error) {
	rows, err := r.queries().ListAllocationBlocksByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	items := make([]sqlc.AllocationBlock, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapAllocationBlockRow(row))
	}
	return items, nil
}

func (r *PGRepository) GetAllocationBlockByProject(ctx context.Context, arg sqlc.GetAllocationBlockByProjectParams) (sqlc.AllocationBlock, error) {
	row, err := r.queries().GetAllocationBlockByProject(ctx, arg)
	if err != nil {
		return sqlc.AllocationBlock{}, err
	}
	return mapAllocationBlockGetRow(row), nil
}

func (r *PGRepository) CreateAllocationBlock(ctx context.Context, arg sqlc.CreateAllocationBlockParams) (sqlc.AllocationBlock, error) {
	row, err := r.queries().CreateAllocationBlock(ctx, arg)
	if err != nil {
		return sqlc.AllocationBlock{}, err
	}
	return mapAllocationBlockCreateRow(row), nil
}

func (r *PGRepository) UpdateAllocationBlock(ctx context.Context, arg sqlc.UpdateAllocationBlockParams) (sqlc.AllocationBlock, error) {
	row, err := r.queries().UpdateAllocationBlock(ctx, arg)
	if err != nil {
		return sqlc.AllocationBlock{}, err
	}
	return mapAllocationBlockUpdateRow(row), nil
}

func (r *PGRepository) DeleteAllocationBlock(ctx context.Context, arg sqlc.DeleteAllocationBlockParams) (int64, error) {
	return r.queries().DeleteAllocationBlock(ctx, arg)
}

func (r *PGRepository) UpdateAllocationBlockPlacement(ctx context.Context, arg sqlc.UpdateAllocationBlockPlacementParams) error {
	return r.queries().UpdateAllocationBlockPlacement(ctx, arg)
}

func mapAllocationBlockRow(row sqlc.ListAllocationBlocksByProjectRow) sqlc.AllocationBlock {
	return sqlc.AllocationBlock{
		ID:         row.ID,
		ProjectID:  row.ProjectID,
		Name:       row.Name,
		Type:       row.Type,
		LevelIndex: row.LevelIndex,
		OrderIndex: row.OrderIndex,
		Enabled:    row.Enabled,
		ValueType:  row.ValueType,
		Value:      row.Value,
		AppliesTo:  row.AppliesTo,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}
}

func mapAllocationBlockGetRow(row sqlc.GetAllocationBlockByProjectRow) sqlc.AllocationBlock {
	return sqlc.AllocationBlock{
		ID:         row.ID,
		ProjectID:  row.ProjectID,
		Name:       row.Name,
		Type:       row.Type,
		LevelIndex: row.LevelIndex,
		OrderIndex: row.OrderIndex,
		Enabled:    row.Enabled,
		ValueType:  row.ValueType,
		Value:      row.Value,
		AppliesTo:  row.AppliesTo,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}
}

func mapAllocationBlockCreateRow(row sqlc.CreateAllocationBlockRow) sqlc.AllocationBlock {
	return sqlc.AllocationBlock{
		ID:         row.ID,
		ProjectID:  row.ProjectID,
		Name:       row.Name,
		Type:       row.Type,
		LevelIndex: row.LevelIndex,
		OrderIndex: row.OrderIndex,
		Enabled:    row.Enabled,
		ValueType:  row.ValueType,
		Value:      row.Value,
		AppliesTo:  row.AppliesTo,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}
}

func mapAllocationBlockUpdateRow(row sqlc.UpdateAllocationBlockRow) sqlc.AllocationBlock {
	return sqlc.AllocationBlock{
		ID:         row.ID,
		ProjectID:  row.ProjectID,
		Name:       row.Name,
		Type:       row.Type,
		LevelIndex: row.LevelIndex,
		OrderIndex: row.OrderIndex,
		Enabled:    row.Enabled,
		ValueType:  row.ValueType,
		Value:      row.Value,
		AppliesTo:  row.AppliesTo,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}
}
