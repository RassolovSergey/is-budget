package lifecycle

import (
	"context"

	"isbudget/backend/internal/db/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	GetProjectMemberByUser(ctx context.Context, arg sqlc.GetProjectMemberByUserParams) (sqlc.ProjectMember, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (sqlc.User, error)
	GetCampaignConfigByProject(ctx context.Context, projectID uuid.UUID) (sqlc.CampaignConfig, error)
	UpsertCampaignConfig(ctx context.Context, arg sqlc.UpsertCampaignConfigParams) (sqlc.CampaignConfig, error)
	CreateCampaignStageSnapshot(ctx context.Context, arg sqlc.CreateCampaignStageSnapshotParams) (sqlc.CampaignStageSnapshot, error)
	ListCampaignStageSnapshotsByProject(ctx context.Context, projectID uuid.UUID) ([]sqlc.CampaignStageSnapshot, error)
	GetCampaignStageSnapshotByID(ctx context.Context, id uuid.UUID) (sqlc.CampaignStageSnapshot, error)
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

func (r *PGRepository) GetProjectMemberByUser(ctx context.Context, arg sqlc.GetProjectMemberByUserParams) (sqlc.ProjectMember, error) {
	return r.queries().GetProjectMemberByUser(ctx, arg)
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

func (r *PGRepository) GetCampaignConfigByProject(ctx context.Context, projectID uuid.UUID) (sqlc.CampaignConfig, error) {
	row, err := r.queries().GetCampaignConfigByProject(ctx, projectID)
	if err != nil {
		return sqlc.CampaignConfig{}, err
	}
	return sqlc.CampaignConfig(row), nil
}

func (r *PGRepository) UpsertCampaignConfig(ctx context.Context, arg sqlc.UpsertCampaignConfigParams) (sqlc.CampaignConfig, error) {
	row, err := r.queries().UpsertCampaignConfig(ctx, arg)
	if err != nil {
		return sqlc.CampaignConfig{}, err
	}
	return sqlc.CampaignConfig(row), nil
}

func (r *PGRepository) CreateCampaignStageSnapshot(ctx context.Context, arg sqlc.CreateCampaignStageSnapshotParams) (sqlc.CampaignStageSnapshot, error) {
	row, err := r.queries().CreateCampaignStageSnapshot(ctx, arg)
	if err != nil {
		return sqlc.CampaignStageSnapshot{}, err
	}
	return sqlc.CampaignStageSnapshot{
		ID:                  row.ID,
		ProjectID:           row.ProjectID,
		Stage:               row.Stage,
		CreatedBy:           row.CreatedBy,
		CreatedByName:       row.CreatedByName,
		CreatedAt:           row.CreatedAt,
		FollowersCount:      row.FollowersCount,
		Notes:               row.Notes,
		GrossAmountSnapshot: row.GrossAmountSnapshot,
		TeamPoolSnapshot:    row.TeamPoolSnapshot,
	}, nil
}

func (r *PGRepository) ListCampaignStageSnapshotsByProject(ctx context.Context, projectID uuid.UUID) ([]sqlc.CampaignStageSnapshot, error) {
	rows, err := r.queries().ListCampaignStageSnapshotsByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	items := make([]sqlc.CampaignStageSnapshot, 0, len(rows))
	for _, row := range rows {
		items = append(items, sqlc.CampaignStageSnapshot{
			ID:                  row.ID,
			ProjectID:           row.ProjectID,
			Stage:               row.Stage,
			CreatedBy:           row.CreatedBy,
			CreatedByName:       row.CreatedByName,
			CreatedAt:           row.CreatedAt,
			FollowersCount:      row.FollowersCount,
			Notes:               row.Notes,
			GrossAmountSnapshot: row.GrossAmountSnapshot,
			TeamPoolSnapshot:    row.TeamPoolSnapshot,
		})
	}
	return items, nil
}

func (r *PGRepository) GetCampaignStageSnapshotByID(ctx context.Context, id uuid.UUID) (sqlc.CampaignStageSnapshot, error) {
	row, err := r.queries().GetCampaignStageSnapshotByID(ctx, id)
	if err != nil {
		return sqlc.CampaignStageSnapshot{}, err
	}
	return sqlc.CampaignStageSnapshot{
		ID:                  row.ID,
		ProjectID:           row.ProjectID,
		Stage:               row.Stage,
		CreatedBy:           row.CreatedBy,
		CreatedByName:       row.CreatedByName,
		CreatedAt:           row.CreatedAt,
		FollowersCount:      row.FollowersCount,
		Notes:               row.Notes,
		GrossAmountSnapshot: row.GrossAmountSnapshot,
		TeamPoolSnapshot:    row.TeamPoolSnapshot,
	}, nil
}
