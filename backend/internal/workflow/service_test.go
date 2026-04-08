package workflow

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"isbudget/backend/internal/db/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type fakeRepository struct {
	members map[uuid.UUID]map[uuid.UUID]sqlc.ProjectMemberWithUser
	nodes   map[uuid.UUID]sqlc.WorkflowNode
	deps    map[uuid.UUID]sqlc.WorkflowDependency
	ledger  map[uuid.UUID]sqlc.ContributionLedger
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		members: make(map[uuid.UUID]map[uuid.UUID]sqlc.ProjectMemberWithUser),
		nodes:   make(map[uuid.UUID]sqlc.WorkflowNode),
		deps:    make(map[uuid.UUID]sqlc.WorkflowDependency),
		ledger:  make(map[uuid.UUID]sqlc.ContributionLedger),
	}
}

func (r *fakeRepository) WithTx(_ context.Context, fn func(repo Repository) error) error {
	return fn(r)
}

func (r *fakeRepository) GetProjectMemberByUser(_ context.Context, req sqlc.GetProjectMemberByUserParams) (sqlc.ProjectMember, error) {
	member, ok := r.members[req.ProjectID][req.UserID]
	if !ok {
		return sqlc.ProjectMember{}, pgx.ErrNoRows
	}
	return sqlc.ProjectMember{
		ID:        member.ID,
		ProjectID: member.ProjectID,
		UserID:    member.UserID,
		Role:      member.Role,
		CreatedAt: member.CreatedAt,
	}, nil
}

func (r *fakeRepository) ListProjectMembers(_ context.Context, projectID uuid.UUID) ([]sqlc.ProjectMemberWithUser, error) {
	members := make([]sqlc.ProjectMemberWithUser, 0, len(r.members[projectID]))
	for _, member := range r.members[projectID] {
		members = append(members, member)
	}
	slices.SortFunc(members, func(a, b sqlc.ProjectMemberWithUser) int {
		return a.CreatedAt.Compare(b.CreatedAt)
	})
	return members, nil
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
	if exists, _ := r.WorkflowDependencyExists(context.Background(), sqlc.WorkflowDependencyExistsParams{
		ParentNodeID: req.ParentNodeID,
		ChildNodeID:  req.ChildNodeID,
	}); exists {
		return sqlc.WorkflowDependency{}, &pgconn.PgError{Code: uniqueViolationCode}
	}
	dependency := sqlc.WorkflowDependency{
		ID:           uuid.New(),
		ParentNodeID: req.ParentNodeID,
		ChildNodeID:  req.ChildNodeID,
		CreatedAt:    time.Now().UTC(),
	}
	r.deps[dependency.ID] = dependency
	return dependency, nil
}

func (r *fakeRepository) DeleteWorkflowDependency(_ context.Context, parentNodeID, childNodeID uuid.UUID) (int64, error) {
	for dependencyID, dependency := range r.deps {
		if dependency.ParentNodeID == parentNodeID && dependency.ChildNodeID == childNodeID {
			delete(r.deps, dependencyID)
			return 1, nil
		}
	}
	return 0, nil
}

func (r *fakeRepository) WorkflowDependencyExists(_ context.Context, req sqlc.WorkflowDependencyExistsParams) (bool, error) {
	for _, dependency := range r.deps {
		if dependency.ParentNodeID == req.ParentNodeID && dependency.ChildNodeID == req.ChildNodeID {
			return true, nil
		}
	}
	return false, nil
}

func (r *fakeRepository) WorkflowPathExists(_ context.Context, req sqlc.WorkflowPathExistsParams) (bool, error) {
	queue := []uuid.UUID{req.Column1}
	visited := map[uuid.UUID]struct{}{req.Column1: {}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if current == req.ID {
			return true, nil
		}
		for _, dependency := range r.deps {
			if dependency.ParentNodeID != current {
				continue
			}
			if _, seen := visited[dependency.ChildNodeID]; seen {
				continue
			}
			visited[dependency.ChildNodeID] = struct{}{}
			queue = append(queue, dependency.ChildNodeID)
		}
	}

	return false, nil
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
	node.Status = req.Status
	node.UpdatedAt = time.Now().UTC()
	r.nodes[node.ID] = node
	return node, nil
}

func (r *fakeRepository) UpdateWorkflowNode(_ context.Context, req sqlc.UpdateWorkflowNodeParams) (sqlc.WorkflowNode, error) {
	node, ok := r.nodes[req.ID]
	if !ok {
		return sqlc.WorkflowNode{}, pgx.ErrNoRows
	}
	node.Title = req.Title
	node.Description = req.Description
	node.ContributionPoints = req.ContributionPoints
	node.UpdatedAt = time.Now().UTC()
	r.nodes[node.ID] = node
	return node, nil
}

func (r *fakeRepository) CompleteWorkflowNode(_ context.Context, req sqlc.CompleteWorkflowNodeParams) (sqlc.WorkflowNode, error) {
	node, ok := r.nodes[req.ID]
	if !ok {
		return sqlc.WorkflowNode{}, pgx.ErrNoRows
	}
	node.Status = NodeStatusDone
	node.CompletedAt = req.CompletedAt
	node.CompletedBy = req.CompletedBy
	node.UpdatedAt = time.Now().UTC()
	r.nodes[node.ID] = node
	return node, nil
}

func (r *fakeRepository) MarkWorkflowNodeReady(_ context.Context, id uuid.UUID) (sqlc.WorkflowNode, error) {
	return r.setNodeStatus(id, NodeStatusReady)
}

func (r *fakeRepository) MarkWorkflowNodePlanned(_ context.Context, id uuid.UUID) (sqlc.WorkflowNode, error) {
	return r.setNodeStatus(id, NodeStatusPlanned)
}

func (r *fakeRepository) MarkWorkflowNodeInProgress(_ context.Context, id uuid.UUID) (sqlc.WorkflowNode, error) {
	return r.setNodeStatus(id, NodeStatusInProgress)
}

func (r *fakeRepository) setNodeStatus(id uuid.UUID, status string) (sqlc.WorkflowNode, error) {
	node, ok := r.nodes[id]
	if !ok {
		return sqlc.WorkflowNode{}, pgx.ErrNoRows
	}
	node.Status = status
	node.UpdatedAt = time.Now().UTC()
	r.nodes[id] = node
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
	if _, exists := r.ledger[req.NodeID]; exists {
		return sqlc.ContributionLedger{}, &pgconn.PgError{Code: uniqueViolationCode}
	}
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

func (r *fakeRepository) UpdateContributionLedgerPointsByNode(_ context.Context, req sqlc.UpdateContributionLedgerPointsByNodeParams) error {
	entry, ok := r.ledger[req.NodeID]
	if !ok {
		return nil
	}
	entry.Points = req.Points
	r.ledger[entry.NodeID] = entry
	return nil
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

func (r *fakeRepository) addProjectMember(projectID, userID uuid.UUID, name string) {
	r.addProjectMemberWithRole(projectID, userID, name, sqlc.ProjectRolePM)
}

func (r *fakeRepository) addProjectMemberWithRole(projectID, userID uuid.UUID, name string, role sqlc.ProjectRole) {
	if r.members[projectID] == nil {
		r.members[projectID] = make(map[uuid.UUID]sqlc.ProjectMemberWithUser)
	}
	r.members[projectID][userID] = sqlc.ProjectMemberWithUser{
		ID:        uuid.New(),
		ProjectID: projectID,
		UserID:    userID,
		Role:      role,
		CreatedAt: time.Now().UTC(),
		Email:     name + "@example.com",
		FullName:  name,
		Nickname:  name,
	}
}

func TestAssignReadyNodeMovesToInProgress(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo)

	projectID := uuid.New()
	actorID := uuid.New()
	assigneeID := uuid.New()
	nodeID := uuid.New()

	repo.addProjectMember(projectID, actorID, "actor")
	repo.addProjectMember(projectID, assigneeID, "assignee")
	repo.nodes[nodeID] = sqlc.WorkflowNode{
		ID:                 nodeID,
		ProjectID:          projectID,
		Status:             NodeStatusReady,
		ContributionPoints: 3,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}

	node, err := svc.AssignUser(context.Background(), actorID, nodeID, assigneeID)
	if err != nil {
		t.Fatalf("AssignUser() error = %v", err)
	}
	if node.Status != NodeStatusInProgress {
		t.Fatalf("expected in_progress status, got %q", node.Status)
	}
	if node.AssigneeUserID == nil || *node.AssigneeUserID != assigneeID {
		t.Fatalf("expected assignee %s, got %+v", assigneeID, node.AssigneeUserID)
	}
}

func TestTeamSelfTakeReadyNodeMovesToInProgress(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo)

	projectID := uuid.New()
	teamMemberID := uuid.New()
	nodeID := uuid.New()

	repo.addProjectMemberWithRole(projectID, teamMemberID, "teammate", sqlc.ProjectRoleTEAM)
	repo.nodes[nodeID] = sqlc.WorkflowNode{
		ID:                 nodeID,
		ProjectID:          projectID,
		Status:             NodeStatusReady,
		ContributionPoints: 2,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}

	node, err := svc.AssignUser(context.Background(), teamMemberID, nodeID, teamMemberID)
	if err != nil {
		t.Fatalf("AssignUser() error = %v", err)
	}
	if node.Status != NodeStatusInProgress {
		t.Fatalf("expected in_progress status, got %q", node.Status)
	}
	if node.AssigneeUserID == nil || *node.AssigneeUserID != teamMemberID {
		t.Fatalf("expected self assignment to %s, got %+v", teamMemberID, node.AssigneeUserID)
	}
}

func TestAssigneeCanReturnInProgressNodeToReady(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo)

	projectID := uuid.New()
	teamMemberID := uuid.New()
	nodeID := uuid.New()

	repo.addProjectMemberWithRole(projectID, teamMemberID, "teammate", sqlc.ProjectRoleTEAM)
	repo.nodes[nodeID] = sqlc.WorkflowNode{
		ID:                 nodeID,
		ProjectID:          projectID,
		Status:             NodeStatusInProgress,
		AssigneeUserID:     &teamMemberID,
		ContributionPoints: 2,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}

	node, err := svc.UnassignUser(context.Background(), teamMemberID, nodeID)
	if err != nil {
		t.Fatalf("UnassignUser() error = %v", err)
	}
	if node.Status != NodeStatusReady {
		t.Fatalf("expected ready status, got %q", node.Status)
	}
	if node.AssigneeUserID != nil {
		t.Fatalf("expected assignee to be cleared, got %+v", node.AssigneeUserID)
	}
}

func TestOtherMemberCannotReturnAssignedNode(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo)

	projectID := uuid.New()
	assigneeID := uuid.New()
	otherID := uuid.New()
	nodeID := uuid.New()

	repo.addProjectMemberWithRole(projectID, assigneeID, "assignee", sqlc.ProjectRoleTEAM)
	repo.addProjectMemberWithRole(projectID, otherID, "other", sqlc.ProjectRoleTEAM)
	repo.nodes[nodeID] = sqlc.WorkflowNode{
		ID:                 nodeID,
		ProjectID:          projectID,
		Status:             NodeStatusInProgress,
		AssigneeUserID:     &assigneeID,
		ContributionPoints: 2,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}

	_, err := svc.UnassignUser(context.Background(), otherID, nodeID)
	if !errors.Is(err, ErrWorkflowNodeNotOwnedByActor) {
		t.Fatalf("UnassignUser() error = %v, want ErrWorkflowNodeNotOwnedByActor", err)
	}
}

func TestTakeBlockedNodeFails(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo)

	projectID := uuid.New()
	teamMemberID := uuid.New()
	nodeID := uuid.New()

	repo.addProjectMemberWithRole(projectID, teamMemberID, "teammate", sqlc.ProjectRoleTEAM)
	repo.nodes[nodeID] = sqlc.WorkflowNode{
		ID:                 nodeID,
		ProjectID:          projectID,
		Status:             NodeStatusPlanned,
		ContributionPoints: 2,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}

	_, err := svc.AssignUser(context.Background(), teamMemberID, nodeID, teamMemberID)
	if !errors.Is(err, ErrWorkflowNodeNotAssignable) {
		t.Fatalf("AssignUser() error = %v, want ErrWorkflowNodeNotAssignable", err)
	}
}

func TestTakeDoneNodeFails(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo)

	projectID := uuid.New()
	teamMemberID := uuid.New()
	nodeID := uuid.New()

	repo.addProjectMemberWithRole(projectID, teamMemberID, "teammate", sqlc.ProjectRoleTEAM)
	repo.nodes[nodeID] = sqlc.WorkflowNode{
		ID:                 nodeID,
		ProjectID:          projectID,
		Status:             NodeStatusDone,
		ContributionPoints: 2,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}

	_, err := svc.AssignUser(context.Background(), teamMemberID, nodeID, teamMemberID)
	if !errors.Is(err, ErrWorkflowNodeAlreadyDone) {
		t.Fatalf("AssignUser() error = %v, want ErrWorkflowNodeAlreadyDone", err)
	}
}

func TestCompleteNodeRejectsDuplicateCompletion(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo)

	projectID := uuid.New()
	assigneeID := uuid.New()
	nodeID := uuid.New()

	repo.addProjectMember(projectID, assigneeID, "assignee")
	repo.nodes[nodeID] = sqlc.WorkflowNode{
		ID:                 nodeID,
		ProjectID:          projectID,
		Status:             NodeStatusInProgress,
		AssigneeUserID:     &assigneeID,
		ContributionPoints: 3,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}

	if _, err := svc.CompleteNode(context.Background(), assigneeID, nodeID); err != nil {
		t.Fatalf("first CompleteNode() error = %v", err)
	}

	_, err := svc.CompleteNode(context.Background(), assigneeID, nodeID)
	if !errors.Is(err, ErrWorkflowNodeAlreadyDone) {
		t.Fatalf("second CompleteNode() error = %v, want ErrWorkflowNodeAlreadyDone", err)
	}
}

func TestCompleteNodeDoesNotDoubleWriteLedger(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo)

	projectID := uuid.New()
	assigneeID := uuid.New()
	nodeID := uuid.New()

	repo.addProjectMember(projectID, assigneeID, "assignee")
	repo.nodes[nodeID] = sqlc.WorkflowNode{
		ID:                 nodeID,
		ProjectID:          projectID,
		Status:             NodeStatusInProgress,
		AssigneeUserID:     &assigneeID,
		ContributionPoints: 8,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}

	if _, err := svc.CompleteNode(context.Background(), assigneeID, nodeID); err != nil {
		t.Fatalf("first CompleteNode() error = %v", err)
	}

	// Simulate a stale caller that moved the node back out of done without clearing the ledger.
	node := repo.nodes[nodeID]
	node.Status = NodeStatusInProgress
	repo.nodes[nodeID] = node

	_, err := svc.CompleteNode(context.Background(), assigneeID, nodeID)
	if !errors.Is(err, ErrWorkflowContributionAlreadyRecorded) {
		t.Fatalf("second CompleteNode() error = %v, want ErrWorkflowContributionAlreadyRecorded", err)
	}
	if len(repo.ledger) != 1 {
		t.Fatalf("expected exactly 1 ledger entry, got %d", len(repo.ledger))
	}
}

func TestAddDependencyRejectsDuplicateDependency(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo)

	projectID := uuid.New()
	actorID := uuid.New()
	parentID := uuid.New()
	childID := uuid.New()

	repo.addProjectMember(projectID, actorID, "actor")
	repo.nodes[parentID] = sqlc.WorkflowNode{ID: parentID, ProjectID: projectID, Status: NodeStatusReady, ContributionPoints: 1, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	repo.nodes[childID] = sqlc.WorkflowNode{ID: childID, ProjectID: projectID, Status: NodeStatusReady, ContributionPoints: 1, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	dependency, _ := repo.CreateWorkflowDependency(context.Background(), sqlc.CreateWorkflowDependencyParams{ParentNodeID: parentID, ChildNodeID: childID})
	repo.deps[dependency.ID] = dependency

	_, err := svc.AddDependency(context.Background(), actorID, childID, parentID)
	if !errors.Is(err, ErrWorkflowDependencyAlreadyExists) {
		t.Fatalf("AddDependency() error = %v, want ErrWorkflowDependencyAlreadyExists", err)
	}
}

func TestAddDependencyRejectsSelfDependency(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo)

	projectID := uuid.New()
	actorID := uuid.New()
	nodeID := uuid.New()

	repo.addProjectMember(projectID, actorID, "actor")
	repo.nodes[nodeID] = sqlc.WorkflowNode{ID: nodeID, ProjectID: projectID, Status: NodeStatusReady, ContributionPoints: 1, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}

	_, err := svc.AddDependency(context.Background(), actorID, nodeID, nodeID)
	if !errors.Is(err, ErrWorkflowDependencySelfReference) {
		t.Fatalf("AddDependency() error = %v, want ErrWorkflowDependencySelfReference", err)
	}
}

func TestAddDependencyRejectsCrossProjectDependency(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo)

	projectA := uuid.New()
	projectB := uuid.New()
	actorID := uuid.New()
	parentID := uuid.New()
	childID := uuid.New()

	repo.addProjectMember(projectA, actorID, "actor")
	repo.nodes[parentID] = sqlc.WorkflowNode{ID: parentID, ProjectID: projectA, Status: NodeStatusReady, ContributionPoints: 1, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	repo.nodes[childID] = sqlc.WorkflowNode{ID: childID, ProjectID: projectB, Status: NodeStatusReady, ContributionPoints: 1, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}

	_, err := svc.AddDependency(context.Background(), actorID, childID, parentID)
	if !errors.Is(err, ErrWorkflowProjectMemberRequired) {
		t.Fatalf("AddDependency() error = %v, want ErrWorkflowProjectMemberRequired", err)
	}

	repo.addProjectMember(projectB, actorID, "actor")
	_, err = svc.AddDependency(context.Background(), actorID, childID, parentID)
	if !errors.Is(err, ErrWorkflowDependencyProjectMismatch) {
		t.Fatalf("AddDependency() error = %v, want ErrWorkflowDependencyProjectMismatch", err)
	}
}

func TestAddDependencyRejectsCycle(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo)

	projectID := uuid.New()
	actorID := uuid.New()
	nodeA := uuid.New()
	nodeB := uuid.New()
	nodeC := uuid.New()

	repo.addProjectMember(projectID, actorID, "actor")
	repo.nodes[nodeA] = sqlc.WorkflowNode{ID: nodeA, ProjectID: projectID, Status: NodeStatusReady, ContributionPoints: 1, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	repo.nodes[nodeB] = sqlc.WorkflowNode{ID: nodeB, ProjectID: projectID, Status: NodeStatusReady, ContributionPoints: 1, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	repo.nodes[nodeC] = sqlc.WorkflowNode{ID: nodeC, ProjectID: projectID, Status: NodeStatusReady, ContributionPoints: 1, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	r1, _ := repo.CreateWorkflowDependency(context.Background(), sqlc.CreateWorkflowDependencyParams{ParentNodeID: nodeA, ChildNodeID: nodeB})
	r2, _ := repo.CreateWorkflowDependency(context.Background(), sqlc.CreateWorkflowDependencyParams{ParentNodeID: nodeB, ChildNodeID: nodeC})
	repo.deps[r1.ID] = r1
	repo.deps[r2.ID] = r2

	_, err := svc.AddDependency(context.Background(), actorID, nodeA, nodeC)
	if !errors.Is(err, ErrWorkflowDependencyCycle) {
		t.Fatalf("AddDependency() error = %v, want ErrWorkflowDependencyCycle", err)
	}
}

func TestRemoveDependencyUnlocksChild(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo)

	projectID := uuid.New()
	managerID := uuid.New()
	parentID := uuid.New()
	childID := uuid.New()

	repo.addProjectMember(projectID, managerID, "manager")
	repo.nodes[parentID] = sqlc.WorkflowNode{ID: parentID, ProjectID: projectID, Status: NodeStatusReady, ContributionPoints: 1, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	repo.nodes[childID] = sqlc.WorkflowNode{ID: childID, ProjectID: projectID, Status: NodeStatusPlanned, ContributionPoints: 1, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	dependency, _ := repo.CreateWorkflowDependency(context.Background(), sqlc.CreateWorkflowDependencyParams{ParentNodeID: parentID, ChildNodeID: childID})
	repo.deps[dependency.ID] = dependency

	node, err := svc.RemoveDependency(context.Background(), managerID, childID, parentID)
	if err != nil {
		t.Fatalf("RemoveDependency() error = %v", err)
	}
	if node.Status != NodeStatusReady {
		t.Fatalf("expected child to become ready after removing last blocker, got %q", node.Status)
	}
	if _, exists := repo.deps[dependency.ID]; exists {
		t.Fatalf("expected dependency to be removed")
	}
}

func TestRemoveDependencyRejectsMissingLink(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo)

	projectID := uuid.New()
	managerID := uuid.New()
	parentID := uuid.New()
	childID := uuid.New()

	repo.addProjectMember(projectID, managerID, "manager")
	repo.nodes[parentID] = sqlc.WorkflowNode{ID: parentID, ProjectID: projectID, Status: NodeStatusReady, ContributionPoints: 1, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	repo.nodes[childID] = sqlc.WorkflowNode{ID: childID, ProjectID: projectID, Status: NodeStatusPlanned, ContributionPoints: 1, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}

	_, err := svc.RemoveDependency(context.Background(), managerID, childID, parentID)
	if !errors.Is(err, ErrWorkflowDependencyLinkNotFound) {
		t.Fatalf("RemoveDependency() error = %v, want ErrWorkflowDependencyLinkNotFound", err)
	}
}

func TestRemoveDependencyRequiresManagerPermissions(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo)

	projectID := uuid.New()
	teamMemberID := uuid.New()
	parentID := uuid.New()
	childID := uuid.New()

	repo.addProjectMemberWithRole(projectID, teamMemberID, "teammate", sqlc.ProjectRoleTEAM)
	repo.nodes[parentID] = sqlc.WorkflowNode{ID: parentID, ProjectID: projectID, Status: NodeStatusReady, ContributionPoints: 1, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	repo.nodes[childID] = sqlc.WorkflowNode{ID: childID, ProjectID: projectID, Status: NodeStatusPlanned, ContributionPoints: 1, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	dependency, _ := repo.CreateWorkflowDependency(context.Background(), sqlc.CreateWorkflowDependencyParams{ParentNodeID: parentID, ChildNodeID: childID})
	repo.deps[dependency.ID] = dependency

	_, err := svc.RemoveDependency(context.Background(), teamMemberID, childID, parentID)
	if !errors.Is(err, ErrWorkflowManagerRequired) {
		t.Fatalf("RemoveDependency() error = %v, want ErrWorkflowManagerRequired", err)
	}
}

func TestCompletePlannedNodeFails(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo)

	projectID := uuid.New()
	actorID := uuid.New()
	nodeID := uuid.New()

	repo.addProjectMember(projectID, actorID, "actor")
	repo.nodes[nodeID] = sqlc.WorkflowNode{
		ID:                 nodeID,
		ProjectID:          projectID,
		Status:             NodeStatusPlanned,
		ContributionPoints: 2,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}

	_, err := svc.CompleteNode(context.Background(), actorID, nodeID)
	if !errors.Is(err, ErrWorkflowNodePlanned) {
		t.Fatalf("CompleteNode() error = %v, want ErrWorkflowNodePlanned", err)
	}
}

func TestCompleteReadyNodeFails(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo)

	projectID := uuid.New()
	assigneeID := uuid.New()
	nodeID := uuid.New()

	repo.addProjectMemberWithRole(projectID, assigneeID, "assignee", sqlc.ProjectRoleTEAM)
	repo.nodes[nodeID] = sqlc.WorkflowNode{
		ID:                 nodeID,
		ProjectID:          projectID,
		Status:             NodeStatusReady,
		AssigneeUserID:     &assigneeID,
		ContributionPoints: 2,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}

	_, err := svc.CompleteNode(context.Background(), assigneeID, nodeID)
	if !errors.Is(err, ErrWorkflowNodeNotInProgress) {
		t.Fatalf("CompleteNode() error = %v, want ErrWorkflowNodeNotInProgress", err)
	}
}

func TestMultiParentUnlockBehavior(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo)

	projectID := uuid.New()
	assigneeA := uuid.New()
	assigneeB := uuid.New()
	parentA := uuid.New()
	parentB := uuid.New()
	child := uuid.New()

	repo.addProjectMember(projectID, assigneeA, "assignee-a")
	repo.addProjectMember(projectID, assigneeB, "assignee-b")
	repo.nodes[parentA] = sqlc.WorkflowNode{ID: parentA, ProjectID: projectID, Status: NodeStatusInProgress, AssigneeUserID: &assigneeA, ContributionPoints: 2, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	repo.nodes[parentB] = sqlc.WorkflowNode{ID: parentB, ProjectID: projectID, Status: NodeStatusInProgress, AssigneeUserID: &assigneeB, ContributionPoints: 2, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	repo.nodes[child] = sqlc.WorkflowNode{ID: child, ProjectID: projectID, Status: NodeStatusPlanned, ContributionPoints: 1, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	d1, _ := repo.CreateWorkflowDependency(context.Background(), sqlc.CreateWorkflowDependencyParams{ParentNodeID: parentA, ChildNodeID: child})
	d2, _ := repo.CreateWorkflowDependency(context.Background(), sqlc.CreateWorkflowDependencyParams{ParentNodeID: parentB, ChildNodeID: child})
	repo.deps[d1.ID] = d1
	repo.deps[d2.ID] = d2

	if _, err := svc.CompleteNode(context.Background(), assigneeA, parentA); err != nil {
		t.Fatalf("CompleteNode(parentA) error = %v", err)
	}
	if got := repo.nodes[child].Status; got != NodeStatusPlanned {
		t.Fatalf("expected child to stay planned after first parent, got %q", got)
	}

	if _, err := svc.CompleteNode(context.Background(), assigneeB, parentB); err != nil {
		t.Fatalf("CompleteNode(parentB) error = %v", err)
	}
	if got := repo.nodes[child].Status; got != NodeStatusReady {
		t.Fatalf("expected child to become ready after all parents done, got %q", got)
	}
}

func TestSummaryCalculation(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo)

	projectID := uuid.New()
	actorID := uuid.New()
	userA := uuid.New()
	userB := uuid.New()
	userC := uuid.New()

	repo.addProjectMember(projectID, actorID, "actor")
	repo.addProjectMember(projectID, userA, "alice")
	repo.addProjectMember(projectID, userB, "bob")
	repo.addProjectMember(projectID, userC, "carol")
	nodeDoneA := uuid.New()
	nodeDoneB := uuid.New()
	nodeAssigned := uuid.New()
	nodeUnassigned := uuid.New()
	repo.nodes[nodeDoneA] = sqlc.WorkflowNode{ID: nodeDoneA, ProjectID: projectID, Status: NodeStatusDone, AssigneeUserID: &userA, ContributionPoints: 5, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	repo.nodes[nodeDoneB] = sqlc.WorkflowNode{ID: nodeDoneB, ProjectID: projectID, Status: NodeStatusDone, AssigneeUserID: &userB, ContributionPoints: 5, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	repo.nodes[nodeAssigned] = sqlc.WorkflowNode{ID: nodeAssigned, ProjectID: projectID, Status: NodeStatusInProgress, AssigneeUserID: &userA, ContributionPoints: 10, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	repo.nodes[nodeUnassigned] = sqlc.WorkflowNode{ID: nodeUnassigned, ProjectID: projectID, Status: NodeStatusReady, ContributionPoints: 5, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	repo.ledger[uuid.New()] = sqlc.ContributionLedger{ID: uuid.New(), ProjectID: projectID, NodeID: uuid.New(), UserID: userA, Points: 5, CreatedAt: time.Now().UTC()}
	repo.ledger[uuid.New()] = sqlc.ContributionLedger{ID: uuid.New(), ProjectID: projectID, NodeID: uuid.New(), UserID: userA, Points: 3, CreatedAt: time.Now().UTC()}
	repo.ledger[uuid.New()] = sqlc.ContributionLedger{ID: uuid.New(), ProjectID: projectID, NodeID: uuid.New(), UserID: userB, Points: 2, CreatedAt: time.Now().UTC()}

	summary, err := svc.GetSummary(context.Background(), actorID, projectID)
	if err != nil {
		t.Fatalf("GetSummary() error = %v", err)
	}
	if summary.TotalProjectPoints != 25 {
		t.Fatalf("expected total project points 25, got %d", summary.TotalProjectPoints)
	}
	if summary.CompletedPoints != 10 {
		t.Fatalf("expected completed points 10, got %d", summary.CompletedPoints)
	}
	if summary.CompletionPercentage != 40 {
		t.Fatalf("expected completion percentage 40, got %v", summary.CompletionPercentage)
	}

	byUser := make(map[uuid.UUID]ProjectContributionUserSummary, len(summary.Users))
	for _, user := range summary.Users {
		byUser[user.UserID] = user
	}

	if byUser[userA].EarnedPoints != 8 || byUser[userA].CompletedNodesCount != 2 {
		t.Fatalf("unexpected userA summary: %+v", byUser[userA])
	}
	if byUser[userB].EarnedPoints != 2 || byUser[userB].CompletedNodesCount != 1 {
		t.Fatalf("unexpected userB summary: %+v", byUser[userB])
	}
	if byUser[userC].EarnedPoints != 0 || byUser[userC].CompletedNodesCount != 0 {
		t.Fatalf("unexpected userC summary: %+v", byUser[userC])
	}
	if byUser[userA].ShareOfTotalProjectPoints != 32 {
		t.Fatalf("expected userA earned share 32, got %v", byUser[userA].ShareOfTotalProjectPoints)
	}
	if byUser[userB].ShareOfTotalProjectPoints != 8 {
		t.Fatalf("expected userB earned share 8, got %v", byUser[userB].ShareOfTotalProjectPoints)
	}
	if byUser[userA].ProjectedPoints != 18 || byUser[userA].ProjectedSharePercentage != 72 {
		t.Fatalf("unexpected userA projected summary: %+v", byUser[userA])
	}
	if byUser[userB].ProjectedPoints != 2 || byUser[userB].ProjectedSharePercentage != 8 {
		t.Fatalf("unexpected userB projected summary: %+v", byUser[userB])
	}
}

func TestUpdateDoneNodeSyncsLedgerPoints(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo)

	projectID := uuid.New()
	actorID := uuid.New()
	assigneeID := uuid.New()
	nodeID := uuid.New()

	repo.addProjectMember(projectID, actorID, "actor")
	repo.addProjectMember(projectID, assigneeID, "assignee")
	repo.nodes[nodeID] = sqlc.WorkflowNode{
		ID:                 nodeID,
		ProjectID:          projectID,
		Title:              "Initial title",
		Description:        "Initial description",
		Status:             NodeStatusDone,
		AssigneeUserID:     &assigneeID,
		ContributionPoints: 3,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
	repo.ledger[nodeID] = sqlc.ContributionLedger{
		ID:        uuid.New(),
		ProjectID: projectID,
		NodeID:    nodeID,
		UserID:    assigneeID,
		Points:    3,
		CreatedAt: time.Now().UTC(),
	}

	updated, err := svc.UpdateNode(context.Background(), actorID, nodeID, UpdateNodeInput{
		Title:              "Updated title",
		Description:        "Updated description",
		ContributionPoints: 8,
	})
	if err != nil {
		t.Fatalf("UpdateNode() error = %v", err)
	}
	if updated.Title != "Updated title" || updated.ContributionPoints != 8 {
		t.Fatalf("unexpected updated node: %+v", updated)
	}
	if repo.ledger[nodeID].Points != 8 {
		t.Fatalf("expected ledger points 8, got %d", repo.ledger[nodeID].Points)
	}
}

func TestCreateNodeRequiresManagerPermissions(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo)

	projectID := uuid.New()
	teamMemberID := uuid.New()

	repo.addProjectMemberWithRole(projectID, teamMemberID, "teammate", sqlc.ProjectRoleTEAM)

	_, err := svc.CreateNode(context.Background(), teamMemberID, CreateNodeInput{
		ProjectID:          projectID,
		Title:              "Storyboard campaign page",
		ContributionPoints: 3,
	})
	if !errors.Is(err, ErrWorkflowManagerRequired) {
		t.Fatalf("CreateNode() error = %v, want ErrWorkflowManagerRequired", err)
	}
}

func TestUpdateNodeRequiresManagerPermissions(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo)

	projectID := uuid.New()
	teamMemberID := uuid.New()
	nodeID := uuid.New()

	repo.addProjectMemberWithRole(projectID, teamMemberID, "teammate", sqlc.ProjectRoleTEAM)
	repo.nodes[nodeID] = sqlc.WorkflowNode{
		ID:                 nodeID,
		ProjectID:          projectID,
		Title:              "Initial",
		Status:             NodeStatusReady,
		ContributionPoints: 2,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}

	_, err := svc.UpdateNode(context.Background(), teamMemberID, nodeID, UpdateNodeInput{
		Title:              "Updated",
		Description:        "Updated description",
		ContributionPoints: 4,
	})
	if !errors.Is(err, ErrWorkflowManagerRequired) {
		t.Fatalf("UpdateNode() error = %v, want ErrWorkflowManagerRequired", err)
	}
}

func TestAddDependencyRequiresManagerPermissions(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo)

	projectID := uuid.New()
	teamMemberID := uuid.New()
	parentID := uuid.New()
	childID := uuid.New()

	repo.addProjectMemberWithRole(projectID, teamMemberID, "teammate", sqlc.ProjectRoleTEAM)
	repo.nodes[parentID] = sqlc.WorkflowNode{ID: parentID, ProjectID: projectID, Status: NodeStatusReady, ContributionPoints: 1, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	repo.nodes[childID] = sqlc.WorkflowNode{ID: childID, ProjectID: projectID, Status: NodeStatusReady, ContributionPoints: 1, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}

	_, err := svc.AddDependency(context.Background(), teamMemberID, childID, parentID)
	if !errors.Is(err, ErrWorkflowManagerRequired) {
		t.Fatalf("AddDependency() error = %v, want ErrWorkflowManagerRequired", err)
	}
}

func TestAssignUserRejectsNonManagerAssigningAnotherUser(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo)

	projectID := uuid.New()
	actorID := uuid.New()
	assigneeID := uuid.New()
	nodeID := uuid.New()

	repo.addProjectMemberWithRole(projectID, actorID, "actor", sqlc.ProjectRoleTEAM)
	repo.addProjectMemberWithRole(projectID, assigneeID, "assignee", sqlc.ProjectRoleTEAM)
	repo.nodes[nodeID] = sqlc.WorkflowNode{
		ID:                 nodeID,
		ProjectID:          projectID,
		Status:             NodeStatusReady,
		ContributionPoints: 3,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}

	_, err := svc.AssignUser(context.Background(), actorID, nodeID, assigneeID)
	if !errors.Is(err, ErrWorkflowSelfAssignmentOnly) {
		t.Fatalf("AssignUser() error = %v, want ErrWorkflowSelfAssignmentOnly", err)
	}
}

func TestAssignUserRejectsCustomerSelfTake(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo)

	projectID := uuid.New()
	customerID := uuid.New()
	nodeID := uuid.New()

	repo.addProjectMemberWithRole(projectID, customerID, "customer", sqlc.ProjectRoleCUSTOMER)
	repo.nodes[nodeID] = sqlc.WorkflowNode{
		ID:                 nodeID,
		ProjectID:          projectID,
		Status:             NodeStatusReady,
		ContributionPoints: 2,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}

	_, err := svc.AssignUser(context.Background(), customerID, nodeID, customerID)
	if !errors.Is(err, ErrWorkflowContributorRoleRequired) {
		t.Fatalf("AssignUser() error = %v, want ErrWorkflowContributorRoleRequired", err)
	}
}

func TestAssignUserRejectsNonContributorAssignee(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo)

	projectID := uuid.New()
	managerID := uuid.New()
	customerID := uuid.New()
	nodeID := uuid.New()

	repo.addProjectMemberWithRole(projectID, managerID, "manager", sqlc.ProjectRolePM)
	repo.addProjectMemberWithRole(projectID, customerID, "customer", sqlc.ProjectRoleCUSTOMER)
	repo.nodes[nodeID] = sqlc.WorkflowNode{
		ID:                 nodeID,
		ProjectID:          projectID,
		Status:             NodeStatusReady,
		ContributionPoints: 2,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}

	_, err := svc.AssignUser(context.Background(), managerID, nodeID, customerID)
	if !errors.Is(err, ErrWorkflowAssigneeNotContributor) {
		t.Fatalf("AssignUser() error = %v, want ErrWorkflowAssigneeNotContributor", err)
	}
}

func TestCompleteNodeRequiresAssignedActor(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo)

	projectID := uuid.New()
	actorID := uuid.New()
	assigneeID := uuid.New()
	nodeID := uuid.New()

	repo.addProjectMemberWithRole(projectID, actorID, "actor", sqlc.ProjectRoleTEAM)
	repo.addProjectMemberWithRole(projectID, assigneeID, "assignee", sqlc.ProjectRoleTEAM)
	repo.nodes[nodeID] = sqlc.WorkflowNode{
		ID:                 nodeID,
		ProjectID:          projectID,
		Status:             NodeStatusInProgress,
		AssigneeUserID:     &assigneeID,
		ContributionPoints: 5,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}

	_, err := svc.CompleteNode(context.Background(), actorID, nodeID)
	if !errors.Is(err, ErrWorkflowNodeNotOwnedByActor) {
		t.Fatalf("CompleteNode() error = %v, want ErrWorkflowNodeNotOwnedByActor", err)
	}
}

func TestCompleteNodeRejectsCustomerEvenIfAssigned(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo)

	projectID := uuid.New()
	customerID := uuid.New()
	nodeID := uuid.New()

	repo.addProjectMemberWithRole(projectID, customerID, "customer", sqlc.ProjectRoleCUSTOMER)
	repo.nodes[nodeID] = sqlc.WorkflowNode{
		ID:                 nodeID,
		ProjectID:          projectID,
		Status:             NodeStatusInProgress,
		AssigneeUserID:     &customerID,
		ContributionPoints: 1,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}

	_, err := svc.CompleteNode(context.Background(), customerID, nodeID)
	if !errors.Is(err, ErrWorkflowContributorRoleRequired) {
		t.Fatalf("CompleteNode() error = %v, want ErrWorkflowContributorRoleRequired", err)
	}
}
