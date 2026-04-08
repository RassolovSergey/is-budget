package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"isbudget/backend/internal/auth"
	"isbudget/backend/internal/common"
	"isbudget/backend/internal/db/sqlc"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func TestCreateNodeReturnsForbiddenForNonManager(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo)
	handler := NewHandler(svc)

	projectID := uuid.New()
	teamMemberID := uuid.New()
	repo.addProjectMemberWithRole(projectID, teamMemberID, "teammate", sqlc.ProjectRoleTEAM)

	body, _ := json.Marshal(map[string]any{
		"project_id":          projectID.String(),
		"title":               "Write update",
		"contribution_points": 3,
	})
	req := httptest.NewRequest(http.MethodPost, "/workflow/node", bytes.NewReader(body))
	req = req.WithContext(auth.WithUser(req.Context(), auth.AuthUser{ID: teamMemberID}))
	rec := httptest.NewRecorder()

	handler.CreateNode(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("CreateNode() status = %d, want %d", rec.Code, http.StatusForbidden)
	}

	var payload common.ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Code != "forbidden" {
		t.Fatalf("expected forbidden code, got %q", payload.Code)
	}
}

func TestAssignUserReturnsForbiddenForNonManagerAssigningOtherUser(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo)
	handler := NewHandler(svc)

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
		ContributionPoints: 2,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}

	body, _ := json.Marshal(map[string]any{
		"user_id": assigneeID.String(),
	})
	req := httptest.NewRequest(http.MethodPost, "/workflow/node/"+nodeID.String()+"/assign", bytes.NewReader(body))
	req = req.WithContext(auth.WithUser(req.Context(), auth.AuthUser{ID: actorID}))
	rec := httptest.NewRecorder()

	router := chi.NewRouter()
	router.Post("/workflow/node/{id}/assign", handler.AssignUser)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("AssignUser() status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestCompleteNodeReturnsForbiddenForNonAssignee(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo)
	handler := NewHandler(svc)

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
		ContributionPoints: 4,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}

	req := httptest.NewRequest(http.MethodPost, "/workflow/node/"+nodeID.String()+"/complete", http.NoBody)
	req = req.WithContext(auth.WithUser(req.Context(), auth.AuthUser{ID: actorID}))
	rec := httptest.NewRecorder()

	router := chi.NewRouter()
	router.Post("/workflow/node/{id}/complete", handler.CompleteNode)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("CompleteNode() status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestRemoveDependencyReturnsNotFoundForMissingLink(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo)
	handler := NewHandler(svc)

	projectID := uuid.New()
	managerID := uuid.New()
	parentID := uuid.New()
	childID := uuid.New()

	repo.addProjectMemberWithRole(projectID, managerID, "manager", sqlc.ProjectRolePM)
	repo.nodes[parentID] = sqlc.WorkflowNode{
		ID:                 parentID,
		ProjectID:          projectID,
		Status:             NodeStatusReady,
		ContributionPoints: 1,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
	repo.nodes[childID] = sqlc.WorkflowNode{
		ID:                 childID,
		ProjectID:          projectID,
		Status:             NodeStatusPlanned,
		ContributionPoints: 1,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}

	req := httptest.NewRequest(http.MethodDelete, "/workflow/node/"+childID.String()+"/dependency/"+parentID.String(), http.NoBody)
	req = req.WithContext(auth.WithUser(req.Context(), auth.AuthUser{ID: managerID}))
	rec := httptest.NewRecorder()

	router := chi.NewRouter()
	router.Delete("/workflow/node/{id}/dependency/{parentId}", handler.RemoveDependency)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("RemoveDependency() status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestTakeBlockedNodeReturnsBadRequest(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo)
	handler := NewHandler(svc)

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

	body, _ := json.Marshal(map[string]any{
		"user_id": teamMemberID.String(),
	})
	req := httptest.NewRequest(http.MethodPost, "/workflow/node/"+nodeID.String()+"/assign", bytes.NewReader(body))
	req = req.WithContext(auth.WithUser(req.Context(), auth.AuthUser{ID: teamMemberID}))
	rec := httptest.NewRecorder()

	router := chi.NewRouter()
	router.Post("/workflow/node/{id}/assign", handler.AssignUser)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("AssignUser() status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestRemoveDependencyReturnsForbiddenForNonManager(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo)
	handler := NewHandler(svc)

	projectID := uuid.New()
	teamMemberID := uuid.New()
	parentID := uuid.New()
	childID := uuid.New()

	repo.addProjectMemberWithRole(projectID, teamMemberID, "teammate", sqlc.ProjectRoleTEAM)
	repo.nodes[parentID] = sqlc.WorkflowNode{
		ID:                 parentID,
		ProjectID:          projectID,
		Status:             NodeStatusReady,
		ContributionPoints: 1,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
	repo.nodes[childID] = sqlc.WorkflowNode{
		ID:                 childID,
		ProjectID:          projectID,
		Status:             NodeStatusPlanned,
		ContributionPoints: 1,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
	dependency, _ := repo.CreateWorkflowDependency(context.Background(), sqlc.CreateWorkflowDependencyParams{
		ParentNodeID: parentID,
		ChildNodeID:  childID,
	})
	repo.deps[dependency.ID] = dependency

	req := httptest.NewRequest(http.MethodDelete, "/workflow/node/"+childID.String()+"/dependency/"+parentID.String(), http.NoBody)
	req = req.WithContext(auth.WithUser(req.Context(), auth.AuthUser{ID: teamMemberID}))
	rec := httptest.NewRecorder()

	router := chi.NewRouter()
	router.Delete("/workflow/node/{id}/dependency/{parentId}", handler.RemoveDependency)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("RemoveDependency() status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}
