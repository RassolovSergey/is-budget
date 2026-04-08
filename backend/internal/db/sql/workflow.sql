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
SET assignee_user_id = sqlc.arg(assignee_user_id),
    status = sqlc.arg(status),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING id, project_id, title, description, status, assignee_user_id, contribution_points, completed_at, completed_by, created_at, updated_at;

-- name: UpdateWorkflowNode :one
UPDATE workflow_nodes
SET title = sqlc.arg(title),
    description = sqlc.arg(description),
    contribution_points = sqlc.arg(contribution_points),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING id, project_id, title, description, status, assignee_user_id, contribution_points, completed_at, completed_by, created_at, updated_at;

-- name: CompleteWorkflowNode :one
UPDATE workflow_nodes
SET status = 'done',
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

-- name: MarkWorkflowNodePlanned :one
UPDATE workflow_nodes
SET status = 'planned',
    updated_at = NOW()
WHERE id = $1
RETURNING id, project_id, title, description, status, assignee_user_id, contribution_points, completed_at, completed_by, created_at, updated_at;

-- name: MarkWorkflowNodeInProgress :one
UPDATE workflow_nodes
SET status = 'in_progress',
    updated_at = NOW()
WHERE id = $1
RETURNING id, project_id, title, description, status, assignee_user_id, contribution_points, completed_at, completed_by, created_at, updated_at;

-- name: CreateWorkflowDependency :one
INSERT INTO workflow_dependencies (id, parent_node_id, child_node_id)
VALUES (gen_random_uuid(), $1, $2)
RETURNING id, parent_node_id, child_node_id, created_at;

-- name: DeleteWorkflowDependency :execrows
DELETE FROM workflow_dependencies
WHERE parent_node_id = $1
  AND child_node_id = $2;

-- name: WorkflowDependencyExists :one
SELECT EXISTS (
  SELECT 1
  FROM workflow_dependencies
  WHERE parent_node_id = sqlc.arg(parent_node_id)
    AND child_node_id = sqlc.arg(child_node_id)
);

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

-- name: WorkflowPathExists :one
WITH RECURSIVE reachable(id) AS (
  SELECT $1::uuid
  UNION
  SELECT d.child_node_id
  FROM workflow_dependencies d
  JOIN reachable r ON d.parent_node_id = r.id
)
SELECT EXISTS (
  SELECT 1
  FROM reachable
  WHERE id = $2
);

-- name: CreateContributionLedgerEntry :one
INSERT INTO contribution_ledger (id, project_id, node_id, user_id, points)
VALUES (gen_random_uuid(), $1, $2, $3, $4)
RETURNING id, project_id, node_id, user_id, points, created_at;

-- name: UpdateContributionLedgerPointsByNode :exec
UPDATE contribution_ledger
SET points = sqlc.arg(points)
WHERE node_id = sqlc.arg(node_id);

-- name: ListContributionLedgerEntriesByProject :many
SELECT id, project_id, node_id, user_id, points, created_at
FROM contribution_ledger
WHERE project_id = $1
ORDER BY created_at ASC;
