package workflow

import (
	"context"

	"isbudget/backend/internal/db/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	WithTx(ctx context.Context, fn func(repo Repository) error) error
	GetProjectMemberByUser(ctx context.Context, req sqlc.GetProjectMemberByUserParams) (sqlc.ProjectMember, error)
	CreateWorkflowNode(ctx context.Context, req sqlc.CreateWorkflowNodeParams) (sqlc.WorkflowNode, error)
	CreateWorkflowDependency(ctx context.Context, req sqlc.CreateWorkflowDependencyParams) (sqlc.WorkflowDependency, error)
	GetWorkflowNodeByID(ctx context.Context, id uuid.UUID) (sqlc.WorkflowNode, error)
	ListWorkflowNodesByProject(ctx context.Context, projectID uuid.UUID) ([]sqlc.WorkflowNode, error)
	ListWorkflowNodesByIDs(ctx context.Context, ids []uuid.UUID) ([]sqlc.WorkflowNode, error)
	AssignWorkflowNodeUser(ctx context.Context, req sqlc.AssignWorkflowNodeUserParams) (sqlc.WorkflowNode, error)
	CompleteWorkflowNode(ctx context.Context, req sqlc.CompleteWorkflowNodeParams) (sqlc.WorkflowNode, error)
	MarkWorkflowNodeReady(ctx context.Context, id uuid.UUID) (sqlc.WorkflowNode, error)
	ListWorkflowDependenciesByProject(ctx context.Context, projectID uuid.UUID) ([]sqlc.WorkflowDependency, error)
	ListWorkflowDependenciesByChild(ctx context.Context, childID uuid.UUID) ([]sqlc.WorkflowDependency, error)
	ListWorkflowDependenciesByParent(ctx context.Context, parentID uuid.UUID) ([]sqlc.WorkflowDependency, error)
	CreateContributionLedgerEntry(ctx context.Context, req sqlc.CreateContributionLedgerEntryParams) (sqlc.ContributionLedger, error)
	ListContributionLedgerEntriesByProject(ctx context.Context, projectID uuid.UUID) ([]sqlc.ContributionLedger, error)
}

type PGRepository struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

func NewPGRepository(q *sqlc.Queries, pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{q: q, pool: pool}
}

func (r *PGRepository) WithTx(ctx context.Context, fn func(repo Repository) error) error {
	if r.pool == nil {
		return fn(r)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	txRepo := &PGRepository{
		q: r.q.WithTx(tx),
	}
	if err := fn(txRepo); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) GetProjectMemberByUser(ctx context.Context, req sqlc.GetProjectMemberByUserParams) (sqlc.ProjectMember, error) {
	return r.q.GetProjectMemberByUser(ctx, req)
}

func (r *PGRepository) CreateWorkflowNode(ctx context.Context, req sqlc.CreateWorkflowNodeParams) (sqlc.WorkflowNode, error) {
	return r.q.CreateWorkflowNode(ctx, req)
}

func (r *PGRepository) CreateWorkflowDependency(ctx context.Context, req sqlc.CreateWorkflowDependencyParams) (sqlc.WorkflowDependency, error) {
	return r.q.CreateWorkflowDependency(ctx, req)
}

func (r *PGRepository) GetWorkflowNodeByID(ctx context.Context, id uuid.UUID) (sqlc.WorkflowNode, error) {
	return r.q.GetWorkflowNodeByID(ctx, id)
}

func (r *PGRepository) ListWorkflowNodesByProject(ctx context.Context, projectID uuid.UUID) ([]sqlc.WorkflowNode, error) {
	return r.q.ListWorkflowNodesByProject(ctx, projectID)
}

func (r *PGRepository) ListWorkflowNodesByIDs(ctx context.Context, ids []uuid.UUID) ([]sqlc.WorkflowNode, error) {
	return r.q.ListWorkflowNodesByIDs(ctx, ids)
}

func (r *PGRepository) AssignWorkflowNodeUser(ctx context.Context, req sqlc.AssignWorkflowNodeUserParams) (sqlc.WorkflowNode, error) {
	return r.q.AssignWorkflowNodeUser(ctx, req)
}

func (r *PGRepository) CompleteWorkflowNode(ctx context.Context, req sqlc.CompleteWorkflowNodeParams) (sqlc.WorkflowNode, error) {
	return r.q.CompleteWorkflowNode(ctx, req)
}

func (r *PGRepository) MarkWorkflowNodeReady(ctx context.Context, id uuid.UUID) (sqlc.WorkflowNode, error) {
	return r.q.MarkWorkflowNodeReady(ctx, id)
}

func (r *PGRepository) ListWorkflowDependenciesByProject(ctx context.Context, projectID uuid.UUID) ([]sqlc.WorkflowDependency, error) {
	return r.q.ListWorkflowDependenciesByProject(ctx, projectID)
}

func (r *PGRepository) ListWorkflowDependenciesByChild(ctx context.Context, childID uuid.UUID) ([]sqlc.WorkflowDependency, error) {
	return r.q.ListWorkflowDependenciesByChild(ctx, childID)
}

func (r *PGRepository) ListWorkflowDependenciesByParent(ctx context.Context, parentID uuid.UUID) ([]sqlc.WorkflowDependency, error) {
	return r.q.ListWorkflowDependenciesByParent(ctx, parentID)
}

func (r *PGRepository) CreateContributionLedgerEntry(ctx context.Context, req sqlc.CreateContributionLedgerEntryParams) (sqlc.ContributionLedger, error) {
	return r.q.CreateContributionLedgerEntry(ctx, req)
}

func (r *PGRepository) ListContributionLedgerEntriesByProject(ctx context.Context, projectID uuid.UUID) ([]sqlc.ContributionLedger, error) {
	return r.q.ListContributionLedgerEntriesByProject(ctx, projectID)
}
