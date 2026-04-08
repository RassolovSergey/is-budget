package workflow

type createNodeRequest struct {
	ProjectID          string   `json:"project_id" validate:"required"`
	Title              string   `json:"title" validate:"required,min=1,max=200"`
	Description        string   `json:"description" validate:"max=1000"`
	ContributionPoints int32    `json:"contribution_points" validate:"gte=0,lte=100000"`
	ParentNodeIDs      []string `json:"parent_node_ids"`
}

type updateNodeRequest struct {
	Title              string `json:"title" validate:"required,min=1,max=200"`
	Description        string `json:"description" validate:"max=1000"`
	ContributionPoints int32  `json:"contribution_points" validate:"gte=0,lte=100000"`
}

type assignNodeRequest struct {
	UserID string `json:"user_id" validate:"required"`
}

type addDependencyRequest struct {
	ParentNodeID  string   `json:"parent_node_id"`
	ParentNodeIDs []string `json:"parent_node_ids"`
}
