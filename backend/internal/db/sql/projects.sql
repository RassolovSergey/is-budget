-- name: ListProjectsForUser :many
SELECT p.id, p.name, p.description, p.duration_days, p.status, p.activated_at, p.currency_code, p.secondary_currency_code, p.category, p.created_by, p.created_at, p.updated_at, pm.role AS current_user_role
FROM projects p
JOIN project_members pm ON pm.project_id = p.id
WHERE pm.user_id = $1
ORDER BY p.created_at DESC;

-- name: CreateProject :one
INSERT INTO projects (id, name, description, duration_days, status, activated_at, currency_code, secondary_currency_code, category, created_by)
VALUES (gen_random_uuid(), $1, $2, $3, 'preparation', NULL, $4, $5, $6, $7)
RETURNING id, name, description, duration_days, status, activated_at, currency_code, secondary_currency_code, category, created_by, created_at, updated_at;

-- name: GetProjectByID :one
SELECT id, name, description, duration_days, status, activated_at, currency_code, secondary_currency_code, category, created_by, created_at, updated_at
FROM projects
WHERE id = $1;

-- name: UpdateProject :one
UPDATE projects
SET name = $2,
    description = $3,
    duration_days = $4,
    currency_code = $5,
    secondary_currency_code = $6,
    category = $7,
    updated_at = NOW()
WHERE id = $1
RETURNING id, name, description, duration_days, status, activated_at, currency_code, secondary_currency_code, category, created_by, created_at, updated_at;

-- name: ActivateProject :one
UPDATE projects
SET status = 'active',
    activated_at = COALESCE(activated_at, NOW()),
    updated_at = NOW()
WHERE id = $1
  AND status = 'preparation'
RETURNING id, name, description, duration_days, status, activated_at, currency_code, secondary_currency_code, category, created_by, created_at, updated_at;

-- name: CompleteProjectsForUserIfDue :exec
UPDATE projects p
SET status = 'completed',
    updated_at = NOW()
WHERE p.status = 'active'
  AND p.activated_at IS NOT NULL
  AND NOW() >= p.activated_at + make_interval(days => p.duration_days)
  AND EXISTS (
    SELECT 1
    FROM project_members pm
    WHERE pm.project_id = p.id
      AND pm.user_id = $1
  );

-- name: CompleteProjectIfDue :exec
UPDATE projects
SET status = 'completed',
    updated_at = NOW()
WHERE id = $1
  AND status = 'active'
  AND activated_at IS NOT NULL
  AND NOW() >= activated_at + make_interval(days => duration_days);

-- name: CompleteAllProjectsIfDue :exec
UPDATE projects
SET status = 'completed',
    updated_at = NOW()
WHERE status = 'active'
  AND activated_at IS NOT NULL
  AND NOW() >= activated_at + make_interval(days => duration_days);

-- name: DeleteProject :exec
DELETE FROM projects
WHERE id = $1;
