package workflow

import (
	"context"
	"errors"
	"strings"
	"time"

	"isbudget/backend/internal/db/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	NodeStatusPending   = "pending"
	NodeStatusReady     = "ready"
	NodeStatusCompleted = "completed"

	defaultContributionPoints int32 = 1
)

var (
	ErrWorkflowNodeNotFound              = errors.New("workflow node not found")
	ErrWorkflowProjectMemberRequired     = errors.New("workflow action requires project membership")
	ErrWorkflowAssigneeNotProjectMember  = errors.New("workflow assignee must be a project member")
	ErrWorkflowNodeNotReady              = errors.New("workflow node is not ready")
	ErrWorkflowNodeAlreadyCompleted      = errors.New("workflow node is already completed")
	ErrWorkflowNodeUnassigned            = errors.New("workflow node must be assigned before completion")
	ErrWorkflowDependencyNotFound        = errors.New("workflow parent node not found")
	ErrWorkflowDependencyProjectMismatch = errors.New("workflow parent node must belong to the same project")
)

type Service struct {
	repo Repository
}

type CreateNodeInput struct {
	ProjectID          uuid.UUID
	Title              string
	Description        string
	ParentNodeIDs      []uuid.UUID
	ContributionPoints int32
}

type ProjectSnapshot struct {
	ProjectID          uuid.UUID                 `json:"project_id"`
	Nodes              []sqlc.WorkflowNode       `json:"nodes"`
	Dependencies       []sqlc.WorkflowDependency `json:"dependencies"`
	ContributionLedger []sqlc.ContributionLedger `json:"contribution_ledger"`
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateNode(ctx context.Context, actorID uuid.UUID, input CreateNodeInput) (sqlc.WorkflowNode, error) {
	if err := s.assertProjectMember(ctx, s.repo, input.ProjectID, actorID); err != nil {
		return sqlc.WorkflowNode{}, err
	}

	parentIDs := dedupeUUIDs(input.ParentNodeIDs)
	parents, err := s.loadParentNodes(ctx, s.repo, input.ProjectID, parentIDs)
	if err != nil {
		return sqlc.WorkflowNode{}, err
	}

	status := NodeStatusReady
	if len(parents) > 0 && !allNodesCompleted(parents) {
		status = NodeStatusPending
	}

	var created sqlc.WorkflowNode
	err = s.repo.WithTx(ctx, func(repo Repository) error {
		node, err := repo.CreateWorkflowNode(ctx, sqlc.CreateWorkflowNodeParams{
			ProjectID:          input.ProjectID,
			Title:              strings.TrimSpace(input.Title),
			Description:        strings.TrimSpace(input.Description),
			Status:             status,
			AssigneeUserID:     nil,
			ContributionPoints: normalizeContributionPoints(input.ContributionPoints),
		})
		if err != nil {
			return err
		}

		for _, parentID := range parentIDs {
			if _, err := repo.CreateWorkflowDependency(ctx, sqlc.CreateWorkflowDependencyParams{
				ParentNodeID: parentID,
				ChildNodeID:  node.ID,
			}); err != nil {
				return err
			}
		}

		created = node
		return nil
	})
	if err != nil {
		return sqlc.WorkflowNode{}, err
	}

	return created, nil
}

func (s *Service) AssignUser(ctx context.Context, actorID, nodeID, userID uuid.UUID) (sqlc.WorkflowNode, error) {
	node, err := s.getNode(ctx, s.repo, nodeID)
	if err != nil {
		return sqlc.WorkflowNode{}, err
	}
	if err := s.assertProjectMember(ctx, s.repo, node.ProjectID, actorID); err != nil {
		return sqlc.WorkflowNode{}, err
	}
	if err := s.assertProjectMember(ctx, s.repo, node.ProjectID, userID); err != nil {
		if errors.Is(err, ErrWorkflowProjectMemberRequired) {
			return sqlc.WorkflowNode{}, ErrWorkflowAssigneeNotProjectMember
		}
		return sqlc.WorkflowNode{}, err
	}
	if node.Status == NodeStatusCompleted {
		return sqlc.WorkflowNode{}, ErrWorkflowNodeAlreadyCompleted
	}

	return s.repo.AssignWorkflowNodeUser(ctx, sqlc.AssignWorkflowNodeUserParams{
		ID:             nodeID,
		AssigneeUserID: &userID,
	})
}

func (s *Service) CompleteNode(ctx context.Context, actorID, nodeID uuid.UUID) (sqlc.WorkflowNode, error) {
	var completed sqlc.WorkflowNode

	err := s.repo.WithTx(ctx, func(repo Repository) error {
		node, err := s.getNode(ctx, repo, nodeID)
		if err != nil {
			return err
		}
		if err := s.assertProjectMember(ctx, repo, node.ProjectID, actorID); err != nil {
			return err
		}
		switch node.Status {
		case NodeStatusCompleted:
			return ErrWorkflowNodeAlreadyCompleted
		case NodeStatusPending:
			return ErrWorkflowNodeNotReady
		}
		if node.AssigneeUserID == nil {
			return ErrWorkflowNodeUnassigned
		}

		now := time.Now().UTC()
		updated, err := repo.CompleteWorkflowNode(ctx, sqlc.CompleteWorkflowNodeParams{
			ID:          node.ID,
			CompletedAt: &now,
			CompletedBy: &actorID,
		})
		if err != nil {
			return err
		}

		if _, err := repo.CreateContributionLedgerEntry(ctx, sqlc.CreateContributionLedgerEntryParams{
			ProjectID: node.ProjectID,
			NodeID:    node.ID,
			UserID:    *node.AssigneeUserID,
			Points:    node.ContributionPoints,
		}); err != nil {
			return err
		}

		if err := s.unlockReadyChildren(ctx, repo, node.ID); err != nil {
			return err
		}

		completed = updated
		return nil
	})
	if err != nil {
		return sqlc.WorkflowNode{}, err
	}

	return completed, nil
}

func (s *Service) GetProject(ctx context.Context, actorID, projectID uuid.UUID) (ProjectSnapshot, error) {
	if err := s.assertProjectMember(ctx, s.repo, projectID, actorID); err != nil {
		return ProjectSnapshot{}, err
	}

	nodes, err := s.repo.ListWorkflowNodesByProject(ctx, projectID)
	if err != nil {
		return ProjectSnapshot{}, err
	}
	dependencies, err := s.repo.ListWorkflowDependenciesByProject(ctx, projectID)
	if err != nil {
		return ProjectSnapshot{}, err
	}
	ledger, err := s.repo.ListContributionLedgerEntriesByProject(ctx, projectID)
	if err != nil {
		return ProjectSnapshot{}, err
	}

	return ProjectSnapshot{
		ProjectID:          projectID,
		Nodes:              nodes,
		Dependencies:       dependencies,
		ContributionLedger: ledger,
	}, nil
}

func (s *Service) unlockReadyChildren(ctx context.Context, repo Repository, parentNodeID uuid.UUID) error {
	dependencies, err := repo.ListWorkflowDependenciesByParent(ctx, parentNodeID)
	if err != nil {
		return err
	}

	for _, dependency := range dependencies {
		child, err := s.getNode(ctx, repo, dependency.ChildNodeID)
		if err != nil {
			return err
		}
		if child.Status != NodeStatusPending {
			continue
		}

		parents, err := s.loadParentNodesForChild(ctx, repo, child.ID)
		if err != nil {
			return err
		}
		if allNodesCompleted(parents) {
			if _, err := repo.MarkWorkflowNodeReady(ctx, child.ID); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Service) loadParentNodes(ctx context.Context, repo Repository, projectID uuid.UUID, parentIDs []uuid.UUID) ([]sqlc.WorkflowNode, error) {
	if len(parentIDs) == 0 {
		return nil, nil
	}

	nodes, err := repo.ListWorkflowNodesByIDs(ctx, parentIDs)
	if err != nil {
		return nil, err
	}
	if len(nodes) != len(parentIDs) {
		return nil, ErrWorkflowDependencyNotFound
	}
	for _, node := range nodes {
		if projectID != uuid.Nil && node.ProjectID != projectID {
			return nil, ErrWorkflowDependencyProjectMismatch
		}
	}
	return nodes, nil
}

func (s *Service) loadParentNodesForChild(ctx context.Context, repo Repository, childNodeID uuid.UUID) ([]sqlc.WorkflowNode, error) {
	dependencies, err := repo.ListWorkflowDependenciesByChild(ctx, childNodeID)
	if err != nil {
		return nil, err
	}
	parentIDs := make([]uuid.UUID, 0, len(dependencies))
	for _, dependency := range dependencies {
		parentIDs = append(parentIDs, dependency.ParentNodeID)
	}
	return s.loadParentNodes(ctx, repo, uuid.Nil, parentIDs)
}

func (s *Service) assertProjectMember(ctx context.Context, repo Repository, projectID, userID uuid.UUID) error {
	_, err := repo.GetProjectMemberByUser(ctx, sqlc.GetProjectMemberByUserParams{
		ProjectID: projectID,
		UserID:    userID,
	})
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrWorkflowProjectMemberRequired
	}
	return err
}

func (s *Service) getNode(ctx context.Context, repo Repository, nodeID uuid.UUID) (sqlc.WorkflowNode, error) {
	node, err := repo.GetWorkflowNodeByID(ctx, nodeID)
	if err == nil {
		return node, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.WorkflowNode{}, ErrWorkflowNodeNotFound
	}
	return sqlc.WorkflowNode{}, err
}

func allNodesCompleted(nodes []sqlc.WorkflowNode) bool {
	for _, node := range nodes {
		if node.Status != NodeStatusCompleted {
			return false
		}
	}
	return true
}

func dedupeUUIDs(ids []uuid.UUID) []uuid.UUID {
	if len(ids) == 0 {
		return nil
	}

	seen := make(map[uuid.UUID]struct{}, len(ids))
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func normalizeContributionPoints(points int32) int32 {
	if points == 0 {
		return defaultContributionPoints
	}
	return points
}
