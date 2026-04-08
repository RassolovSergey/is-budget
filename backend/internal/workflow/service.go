package workflow

import (
	"context"
	"errors"
	"strings"
	"time"

	"isbudget/backend/internal/auth"
	"isbudget/backend/internal/db/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	NodeStatusPlanned    = "planned"
	NodeStatusReady      = "ready"
	NodeStatusInProgress = "in_progress"
	NodeStatusDone       = "done"

	// Default points stay centralized here for now so the workflow domain,
	// not the transport layer, owns the fallback behavior.
	defaultContributionPoints int32 = 1
	uniqueViolationCode             = "23505"
)

var (
	ErrWorkflowNodeNotFound                = errors.New("workflow node not found")
	ErrWorkflowProjectMemberRequired       = errors.New("workflow action requires project membership")
	ErrWorkflowManagerRequired             = errors.New("workflow action requires project manager permissions")
	ErrWorkflowContributorRoleRequired     = errors.New("workflow action requires a contributor role")
	ErrWorkflowAssigneeNotProjectMember    = errors.New("workflow assignee must be a project member")
	ErrWorkflowAssigneeNotContributor      = errors.New("workflow assignee must have a contributor role")
	ErrWorkflowNodePlanned                 = errors.New("workflow node is still planned")
	ErrWorkflowNodeNotInProgress           = errors.New("workflow node is not in progress")
	ErrWorkflowNodeAlreadyDone             = errors.New("workflow node is already done")
	ErrWorkflowNodeUnassigned              = errors.New("workflow node must be assigned before completion")
	ErrWorkflowNodeNotAssignable           = errors.New("workflow node cannot be assigned in its current state")
	ErrWorkflowNodeNotUnassignable         = errors.New("workflow node cannot be unassigned in its current state")
	ErrWorkflowSelfAssignmentOnly          = errors.New("workflow take action must assign the current user")
	ErrWorkflowNodeNotOwnedByActor         = errors.New("workflow node can only be completed by its assignee")
	ErrWorkflowDependencyNotFound          = errors.New("workflow parent node not found")
	ErrWorkflowDependencyLinkNotFound      = errors.New("workflow dependency link not found")
	ErrWorkflowDependencyProjectMismatch   = errors.New("workflow parent node must belong to the same project")
	ErrWorkflowDependencyAlreadyExists     = errors.New("workflow dependency already exists")
	ErrWorkflowDependencySelfReference     = errors.New("workflow node cannot depend on itself")
	ErrWorkflowDependencyCycle             = errors.New("workflow dependency would create a cycle")
	ErrWorkflowContributionAlreadyRecorded = errors.New("workflow contribution already recorded for node")
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

type UpdateNodeInput struct {
	Title              string
	Description        string
	ContributionPoints int32
}

type ProjectSnapshot struct {
	ProjectID          uuid.UUID                 `json:"project_id"`
	Nodes              []sqlc.WorkflowNode       `json:"nodes"`
	Dependencies       []sqlc.WorkflowDependency `json:"dependencies"`
	ContributionLedger []sqlc.ContributionLedger `json:"contribution_ledger"`
}

type ProjectContributionSummary struct {
	ProjectID            uuid.UUID                        `json:"project_id"`
	TotalProjectPoints   int32                            `json:"total_project_points"`
	CompletedPoints      int32                            `json:"completed_points"`
	CompletionPercentage float64                          `json:"completion_percentage"`
	Users                []ProjectContributionUserSummary `json:"users"`
}

type ProjectContributionUserSummary struct {
	UserID                    uuid.UUID `json:"user_id"`
	Email                     string    `json:"email"`
	FullName                  string    `json:"full_name"`
	Nickname                  string    `json:"nickname"`
	EarnedPoints              int32     `json:"earned_points"`
	ShareOfTotalProjectPoints float64   `json:"share_of_total_project_points"`
	ProjectedPoints           int32     `json:"projected_points"`
	ProjectedSharePercentage  float64   `json:"projected_share_percentage"`
	CompletedNodesCount       int32     `json:"completed_nodes_count"`
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateNode(ctx context.Context, actorID uuid.UUID, input CreateNodeInput) (sqlc.WorkflowNode, error) {
	if err := s.assertWorkflowManager(ctx, s.repo, input.ProjectID, actorID); err != nil {
		return sqlc.WorkflowNode{}, err
	}
	if hasDuplicateUUIDs(input.ParentNodeIDs) {
		return sqlc.WorkflowNode{}, ErrWorkflowDependencyAlreadyExists
	}

	parents, err := s.loadParentNodes(ctx, s.repo, input.ProjectID, input.ParentNodeIDs)
	if err != nil {
		return sqlc.WorkflowNode{}, err
	}

	var created sqlc.WorkflowNode
	err = s.repo.WithTx(ctx, func(repo Repository) error {
		node, err := repo.CreateWorkflowNode(ctx, sqlc.CreateWorkflowNodeParams{
			ProjectID:          input.ProjectID,
			Title:              strings.TrimSpace(input.Title),
			Description:        strings.TrimSpace(input.Description),
			Status:             initialNodeStatus(parents),
			AssigneeUserID:     nil,
			ContributionPoints: normalizeContributionPoints(input.ContributionPoints),
		})
		if err != nil {
			return err
		}

		for _, parentID := range input.ParentNodeIDs {
			if _, err := repo.CreateWorkflowDependency(ctx, sqlc.CreateWorkflowDependencyParams{
				ParentNodeID: parentID,
				ChildNodeID:  node.ID,
			}); err != nil {
				if isUniqueViolation(err) {
					return ErrWorkflowDependencyAlreadyExists
				}
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

func (s *Service) AddDependency(ctx context.Context, actorID, childNodeID, parentNodeID uuid.UUID) (sqlc.WorkflowNode, error) {
	child, err := s.getNode(ctx, s.repo, childNodeID)
	if err != nil {
		return sqlc.WorkflowNode{}, err
	}
	if err := s.assertWorkflowManager(ctx, s.repo, child.ProjectID, actorID); err != nil {
		return sqlc.WorkflowNode{}, err
	}
	if parentNodeID == childNodeID {
		return sqlc.WorkflowNode{}, ErrWorkflowDependencySelfReference
	}
	if child.Status == NodeStatusDone {
		return sqlc.WorkflowNode{}, ErrWorkflowNodeAlreadyDone
	}

	parent, err := s.getNode(ctx, s.repo, parentNodeID)
	if err != nil {
		if errors.Is(err, ErrWorkflowNodeNotFound) {
			return sqlc.WorkflowNode{}, ErrWorkflowDependencyNotFound
		}
		return sqlc.WorkflowNode{}, err
	}
	if parent.ProjectID != child.ProjectID {
		return sqlc.WorkflowNode{}, ErrWorkflowDependencyProjectMismatch
	}

	exists, err := s.repo.WorkflowDependencyExists(ctx, sqlc.WorkflowDependencyExistsParams{
		ParentNodeID: parentNodeID,
		ChildNodeID:  childNodeID,
	})
	if err != nil {
		return sqlc.WorkflowNode{}, err
	}
	if exists {
		return sqlc.WorkflowNode{}, ErrWorkflowDependencyAlreadyExists
	}

	cycle, err := s.repo.WorkflowPathExists(ctx, sqlc.WorkflowPathExistsParams{
		Column1: childNodeID,
		ID:      parentNodeID,
	})
	if err != nil {
		return sqlc.WorkflowNode{}, err
	}
	if cycle {
		return sqlc.WorkflowNode{}, ErrWorkflowDependencyCycle
	}

	var updated sqlc.WorkflowNode
	err = s.repo.WithTx(ctx, func(repo Repository) error {
		if _, err := repo.CreateWorkflowDependency(ctx, sqlc.CreateWorkflowDependencyParams{
			ParentNodeID: parentNodeID,
			ChildNodeID:  childNodeID,
		}); err != nil {
			if isUniqueViolation(err) {
				return ErrWorkflowDependencyAlreadyExists
			}
			return err
		}

		node, err := s.syncNodeState(ctx, repo, childNodeID)
		if err != nil {
			return err
		}
		updated = node
		return nil
	})
	if err != nil {
		return sqlc.WorkflowNode{}, err
	}

	return updated, nil
}

func (s *Service) RemoveDependency(ctx context.Context, actorID, childNodeID, parentNodeID uuid.UUID) (sqlc.WorkflowNode, error) {
	child, err := s.getNode(ctx, s.repo, childNodeID)
	if err != nil {
		return sqlc.WorkflowNode{}, err
	}
	if err := s.assertWorkflowManager(ctx, s.repo, child.ProjectID, actorID); err != nil {
		return sqlc.WorkflowNode{}, err
	}
	if parentNodeID == childNodeID {
		return sqlc.WorkflowNode{}, ErrWorkflowDependencySelfReference
	}
	if child.Status == NodeStatusDone {
		return sqlc.WorkflowNode{}, ErrWorkflowNodeAlreadyDone
	}

	parent, err := s.getNode(ctx, s.repo, parentNodeID)
	if err != nil {
		if errors.Is(err, ErrWorkflowNodeNotFound) {
			return sqlc.WorkflowNode{}, ErrWorkflowDependencyNotFound
		}
		return sqlc.WorkflowNode{}, err
	}
	if parent.ProjectID != child.ProjectID {
		return sqlc.WorkflowNode{}, ErrWorkflowDependencyProjectMismatch
	}

	var updated sqlc.WorkflowNode
	err = s.repo.WithTx(ctx, func(repo Repository) error {
		rows, err := repo.DeleteWorkflowDependency(ctx, parentNodeID, childNodeID)
		if err != nil {
			return err
		}
		if rows == 0 {
			return ErrWorkflowDependencyLinkNotFound
		}

		node, err := s.syncNodeState(ctx, repo, childNodeID)
		if err != nil {
			return err
		}
		updated = node
		return nil
	})
	if err != nil {
		return sqlc.WorkflowNode{}, err
	}

	return updated, nil
}

func (s *Service) UpdateNode(ctx context.Context, actorID, nodeID uuid.UUID, input UpdateNodeInput) (sqlc.WorkflowNode, error) {
	node, err := s.getNode(ctx, s.repo, nodeID)
	if err != nil {
		return sqlc.WorkflowNode{}, err
	}
	if err := s.assertWorkflowManager(ctx, s.repo, node.ProjectID, actorID); err != nil {
		return sqlc.WorkflowNode{}, err
	}

	var updated sqlc.WorkflowNode
	err = s.repo.WithTx(ctx, func(repo Repository) error {
		item, err := repo.UpdateWorkflowNode(ctx, sqlc.UpdateWorkflowNodeParams{
			ID:                 nodeID,
			Title:              strings.TrimSpace(input.Title),
			Description:        strings.TrimSpace(input.Description),
			ContributionPoints: normalizeContributionPoints(input.ContributionPoints),
		})
		if err != nil {
			return err
		}

		if item.Status == NodeStatusDone {
			if err := repo.UpdateContributionLedgerPointsByNode(ctx, sqlc.UpdateContributionLedgerPointsByNodeParams{
				NodeID: item.ID,
				Points: item.ContributionPoints,
			}); err != nil {
				return err
			}
		}

		updated = item
		return nil
	})
	if err != nil {
		return sqlc.WorkflowNode{}, err
	}

	return updated, nil
}

func (s *Service) AssignUser(ctx context.Context, actorID, nodeID, userID uuid.UUID) (sqlc.WorkflowNode, error) {
	node, err := s.getNode(ctx, s.repo, nodeID)
	if err != nil {
		return sqlc.WorkflowNode{}, err
	}
	actorMember, err := s.getProjectMember(ctx, s.repo, node.ProjectID, actorID)
	if err != nil {
		return sqlc.WorkflowNode{}, err
	}
	assigneeMember, err := s.getProjectMember(ctx, s.repo, node.ProjectID, userID)
	if err != nil {
		if errors.Is(err, ErrWorkflowProjectMemberRequired) {
			return sqlc.WorkflowNode{}, ErrWorkflowAssigneeNotProjectMember
		}
		return sqlc.WorkflowNode{}, err
	}
	isManager := hasProjectPermission(actorMember.Role, auth.PermProjectWrite)

	switch node.Status {
	case NodeStatusDone:
		return sqlc.WorkflowNode{}, ErrWorkflowNodeAlreadyDone
	case NodeStatusPlanned:
		if !isManager {
			return sqlc.WorkflowNode{}, ErrWorkflowNodeNotAssignable
		}
	case NodeStatusReady, NodeStatusInProgress:
	default:
		return sqlc.WorkflowNode{}, ErrWorkflowNodeNotAssignable
	}

	if !isManager {
		if actorID != userID {
			return sqlc.WorkflowNode{}, ErrWorkflowSelfAssignmentOnly
		}
		if !isWorkflowContributorRole(actorMember.Role) {
			return sqlc.WorkflowNode{}, ErrWorkflowContributorRoleRequired
		}
		if node.Status != NodeStatusReady {
			return sqlc.WorkflowNode{}, ErrWorkflowNodeNotAssignable
		}
	}
	if !isWorkflowContributorRole(assigneeMember.Role) {
		return sqlc.WorkflowNode{}, ErrWorkflowAssigneeNotContributor
	}

	switch node.Status {
	case NodeStatusReady:
		return s.repo.AssignWorkflowNodeUser(ctx, sqlc.AssignWorkflowNodeUserParams{
			ID:             nodeID,
			AssigneeUserID: &userID,
			Status:         NodeStatusInProgress,
		})
	case NodeStatusInProgress:
		return s.repo.AssignWorkflowNodeUser(ctx, sqlc.AssignWorkflowNodeUserParams{
			ID:             nodeID,
			AssigneeUserID: &userID,
			Status:         NodeStatusInProgress,
		})
	default:
		return sqlc.WorkflowNode{}, ErrWorkflowNodeNotAssignable
	}
}

func (s *Service) UnassignUser(ctx context.Context, actorID, nodeID uuid.UUID) (sqlc.WorkflowNode, error) {
	node, err := s.getNode(ctx, s.repo, nodeID)
	if err != nil {
		return sqlc.WorkflowNode{}, err
	}
	actorMember, err := s.getProjectMember(ctx, s.repo, node.ProjectID, actorID)
	if err != nil {
		return sqlc.WorkflowNode{}, err
	}
	isManager := hasProjectPermission(actorMember.Role, auth.PermProjectWrite)

	if node.Status == NodeStatusDone {
		return sqlc.WorkflowNode{}, ErrWorkflowNodeAlreadyDone
	}
	if node.Status != NodeStatusInProgress {
		return sqlc.WorkflowNode{}, ErrWorkflowNodeNotUnassignable
	}
	if node.AssigneeUserID == nil {
		return sqlc.WorkflowNode{}, ErrWorkflowNodeUnassigned
	}
	if *node.AssigneeUserID != actorID && !isManager {
		return sqlc.WorkflowNode{}, ErrWorkflowNodeNotOwnedByActor
	}

	return s.repo.AssignWorkflowNodeUser(ctx, sqlc.AssignWorkflowNodeUserParams{
		ID:             nodeID,
		AssigneeUserID: nil,
		Status:         NodeStatusReady,
	})
}

func (s *Service) CompleteNode(ctx context.Context, actorID, nodeID uuid.UUID) (sqlc.WorkflowNode, error) {
	var completed sqlc.WorkflowNode

	err := s.repo.WithTx(ctx, func(repo Repository) error {
		node, err := s.getNode(ctx, repo, nodeID)
		if err != nil {
			return err
		}
		actorMember, err := s.getProjectMember(ctx, repo, node.ProjectID, actorID)
		if err != nil {
			return err
		}
		if !isWorkflowContributorRole(actorMember.Role) {
			return ErrWorkflowContributorRoleRequired
		}

		switch node.Status {
		case NodeStatusDone:
			return ErrWorkflowNodeAlreadyDone
		case NodeStatusPlanned:
			return ErrWorkflowNodePlanned
		case NodeStatusReady:
			if node.AssigneeUserID == nil {
				return ErrWorkflowNodeUnassigned
			}
			return ErrWorkflowNodeNotInProgress
		case NodeStatusInProgress:
			if node.AssigneeUserID == nil {
				return ErrWorkflowNodeUnassigned
			}
		default:
			return ErrWorkflowNodeNotInProgress
		}
		if *node.AssigneeUserID != actorID {
			return ErrWorkflowNodeNotOwnedByActor
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
			if isUniqueViolation(err) {
				return ErrWorkflowContributionAlreadyRecorded
			}
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

func (s *Service) GetSummary(ctx context.Context, actorID, projectID uuid.UUID) (ProjectContributionSummary, error) {
	if err := s.assertProjectMember(ctx, s.repo, projectID, actorID); err != nil {
		return ProjectContributionSummary{}, err
	}

	members, err := s.repo.ListProjectMembers(ctx, projectID)
	if err != nil {
		return ProjectContributionSummary{}, err
	}
	nodes, err := s.repo.ListWorkflowNodesByProject(ctx, projectID)
	if err != nil {
		return ProjectContributionSummary{}, err
	}
	ledger, err := s.repo.ListContributionLedgerEntriesByProject(ctx, projectID)
	if err != nil {
		return ProjectContributionSummary{}, err
	}

	pointsByUser := make(map[uuid.UUID]int32, len(members))
	projectedPointsByUser := make(map[uuid.UUID]int32, len(members))
	completedByUser := make(map[uuid.UUID]int32, len(members))
	var completedPoints int32
	for _, entry := range ledger {
		pointsByUser[entry.UserID] += entry.Points
		completedByUser[entry.UserID]++
		completedPoints += entry.Points
	}

	var totalProjectPoints int32
	for _, node := range nodes {
		totalProjectPoints += node.ContributionPoints
		if node.Status == NodeStatusDone {
			continue
		}
		if node.AssigneeUserID != nil {
			projectedPointsByUser[*node.AssigneeUserID] += node.ContributionPoints
		}
	}

	users := make([]ProjectContributionUserSummary, 0, len(members))
	for _, member := range members {
		user := ProjectContributionUserSummary{
			UserID:              member.UserID,
			Email:               member.Email,
			FullName:            member.FullName,
			Nickname:            member.Nickname,
			EarnedPoints:        pointsByUser[member.UserID],
			ProjectedPoints:     pointsByUser[member.UserID] + projectedPointsByUser[member.UserID],
			CompletedNodesCount: completedByUser[member.UserID],
		}
		if totalProjectPoints > 0 {
			user.ShareOfTotalProjectPoints = (float64(user.EarnedPoints) / float64(totalProjectPoints)) * 100
			user.ProjectedSharePercentage = (float64(user.ProjectedPoints) / float64(totalProjectPoints)) * 100
		}
		users = append(users, user)
	}

	completionPercentage := 0.0
	if totalProjectPoints > 0 {
		completionPercentage = (float64(completedPoints) / float64(totalProjectPoints)) * 100
	}

	return ProjectContributionSummary{
		ProjectID:            projectID,
		TotalProjectPoints:   totalProjectPoints,
		CompletedPoints:      completedPoints,
		CompletionPercentage: completionPercentage,
		Users:                users,
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
		if child.Status == NodeStatusDone {
			continue
		}
		if _, err := s.syncNodeState(ctx, repo, child.ID); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) syncNodeState(ctx context.Context, repo Repository, nodeID uuid.UUID) (sqlc.WorkflowNode, error) {
	node, err := s.getNode(ctx, repo, nodeID)
	if err != nil {
		return sqlc.WorkflowNode{}, err
	}
	if node.Status == NodeStatusDone {
		return node, nil
	}

	parents, err := s.loadParentNodesForChild(ctx, repo, nodeID)
	if err != nil {
		return sqlc.WorkflowNode{}, err
	}

	switch desiredNodeStatus(node, parents) {
	case NodeStatusPlanned:
		return repo.MarkWorkflowNodePlanned(ctx, nodeID)
	case NodeStatusReady:
		return repo.MarkWorkflowNodeReady(ctx, nodeID)
	case NodeStatusInProgress:
		return repo.MarkWorkflowNodeInProgress(ctx, nodeID)
	default:
		return node, nil
	}
}

func initialNodeStatus(parents []sqlc.WorkflowNode) string {
	if allNodesDone(parents) {
		return NodeStatusReady
	}
	return NodeStatusPlanned
}

func desiredNodeStatus(node sqlc.WorkflowNode, parents []sqlc.WorkflowNode) string {
	if !allNodesDone(parents) {
		return NodeStatusPlanned
	}
	if node.AssigneeUserID != nil {
		return NodeStatusInProgress
	}
	return NodeStatusReady
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
	_, err := s.getProjectMember(ctx, repo, projectID, userID)
	return err
}

func (s *Service) assertWorkflowManager(ctx context.Context, repo Repository, projectID, userID uuid.UUID) error {
	member, err := s.getProjectMember(ctx, repo, projectID, userID)
	if err != nil {
		return err
	}
	if !hasProjectPermission(member.Role, auth.PermProjectWrite) {
		return ErrWorkflowManagerRequired
	}
	return nil
}

func (s *Service) getProjectMember(ctx context.Context, repo Repository, projectID, userID uuid.UUID) (sqlc.ProjectMember, error) {
	member, err := repo.GetProjectMemberByUser(ctx, sqlc.GetProjectMemberByUserParams{
		ProjectID: projectID,
		UserID:    userID,
	})
	if err == nil {
		return member, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.ProjectMember{}, ErrWorkflowProjectMemberRequired
	}
	return sqlc.ProjectMember{}, err
}

func hasProjectPermission(role sqlc.ProjectRole, permission auth.Permission) bool {
	return auth.IsAllowed(string(role), permission)
}

func isWorkflowContributorRole(role sqlc.ProjectRole) bool {
	switch role {
	case sqlc.ProjectRolePM, sqlc.ProjectRoleANALYST, sqlc.ProjectRoleTEAM:
		return true
	default:
		return false
	}
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

func allNodesDone(nodes []sqlc.WorkflowNode) bool {
	for _, node := range nodes {
		if node.Status != NodeStatusDone {
			return false
		}
	}
	return true
}

func hasDuplicateUUIDs(ids []uuid.UUID) bool {
	seen := make(map[uuid.UUID]struct{}, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			continue
		}
		if _, ok := seen[id]; ok {
			return true
		}
		seen[id] = struct{}{}
	}
	return false
}

func normalizeContributionPoints(points int32) int32 {
	if points <= 0 {
		return defaultContributionPoints
	}
	return points
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode
}
