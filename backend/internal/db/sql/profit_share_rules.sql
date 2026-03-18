-- name: ListProfitShareRulesByProject :many
SELECT id,
       project_id,
       recipient_user_id,
       recipient_name,
       responsibility,
       currency_code,
       percent::double precision AS percent,
       level,
       applies_after_mandatory,
       created_at,
       updated_at
FROM profit_share_rules
WHERE project_id = $1
ORDER BY level ASC, created_at ASC;

-- name: CreateProfitShareRule :one
INSERT INTO profit_share_rules (
  id, project_id, recipient_user_id, recipient_name, responsibility, currency_code, percent, level, applies_after_mandatory
)
VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id,
          project_id,
          recipient_user_id,
          recipient_name,
          responsibility,
          currency_code,
          percent::double precision AS percent,
          level,
          applies_after_mandatory,
          created_at,
          updated_at;

-- name: UpdateProfitShareRule :one
UPDATE profit_share_rules
SET recipient_user_id = $3,
    recipient_name = $4,
    responsibility = $5,
    currency_code = $6,
    percent = $7,
    level = $8,
    applies_after_mandatory = $9,
    updated_at = NOW()
WHERE project_id = $1 AND id = $2
RETURNING id,
          project_id,
          recipient_user_id,
          recipient_name,
          responsibility,
          currency_code,
          percent::double precision AS percent,
          level,
          applies_after_mandatory,
          created_at,
          updated_at;

-- name: DeleteProfitShareRule :exec
DELETE FROM profit_share_rules
WHERE project_id = $1 AND id = $2;
