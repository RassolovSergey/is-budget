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
			common.ErrorMapping{Err: ErrWorkflowProjectMemberRequired, Status: http.StatusForbidden, Code: "forbidden"},
			common.ErrorMapping{Err: ErrWorkflowDependencyNotFound, Status: http.StatusBadRequest, Code: "invalid_request"},
			common.ErrorMapping{Err: ErrWorkflowDependencyProjectMismatch, Status: http.StatusBadRequest, Code: "invalid_request"},
		)
		return
	}

	common.WriteJSON(w, http.StatusCreated, item)
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
			common.ErrorMapping{Err: ErrWorkflowProjectMemberRequired, Status: http.StatusForbidden, Code: "forbidden"},
			common.ErrorMapping{Err: ErrWorkflowAssigneeNotProjectMember, Status: http.StatusBadRequest, Code: "invalid_request"},
			common.ErrorMapping{Err: ErrWorkflowNodeAlreadyCompleted, Status: http.StatusBadRequest, Code: "invalid_request"},
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
			common.ErrorMapping{Err: ErrWorkflowProjectMemberRequired, Status: http.StatusForbidden, Code: "forbidden"},
			common.ErrorMapping{Err: ErrWorkflowNodeNotReady, Status: http.StatusBadRequest, Code: "invalid_request"},
			common.ErrorMapping{Err: ErrWorkflowNodeAlreadyCompleted, Status: http.StatusBadRequest, Code: "invalid_request"},
			common.ErrorMapping{Err: ErrWorkflowNodeUnassigned, Status: http.StatusBadRequest, Code: "invalid_request"},
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
