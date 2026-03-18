-- name: ListTaxRulesByProject :many
SELECT id,
       project_id,
       name,
       percent::double precision AS percent,
       level,
       created_at,
       updated_at
FROM project_tax_rules
WHERE project_id = $1
ORDER BY level ASC, created_at ASC;

-- name: CreateTaxRule :one
INSERT INTO project_tax_rules (
  id, project_id, name, percent, level
)
VALUES (gen_random_uuid(), $1, $2, $3, $4)
RETURNING id,
          project_id,
          name,
          percent::double precision AS percent,
          level,
          created_at,
          updated_at;

-- name: UpdateTaxRule :one
UPDATE project_tax_rules
SET name = $3,
    percent = $4,
    level = $5,
    updated_at = NOW()
WHERE project_id = $1 AND id = $2
RETURNING id,
          project_id,
          name,
          percent::double precision AS percent,
          level,
          created_at,
          updated_at;

-- name: UpdateTaxRuleLevel :exec
UPDATE project_tax_rules
SET level = $3,
    updated_at = NOW()
WHERE project_id = $1 AND id = $2;

-- name: DeleteTaxRule :exec
DELETE FROM project_tax_rules
WHERE project_id = $1 AND id = $2;
