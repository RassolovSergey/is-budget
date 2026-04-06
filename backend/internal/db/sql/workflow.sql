-- name: CreateWorkflowNode :one
INSERT INTO workflow_nodes (
  id,
  project_id,
  title,
  description,
  status,
  assignee_user_id,
  contribution_points
)
VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6)
RETURNING id, project_id, title, description, status, assignee_user_id, contribution_points, completed_at, completed_by, created_at, updated_at;

-- name: GetWorkflowNodeByID :one
SELECT id, project_id, title, description, status, assignee_user_id, contribution_points, completed_at, completed_by, created_at, updated_at
FROM workflow_nodes
WHERE id = $1;

-- name: ListWorkflowNodesByProject :many
SELECT id, project_id, title, description, status, assignee_user_id, contribution_points, completed_at, completed_by, created_at, updated_at
FROM workflow_nodes
WHERE project_id = $1
ORDER BY created_at ASC;

-- name: ListWorkflowNodesByIDs :many
SELECT id, project_id, title, description, status, assignee_user_id, contribution_points, completed_at, completed_by, created_at, updated_at
FROM workflow_nodes
WHERE id = ANY($1::uuid[]);

-- name: AssignWorkflowNodeUser :one
UPDATE workflow_nodes
SET assignee_user_id = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING id, project_id, title, description, status, assignee_user_id, contribution_points, completed_at, completed_by, created_at, updated_at;

-- name: CompleteWorkflowNode :one
UPDATE workflow_nodes
SET status = 'completed',
    completed_at = $2,
    completed_by = $3,
    updated_at = NOW()
WHERE id = $1
RETURNING id, project_id, title, description, status, assignee_user_id, contribution_points, completed_at, completed_by, created_at, updated_at;

-- name: MarkWorkflowNodeReady :one
UPDATE workflow_nodes
SET status = 'ready',
    updated_at = NOW()
WHERE id = $1
RETURNING id, project_id, title, description, status, assignee_user_id, contribution_points, completed_at, completed_by, created_at, updated_at;

-- name: CreateWorkflowDependency :one
INSERT INTO workflow_dependencies (id, parent_node_id, child_node_id)
VALUES (gen_random_uuid(), $1, $2)
RETURNING id, parent_node_id, child_node_id, created_at;

-- name: ListWorkflowDependenciesByProject :many
SELECT d.id, d.parent_node_id, d.child_node_id, d.created_at
FROM workflow_dependencies d
JOIN workflow_nodes n ON n.id = d.child_node_id
WHERE n.project_id = $1
ORDER BY d.created_at ASC;

-- name: ListWorkflowDependenciesByChild :many
SELECT id, parent_node_id, child_node_id, created_at
FROM workflow_dependencies
WHERE child_node_id = $1
ORDER BY created_at ASC;

-- name: ListWorkflowDependenciesByParent :many
SELECT id, parent_node_id, child_node_id, created_at
FROM workflow_dependencies
WHERE parent_node_id = $1
ORDER BY created_at ASC;

-- name: CreateContributionLedgerEntry :one
INSERT INTO contribution_ledger (id, project_id, node_id, user_id, points)
VALUES (gen_random_uuid(), $1, $2, $3, $4)
RETURNING id, project_id, node_id, user_id, points, created_at;

-- name: ListContributionLedgerEntriesByProject :many
SELECT id, project_id, node_id, user_id, points, created_at
FROM contribution_ledger
WHERE project_id = $1
ORDER BY created_at ASC;
