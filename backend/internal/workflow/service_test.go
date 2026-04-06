package workflow

import (
	"context"
	"slices"
	"testing"
	"time"

	"isbudget/backend/internal/db/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type fakeRepository struct {
	members map[uuid.UUID]map[uuid.UUID]bool
	nodes   map[uuid.UUID]sqlc.WorkflowNode
	deps    map[uuid.UUID]sqlc.WorkflowDependency
	ledger  map[uuid.UUID]sqlc.ContributionLedger
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		members: make(map[uuid.UUID]map[uuid.UUID]bool),
		nodes:   make(map[uuid.UUID]sqlc.WorkflowNode),
		deps:    make(map[uuid.UUID]sqlc.WorkflowDependency),
		ledger:  make(map[uuid.UUID]sqlc.ContributionLedger),
	}
}

func (r *fakeRepository) WithTx(_ context.Context, fn func(repo Repository) error) error {
	return fn(r)
}

func (r *fakeRepository) GetProjectMemberByUser(_ context.Context, req sqlc.GetProjectMemberByUserParams) (sqlc.ProjectMember, error) {
	if r.members[req.ProjectID][req.UserID] {
		return sqlc.ProjectMember{
			ProjectID: req.ProjectID,
			UserID:    req.UserID,
		}, nil
	}
	return sqlc.ProjectMember{}, pgx.ErrNoRows
}

func (r *fakeRepository) CreateWorkflowNode(_ context.Context, req sqlc.CreateWorkflowNodeParams) (sqlc.WorkflowNode, error) {
	now := time.Now().UTC()
	node := sqlc.WorkflowNode{
		ID:                 uuid.New(),
		ProjectID:          req.ProjectID,
		Title:              req.Title,
		Description:        req.Description,
		Status:             req.Status,
		AssigneeUserID:     req.AssigneeUserID,
		ContributionPoints: req.ContributionPoints,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	r.nodes[node.ID] = node
	return node, nil
}

func (r *fakeRepository) CreateWorkflowDependency(_ context.Context, req sqlc.CreateWorkflowDependencyParams) (sqlc.WorkflowDependency, error) {
	dependency := sqlc.WorkflowDependency{
		ID:           uuid.New(),
		ParentNodeID: req.ParentNodeID,
		ChildNodeID:  req.ChildNodeID,
		CreatedAt:    time.Now().UTC(),
	}
	r.deps[dependency.ID] = dependency
	return dependency, nil
}

func (r *fakeRepository) GetWorkflowNodeByID(_ context.Context, id uuid.UUID) (sqlc.WorkflowNode, error) {
	node, ok := r.nodes[id]
	if !ok {
		return sqlc.WorkflowNode{}, pgx.ErrNoRows
	}
	return node, nil
}

func (r *fakeRepository) ListWorkflowNodesByProject(_ context.Context, projectID uuid.UUID) ([]sqlc.WorkflowNode, error) {
	out := make([]sqlc.WorkflowNode, 0)
	for _, node := range r.nodes {
		if node.ProjectID == projectID {
			out = append(out, node)
		}
	}
	slices.SortFunc(out, func(a, b sqlc.WorkflowNode) int {
		return a.CreatedAt.Compare(b.CreatedAt)
	})
	return out, nil
}

func (r *fakeRepository) ListWorkflowNodesByIDs(_ context.Context, ids []uuid.UUID) ([]sqlc.WorkflowNode, error) {
	out := make([]sqlc.WorkflowNode, 0, len(ids))
	for _, id := range ids {
		node, ok := r.nodes[id]
		if ok {
			out = append(out, node)
		}
	}
	return out, nil
}

func (r *fakeRepository) AssignWorkflowNodeUser(_ context.Context, req sqlc.AssignWorkflowNodeUserParams) (sqlc.WorkflowNode, error) {
	node, ok := r.nodes[req.ID]
	if !ok {
		return sqlc.WorkflowNode{}, pgx.ErrNoRows
	}
	node.AssigneeUserID = req.AssigneeUserID
	node.UpdatedAt = time.Now().UTC()
	r.nodes[node.ID] = node
	return node, nil
}

func (r *fakeRepository) CompleteWorkflowNode(_ context.Context, req sqlc.CompleteWorkflowNodeParams) (sqlc.WorkflowNode, error) {
	node, ok := r.nodes[req.ID]
	if !ok {
		return sqlc.WorkflowNode{}, pgx.ErrNoRows
	}
	node.Status = NodeStatusCompleted
	node.CompletedAt = req.CompletedAt
	node.CompletedBy = req.CompletedBy
	node.UpdatedAt = time.Now().UTC()
	r.nodes[node.ID] = node
	return node, nil
}

func (r *fakeRepository) MarkWorkflowNodeReady(_ context.Context, id uuid.UUID) (sqlc.WorkflowNode, error) {
	node, ok := r.nodes[id]
	if !ok {
		return sqlc.WorkflowNode{}, pgx.ErrNoRows
	}
	node.Status = NodeStatusReady
	node.UpdatedAt = time.Now().UTC()
	r.nodes[node.ID] = node
	return node, nil
}

func (r *fakeRepository) ListWorkflowDependenciesByProject(_ context.Context, projectID uuid.UUID) ([]sqlc.WorkflowDependency, error) {
	out := make([]sqlc.WorkflowDependency, 0)
	for _, dependency := range r.deps {
		child, ok := r.nodes[dependency.ChildNodeID]
		if ok && child.ProjectID == projectID {
			out = append(out, dependency)
		}
	}
	return out, nil
}

func (r *fakeRepository) ListWorkflowDependenciesByChild(_ context.Context, childID uuid.UUID) ([]sqlc.WorkflowDependency, error) {
	out := make([]sqlc.WorkflowDependency, 0)
	for _, dependency := range r.deps {
		if dependency.ChildNodeID == childID {
			out = append(out, dependency)
		}
	}
	return out, nil
}

func (r *fakeRepository) ListWorkflowDependenciesByParent(_ context.Context, parentID uuid.UUID) ([]sqlc.WorkflowDependency, error) {
	out := make([]sqlc.WorkflowDependency, 0)
	for _, dependency := range r.deps {
		if dependency.ParentNodeID == parentID {
			out = append(out, dependency)
		}
	}
	return out, nil
}

func (r *fakeRepository) CreateContributionLedgerEntry(_ context.Context, req sqlc.CreateContributionLedgerEntryParams) (sqlc.ContributionLedger, error) {
	entry := sqlc.ContributionLedger{
		ID:        uuid.New(),
		ProjectID: req.ProjectID,
		NodeID:    req.NodeID,
		UserID:    req.UserID,
		Points:    req.Points,
		CreatedAt: time.Now().UTC(),
	}
	r.ledger[entry.NodeID] = entry
	return entry, nil
}

func (r *fakeRepository) ListContributionLedgerEntriesByProject(_ context.Context, projectID uuid.UUID) ([]sqlc.ContributionLedger, error) {
	out := make([]sqlc.ContributionLedger, 0)
	for _, entry := range r.ledger {
		if entry.ProjectID == projectID {
			out = append(out, entry)
		}
	}
	return out, nil
}

func (r *fakeRepository) addProjectMember(projectID, userID uuid.UUID) {
	if r.members[projectID] == nil {
		r.members[projectID] = make(map[uuid.UUID]bool)
	}
	r.members[projectID][userID] = true
}

func TestCompleteNodeMarksNodeCompleted(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo)

	projectID := uuid.New()
	actorID := uuid.New()
	assigneeID := uuid.New()
	nodeID := uuid.New()

	repo.addProjectMember(projectID, actorID)
	repo.addProjectMember(projectID, assigneeID)
	repo.nodes[nodeID] = sqlc.WorkflowNode{
		ID:                 nodeID,
		ProjectID:          projectID,
		Status:             NodeStatusReady,
		AssigneeUserID:     &assigneeID,
		ContributionPoints: 3,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}

	node, err := svc.CompleteNode(context.Background(), actorID, nodeID)
	if err != nil {
		t.Fatalf("CompleteNode() error = %v", err)
	}
	if node.Status != NodeStatusCompleted {
		t.Fatalf("expected completed status, got %q", node.Status)
	}
	if node.CompletedBy == nil || *node.CompletedBy != actorID {
		t.Fatalf("expected completed_by %s, got %+v", actorID, node.CompletedBy)
	}
	if node.CompletedAt == nil {
		t.Fatal("expected completed_at to be set")
	}
}

func TestCompleteNodeUnlocksDependentNode(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo)

	projectID := uuid.New()
	actorID := uuid.New()
	assigneeID := uuid.New()
	parentID := uuid.New()
	childID := uuid.New()

	repo.addProjectMember(projectID, actorID)
	repo.addProjectMember(projectID, assigneeID)
	repo.nodes[parentID] = sqlc.WorkflowNode{
		ID:                 parentID,
		ProjectID:          projectID,
		Status:             NodeStatusReady,
		AssigneeUserID:     &assigneeID,
		ContributionPoints: 2,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
	repo.nodes[childID] = sqlc.WorkflowNode{
		ID:                 childID,
		ProjectID:          projectID,
		Status:             NodeStatusPending,
		ContributionPoints: 1,
		CreatedAt:          time.Now().UTC().Add(time.Second),
		UpdatedAt:          time.Now().UTC().Add(time.Second),
	}
	dependency := sqlc.WorkflowDependency{
		ID:           uuid.New(),
		ParentNodeID: parentID,
		ChildNodeID:  childID,
		CreatedAt:    time.Now().UTC(),
	}
	repo.deps[dependency.ID] = dependency

	if _, err := svc.CompleteNode(context.Background(), actorID, parentID); err != nil {
		t.Fatalf("CompleteNode() error = %v", err)
	}

	child := repo.nodes[childID]
	if child.Status != NodeStatusReady {
		t.Fatalf("expected child node to become ready, got %q", child.Status)
	}
}

func TestCompleteNodeWritesContributionLedgerEntry(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo)

	projectID := uuid.New()
	actorID := uuid.New()
	assigneeID := uuid.New()
	nodeID := uuid.New()

	repo.addProjectMember(projectID, actorID)
	repo.addProjectMember(projectID, assigneeID)
	repo.nodes[nodeID] = sqlc.WorkflowNode{
		ID:                 nodeID,
		ProjectID:          projectID,
		Status:             NodeStatusReady,
		AssigneeUserID:     &assigneeID,
		ContributionPoints: 8,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}

	if _, err := svc.CompleteNode(context.Background(), actorID, nodeID); err != nil {
		t.Fatalf("CompleteNode() error = %v", err)
	}

	entry, ok := repo.ledger[nodeID]
	if !ok {
		t.Fatal("expected contribution ledger entry to be created")
	}
	if entry.UserID != assigneeID {
		t.Fatalf("expected ledger user %s, got %s", assigneeID, entry.UserID)
	}
	if entry.Points != 8 {
		t.Fatalf("expected 8 ledger points, got %d", entry.Points)
	}
}
