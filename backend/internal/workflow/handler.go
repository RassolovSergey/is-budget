package workflow

import (
	"net/http"

	"isbudget/backend/internal/auth"
	"isbudget/backend/internal/common"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type Handler struct {
	svc      *Service
	validate *validator.Validate
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc, validate: common.NewValidator()}
}

func (h *Handler) CreateNode(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req createNodeRequest
	if !common.DecodeAndValidate(w, r, h.validate, &req) {
		return
	}

	projectID, ok := common.ParseUUIDValue(w, req.ProjectID, "project_id")
	if !ok {
		return
	}
	parentNodeIDs, ok := parseUUIDValues(w, req.ParentNodeIDs, "parent_node_ids")
	if !ok {
		return
	}

	item, err := h.svc.CreateNode(r.Context(), user.ID, CreateNodeInput{
		ProjectID:          projectID,
		Title:              req.Title,
		Description:        req.Description,
		ParentNodeIDs:      parentNodeIDs,
		ContributionPoints: req.ContributionPoints,
	})
	if err != nil {
		common.WriteMappedError(w, err, http.StatusInternalServerError, "workflow_create_failed", "failed to create workflow node",
			common.ErrorMapping{Err: ErrWorkflowProjectMemberRequired, Status: http.StatusForbidden, Code: "forbidden", Message: "workflow write access requires project membership"},
			common.ErrorMapping{Err: ErrWorkflowManagerRequired, Status: http.StatusForbidden, Code: "forbidden", Message: "only project managers can create workflow nodes"},
			common.ErrorMapping{Err: ErrWorkflowDependencyNotFound, Status: http.StatusBadRequest, Code: "invalid_request", Message: "one or more parent nodes were not found"},
			common.ErrorMapping{Err: ErrWorkflowDependencyProjectMismatch, Status: http.StatusBadRequest, Code: "invalid_request", Message: "all parent nodes must belong to the same project"},
			common.ErrorMapping{Err: ErrWorkflowDependencyAlreadyExists, Status: http.StatusBadRequest, Code: "invalid_request", Message: "duplicate parent nodes are not allowed"},
		)
		return
	}

	common.WriteJSON(w, http.StatusCreated, item)
}

func (h *Handler) UpdateNode(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	nodeID, ok := workflowNodeIDFromPath(w, r)
	if !ok {
		return
	}

	var req updateNodeRequest
	if !common.DecodeAndValidate(w, r, h.validate, &req) {
		return
	}

	item, err := h.svc.UpdateNode(r.Context(), user.ID, nodeID, UpdateNodeInput{
		Title:              req.Title,
		Description:        req.Description,
		ContributionPoints: req.ContributionPoints,
	})
	if err != nil {
		common.WriteMappedError(w, err, http.StatusInternalServerError, "workflow_update_failed", "failed to update workflow node",
			common.ErrorMapping{Err: ErrWorkflowNodeNotFound, Status: http.StatusNotFound, Code: "not_found"},
			common.ErrorMapping{Err: ErrWorkflowProjectMemberRequired, Status: http.StatusForbidden, Code: "forbidden", Message: "workflow write access requires project membership"},
			common.ErrorMapping{Err: ErrWorkflowManagerRequired, Status: http.StatusForbidden, Code: "forbidden", Message: "only project managers can edit workflow nodes"},
		)
		return
	}

	common.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) AddDependency(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	childNodeID, ok := workflowNodeIDFromPath(w, r)
	if !ok {
		return
	}

	var req addDependencyRequest
	if !common.DecodeAndValidate(w, r, h.validate, &req) {
		return
	}

	parentNodeIDs, ok := parseDependencyParentIDs(w, req)
	if !ok {
		return
	}

	var item interface{}
	for _, parentNodeID := range parentNodeIDs {
		updated, err := h.svc.AddDependency(r.Context(), user.ID, childNodeID, parentNodeID)
		if err != nil {
			common.WriteMappedError(w, err, http.StatusInternalServerError, "workflow_dependency_failed", "failed to add workflow dependency",
				common.ErrorMapping{Err: ErrWorkflowNodeNotFound, Status: http.StatusNotFound, Code: "not_found"},
				common.ErrorMapping{Err: ErrWorkflowProjectMemberRequired, Status: http.StatusForbidden, Code: "forbidden", Message: "workflow write access requires project membership"},
				common.ErrorMapping{Err: ErrWorkflowManagerRequired, Status: http.StatusForbidden, Code: "forbidden", Message: "only project managers can edit workflow dependencies"},
				common.ErrorMapping{Err: ErrWorkflowNodeAlreadyDone, Status: http.StatusBadRequest, Code: "invalid_request", Message: "completed nodes cannot change dependencies"},
				common.ErrorMapping{Err: ErrWorkflowDependencyNotFound, Status: http.StatusBadRequest, Code: "invalid_request", Message: "parent node not found for this workflow project"},
				common.ErrorMapping{Err: ErrWorkflowDependencyProjectMismatch, Status: http.StatusBadRequest, Code: "invalid_request", Message: "parent and child nodes must belong to the same project"},
				common.ErrorMapping{Err: ErrWorkflowDependencyAlreadyExists, Status: http.StatusBadRequest, Code: "invalid_request", Message: "this dependency already exists"},
				common.ErrorMapping{Err: ErrWorkflowDependencySelfReference, Status: http.StatusBadRequest, Code: "invalid_request", Message: "a node cannot depend on itself"},
				common.ErrorMapping{Err: ErrWorkflowDependencyCycle, Status: http.StatusBadRequest, Code: "invalid_request", Message: "this dependency would create a circular workflow"},
			)
			return
		}
		item = updated
	}

	common.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) RemoveDependency(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	childNodeID, ok := workflowNodeIDFromPath(w, r)
	if !ok {
		return
	}

	parentNodeID, ok := common.ParseUUIDParam(w, r, "parentId", "workflow parent node")
	if !ok {
		return
	}

	item, err := h.svc.RemoveDependency(r.Context(), user.ID, childNodeID, parentNodeID)
	if err != nil {
		common.WriteMappedError(w, err, http.StatusInternalServerError, "workflow_dependency_remove_failed", "failed to remove workflow dependency",
			common.ErrorMapping{Err: ErrWorkflowNodeNotFound, Status: http.StatusNotFound, Code: "not_found"},
			common.ErrorMapping{Err: ErrWorkflowProjectMemberRequired, Status: http.StatusForbidden, Code: "forbidden", Message: "workflow write access requires project membership"},
			common.ErrorMapping{Err: ErrWorkflowManagerRequired, Status: http.StatusForbidden, Code: "forbidden", Message: "only project managers can edit workflow dependencies"},
			common.ErrorMapping{Err: ErrWorkflowNodeAlreadyDone, Status: http.StatusBadRequest, Code: "invalid_request", Message: "completed nodes cannot change dependencies"},
			common.ErrorMapping{Err: ErrWorkflowDependencyNotFound, Status: http.StatusBadRequest, Code: "invalid_request", Message: "parent node not found for this workflow project"},
			common.ErrorMapping{Err: ErrWorkflowDependencyProjectMismatch, Status: http.StatusBadRequest, Code: "invalid_request", Message: "parent and child nodes must belong to the same project"},
			common.ErrorMapping{Err: ErrWorkflowDependencySelfReference, Status: http.StatusBadRequest, Code: "invalid_request", Message: "a node cannot remove itself as a dependency"},
			common.ErrorMapping{Err: ErrWorkflowDependencyLinkNotFound, Status: http.StatusNotFound, Code: "not_found", Message: "dependency link not found"},
		)
		return
	}

	common.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) AssignUser(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	nodeID, ok := workflowNodeIDFromPath(w, r)
	if !ok {
		return
	}

	var req assignNodeRequest
	if !common.DecodeAndValidate(w, r, h.validate, &req) {
		return
	}

	assigneeID, ok := common.ParseUUIDValue(w, req.UserID, "user_id")
	if !ok {
		return
	}

	item, err := h.svc.AssignUser(r.Context(), user.ID, nodeID, assigneeID)
	if err != nil {
		common.WriteMappedError(w, err, http.StatusInternalServerError, "workflow_assign_failed", "failed to assign workflow node",
			common.ErrorMapping{Err: ErrWorkflowNodeNotFound, Status: http.StatusNotFound, Code: "not_found"},
			common.ErrorMapping{Err: ErrWorkflowProjectMemberRequired, Status: http.StatusForbidden, Code: "forbidden", Message: "workflow write access requires project membership"},
			common.ErrorMapping{Err: ErrWorkflowAssigneeNotProjectMember, Status: http.StatusBadRequest, Code: "invalid_request", Message: "assignee must be a member of this project"},
			common.ErrorMapping{Err: ErrWorkflowAssigneeNotContributor, Status: http.StatusBadRequest, Code: "invalid_request", Message: "assignee must have a workflow contributor role"},
			common.ErrorMapping{Err: ErrWorkflowContributorRoleRequired, Status: http.StatusForbidden, Code: "forbidden", Message: "only workflow contributors can take nodes"},
			common.ErrorMapping{Err: ErrWorkflowSelfAssignmentOnly, Status: http.StatusForbidden, Code: "forbidden", Message: "only project managers can assign workflow nodes to other users"},
			common.ErrorMapping{Err: ErrWorkflowNodeAlreadyDone, Status: http.StatusBadRequest, Code: "invalid_request", Message: "completed nodes cannot be assigned again"},
			common.ErrorMapping{Err: ErrWorkflowNodeNotAssignable, Status: http.StatusBadRequest, Code: "invalid_request", Message: "this node cannot be assigned in its current state"},
		)
		return
	}

	common.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) UnassignUser(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	nodeID, ok := workflowNodeIDFromPath(w, r)
	if !ok {
		return
	}

	item, err := h.svc.UnassignUser(r.Context(), user.ID, nodeID)
	if err != nil {
		common.WriteMappedError(w, err, http.StatusInternalServerError, "workflow_unassign_failed", "failed to unassign workflow node",
			common.ErrorMapping{Err: ErrWorkflowNodeNotFound, Status: http.StatusNotFound, Code: "not_found"},
			common.ErrorMapping{Err: ErrWorkflowProjectMemberRequired, Status: http.StatusForbidden, Code: "forbidden", Message: "workflow write access requires project membership"},
			common.ErrorMapping{Err: ErrWorkflowContributorRoleRequired, Status: http.StatusForbidden, Code: "forbidden", Message: "only workflow contributors can return nodes"},
			common.ErrorMapping{Err: ErrWorkflowNodeNotOwnedByActor, Status: http.StatusForbidden, Code: "forbidden", Message: "only the assignee or a project manager can return this node"},
			common.ErrorMapping{Err: ErrWorkflowNodeAlreadyDone, Status: http.StatusBadRequest, Code: "invalid_request", Message: "completed nodes cannot be returned"},
			common.ErrorMapping{Err: ErrWorkflowNodeUnassigned, Status: http.StatusBadRequest, Code: "invalid_request", Message: "this node is not assigned right now"},
			common.ErrorMapping{Err: ErrWorkflowNodeNotUnassignable, Status: http.StatusBadRequest, Code: "invalid_request", Message: "only in-progress nodes can be returned"},
		)
		return
	}

	common.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) CompleteNode(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	nodeID, ok := workflowNodeIDFromPath(w, r)
	if !ok {
		return
	}

	item, err := h.svc.CompleteNode(r.Context(), user.ID, nodeID)
	if err != nil {
		common.WriteMappedError(w, err, http.StatusInternalServerError, "workflow_complete_failed", "failed to complete workflow node",
			common.ErrorMapping{Err: ErrWorkflowNodeNotFound, Status: http.StatusNotFound, Code: "not_found"},
			common.ErrorMapping{Err: ErrWorkflowProjectMemberRequired, Status: http.StatusForbidden, Code: "forbidden", Message: "workflow write access requires project membership"},
			common.ErrorMapping{Err: ErrWorkflowContributorRoleRequired, Status: http.StatusForbidden, Code: "forbidden", Message: "only workflow contributors can complete nodes"},
			common.ErrorMapping{Err: ErrWorkflowNodeNotOwnedByActor, Status: http.StatusForbidden, Code: "forbidden", Message: "only the assigned contributor can complete this node"},
			common.ErrorMapping{Err: ErrWorkflowNodePlanned, Status: http.StatusBadRequest, Code: "invalid_request", Message: "blocked nodes cannot be completed"},
			common.ErrorMapping{Err: ErrWorkflowNodeNotInProgress, Status: http.StatusBadRequest, Code: "invalid_request", Message: "only in-progress nodes can be completed"},
			common.ErrorMapping{Err: ErrWorkflowNodeAlreadyDone, Status: http.StatusBadRequest, Code: "invalid_request", Message: "completed nodes cannot be completed again"},
			common.ErrorMapping{Err: ErrWorkflowNodeUnassigned, Status: http.StatusBadRequest, Code: "invalid_request", Message: "an assignee is required before completion"},
			common.ErrorMapping{Err: ErrWorkflowContributionAlreadyRecorded, Status: http.StatusConflict, Code: "conflict"},
		)
		return
	}

	common.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) GetProject(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	projectID, ok := common.ParseUUIDParam(w, r, "id", "workflow project")
	if !ok {
		return
	}

	item, err := h.svc.GetProject(r.Context(), user.ID, projectID)
	if err != nil {
		common.WriteMappedError(w, err, http.StatusInternalServerError, "workflow_get_failed", "failed to load workflow project",
			common.ErrorMapping{Err: ErrWorkflowProjectMemberRequired, Status: http.StatusForbidden, Code: "forbidden"},
		)
		return
	}

	common.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) GetSummary(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	projectID, ok := common.ParseUUIDParam(w, r, "id", "workflow project")
	if !ok {
		return
	}

	item, err := h.svc.GetSummary(r.Context(), user.ID, projectID)
	if err != nil {
		common.WriteMappedError(w, err, http.StatusInternalServerError, "workflow_summary_failed", "failed to load workflow summary",
			common.ErrorMapping{Err: ErrWorkflowProjectMemberRequired, Status: http.StatusForbidden, Code: "forbidden"},
		)
		return
	}

	common.WriteJSON(w, http.StatusOK, item)
}

func workflowNodeIDFromPath(w http.ResponseWriter, r *http.Request) (id uuid.UUID, ok bool) {
	return common.ParseUUIDParam(w, r, "id", "workflow node")
}

func parseUUIDValues(w http.ResponseWriter, values []string, fieldName string) ([]uuid.UUID, bool) {
	ids := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		id, ok := common.ParseUUIDValue(w, value, fieldName)
		if !ok {
			return nil, false
		}
		ids = append(ids, id)
	}
	return ids, true
}

func parseDependencyParentIDs(w http.ResponseWriter, req addDependencyRequest) ([]uuid.UUID, bool) {
	values := make([]string, 0, len(req.ParentNodeIDs)+1)
	if req.ParentNodeID != "" {
		values = append(values, req.ParentNodeID)
	}
	values = append(values, req.ParentNodeIDs...)
	if len(values) == 0 {
		common.WriteError(w, http.StatusBadRequest, "parent_node_id or parent_node_ids is required")
		return nil, false
	}

	ids, ok := parseUUIDValues(w, values, "parent_node_ids")
	if !ok {
		return nil, false
	}
	seen := make(map[uuid.UUID]struct{}, len(ids))
	for _, id := range ids {
		if _, exists := seen[id]; exists {
			common.WriteError(w, http.StatusBadRequest, "duplicate parent_node_ids are not allowed")
			return nil, false
		}
		seen[id] = struct{}{}
	}

	return ids, true
}
