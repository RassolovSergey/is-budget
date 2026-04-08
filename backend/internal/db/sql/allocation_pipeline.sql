-- name: ListAllocationBlocksByProject :many
SELECT id,
       project_id,
       name,
       type,
       level_index,
       order_index,
       enabled,
       value_type,
       value::double precision AS value,
       applies_to,
       created_at,
       updated_at
FROM allocation_blocks
WHERE project_id = $1
ORDER BY level_index ASC, order_index ASC, created_at ASC;

-- name: GetAllocationBlockByProject :one
SELECT id,
       project_id,
       name,
       type,
       level_index,
       order_index,
       enabled,
       value_type,
       value::double precision AS value,
       applies_to,
       created_at,
       updated_at
FROM allocation_blocks
WHERE project_id = $1
  AND id = $2;

-- name: CreateAllocationBlock :one
INSERT INTO allocation_blocks (
  id,
  project_id,
  name,
  type,
  level_index,
  order_index,
  enabled,
  value_type,
  value,
  applies_to
)
VALUES (
  gen_random_uuid(),
  $1,
  $2,
  $3,
  $4,
  $5,
  $6,
  $7,
  $8,
  $9
)
RETURNING id,
          project_id,
          name,
          type,
          level_index,
          order_index,
          enabled,
          value_type,
          value::double precision AS value,
          applies_to,
          created_at,
          updated_at;

-- name: UpdateAllocationBlock :one
UPDATE allocation_blocks
SET name = $3,
    type = $4,
    level_index = $5,
    enabled = $6,
    value_type = $7,
    value = $8,
    applies_to = $9,
    updated_at = NOW()
WHERE project_id = $1
  AND id = $2
RETURNING id,
          project_id,
          name,
          type,
          level_index,
          order_index,
          enabled,
          value_type,
          value::double precision AS value,
          applies_to,
          created_at,
          updated_at;

-- name: DeleteAllocationBlock :execrows
DELETE FROM allocation_blocks
WHERE project_id = $1
  AND id = $2;

-- name: UpdateAllocationBlockPlacement :exec
UPDATE allocation_blocks
SET level_index = $3,
    order_index = $4,
    updated_at = NOW()
WHERE project_id = $1
  AND id = $2;
