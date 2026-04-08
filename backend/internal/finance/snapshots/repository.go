package snapshots

import (
	"context"

	"isbudget/backend/internal/db/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	WithTx(ctx context.Context, fn func(repo Repository) error) error
	GetUserByID(ctx context.Context, id uuid.UUID) (sqlc.User, error)
	GetProjectMemberByUser(ctx context.Context, arg sqlc.GetProjectMemberByUserParams) (sqlc.ProjectMember, error)
	CreatePayoutSnapshot(ctx context.Context, arg sqlc.CreatePayoutSnapshotParams) (sqlc.PayoutSnapshot, error)
	CreatePayoutSnapshotMember(ctx context.Context, arg sqlc.CreatePayoutSnapshotMemberParams) (sqlc.PayoutSnapshotMember, error)
	CreatePayoutSnapshotBlock(ctx context.Context, arg sqlc.CreatePayoutSnapshotBlockParams) (sqlc.PayoutSnapshotBlock, error)
	UpdatePayoutSnapshotSettlement(ctx context.Context, arg sqlc.UpdatePayoutSnapshotSettlementParams) (sqlc.PayoutSnapshot, error)
	ListPayoutSnapshotsByProject(ctx context.Context, projectID uuid.UUID) ([]sqlc.PayoutSnapshot, error)
	GetPayoutSnapshotByID(ctx context.Context, id uuid.UUID) (sqlc.PayoutSnapshot, error)
	ListPayoutSnapshotMembersBySnapshot(ctx context.Context, snapshotID uuid.UUID) ([]sqlc.PayoutSnapshotMember, error)
	ListPayoutSnapshotBlocksBySnapshot(ctx context.Context, snapshotID uuid.UUID) ([]sqlc.PayoutSnapshotBlock, error)
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

func (r *PGRepository) GetUserByID(ctx context.Context, id uuid.UUID) (sqlc.User, error) {
	row, err := r.queries().GetUserByID(ctx, id)
	if err != nil {
		return sqlc.User{}, err
	}
	return sqlc.User{
		ID:                  row.ID,
		Email:               row.Email,
		PasswordHash:        row.PasswordHash,
		FullName:            row.FullName,
		CreatedAt:           row.CreatedAt,
		DefaultCurrencyCode: row.DefaultCurrencyCode,
		FirstName:           row.FirstName,
		LastName:            row.LastName,
		Nickname:            row.Nickname,
		AvatarPath:          row.AvatarPath,
	}, nil
}

func (r *PGRepository) GetProjectMemberByUser(ctx context.Context, arg sqlc.GetProjectMemberByUserParams) (sqlc.ProjectMember, error) {
	return r.queries().GetProjectMemberByUser(ctx, arg)
}

func (r *PGRepository) CreatePayoutSnapshot(ctx context.Context, arg sqlc.CreatePayoutSnapshotParams) (sqlc.PayoutSnapshot, error) {
	row, err := r.queries().CreatePayoutSnapshot(ctx, arg)
	if err != nil {
		return sqlc.PayoutSnapshot{}, err
	}
	return sqlc.PayoutSnapshot(row), nil
}

func (r *PGRepository) CreatePayoutSnapshotMember(ctx context.Context, arg sqlc.CreatePayoutSnapshotMemberParams) (sqlc.PayoutSnapshotMember, error) {
	row, err := r.queries().CreatePayoutSnapshotMember(ctx, arg)
	if err != nil {
		return sqlc.PayoutSnapshotMember{}, err
	}
	return sqlc.PayoutSnapshotMember(row), nil
}

func (r *PGRepository) CreatePayoutSnapshotBlock(ctx context.Context, arg sqlc.CreatePayoutSnapshotBlockParams) (sqlc.PayoutSnapshotBlock, error) {
	row, err := r.queries().CreatePayoutSnapshotBlock(ctx, arg)
	if err != nil {
		return sqlc.PayoutSnapshotBlock{}, err
	}
	return mapPayoutSnapshotBlockCreateRow(row), nil
}

func (r *PGRepository) UpdatePayoutSnapshotSettlement(ctx context.Context, arg sqlc.UpdatePayoutSnapshotSettlementParams) (sqlc.PayoutSnapshot, error) {
	row, err := r.queries().UpdatePayoutSnapshotSettlement(ctx, arg)
	if err != nil {
		return sqlc.PayoutSnapshot{}, err
	}
	return sqlc.PayoutSnapshot(row), nil
}

func (r *PGRepository) ListPayoutSnapshotsByProject(ctx context.Context, projectID uuid.UUID) ([]sqlc.PayoutSnapshot, error) {
	rows, err := r.queries().ListPayoutSnapshotsByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	items := make([]sqlc.PayoutSnapshot, 0, len(rows))
	for _, row := range rows {
		items = append(items, sqlc.PayoutSnapshot(row))
	}
	return items, nil
}

func (r *PGRepository) GetPayoutSnapshotByID(ctx context.Context, id uuid.UUID) (sqlc.PayoutSnapshot, error) {
	row, err := r.queries().GetPayoutSnapshotByID(ctx, id)
	if err != nil {
		return sqlc.PayoutSnapshot{}, err
	}
	return sqlc.PayoutSnapshot(row), nil
}

func (r *PGRepository) ListPayoutSnapshotMembersBySnapshot(ctx context.Context, snapshotID uuid.UUID) ([]sqlc.PayoutSnapshotMember, error) {
	rows, err := r.queries().ListPayoutSnapshotMembersBySnapshot(ctx, snapshotID)
	if err != nil {
		return nil, err
	}
	items := make([]sqlc.PayoutSnapshotMember, 0, len(rows))
	for _, row := range rows {
		items = append(items, sqlc.PayoutSnapshotMember(row))
	}
	return items, nil
}

func (r *PGRepository) ListPayoutSnapshotBlocksBySnapshot(ctx context.Context, snapshotID uuid.UUID) ([]sqlc.PayoutSnapshotBlock, error) {
	rows, err := r.queries().ListPayoutSnapshotBlocksBySnapshot(ctx, snapshotID)
	if err != nil {
		return nil, err
	}
	items := make([]sqlc.PayoutSnapshotBlock, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapPayoutSnapshotBlockRow(row))
	}
	return items, nil
}

func mapPayoutSnapshotBlockCreateRow(row sqlc.CreatePayoutSnapshotBlockRow) sqlc.PayoutSnapshotBlock {
	return sqlc.PayoutSnapshotBlock{
		ID:              row.ID,
		SnapshotID:      row.SnapshotID,
		SourceBlockID:   row.SourceBlockID,
		Name:            row.Name,
		Type:            row.Type,
		LevelIndex:      row.LevelIndex,
		OrderIndex:      row.OrderIndex,
		ValueType:       row.ValueType,
		Value:           row.Value,
		AppliesTo:       row.AppliesTo,
		InputAmount:     row.InputAmount,
		AllocatedAmount: row.AllocatedAmount,
		RemainingAmount: row.RemainingAmount,
		IsTeamPool:      row.IsTeamPool,
	}
}

func mapPayoutSnapshotBlockRow(row sqlc.ListPayoutSnapshotBlocksBySnapshotRow) sqlc.PayoutSnapshotBlock {
	return sqlc.PayoutSnapshotBlock{
		ID:              row.ID,
		SnapshotID:      row.SnapshotID,
		SourceBlockID:   row.SourceBlockID,
		Name:            row.Name,
		Type:            row.Type,
		LevelIndex:      row.LevelIndex,
		OrderIndex:      row.OrderIndex,
		ValueType:       row.ValueType,
		Value:           row.Value,
		AppliesTo:       row.AppliesTo,
		InputAmount:     row.InputAmount,
		AllocatedAmount: row.AllocatedAmount,
		RemainingAmount: row.RemainingAmount,
		IsTeamPool:      row.IsTeamPool,
	}
}
