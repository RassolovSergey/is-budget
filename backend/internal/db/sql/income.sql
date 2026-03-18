-- name: ListIncomeDailyByProjectRange :many
SELECT id,
       project_id,
       income_date,
       amount::double precision AS amount,
       created_at,
       updated_at
FROM actual_income_daily
WHERE project_id = $1
  AND income_date BETWEEN sqlc.arg(from_date) AND sqlc.arg(to_date)
ORDER BY income_date ASC;

-- name: ListIncomeDailyByProject :many
SELECT id,
       project_id,
       income_date,
       amount::double precision AS amount,
       created_at,
       updated_at
FROM actual_income_daily
WHERE project_id = $1
ORDER BY income_date ASC;

-- name: UpsertIncomeDaily :one
INSERT INTO actual_income_daily (id, project_id, income_date, amount)
VALUES (gen_random_uuid(), $1, $2, $3)
ON CONFLICT (project_id, income_date)
DO UPDATE SET amount = EXCLUDED.amount,
              updated_at = NOW()
RETURNING id,
          project_id,
          income_date,
          amount::double precision AS amount,
          created_at,
          updated_at;

-- name: DeleteIncomeDaily :exec
DELETE FROM actual_income_daily
WHERE project_id = $1 AND income_date = $2;

-- name: DeleteIncomeDailyByProjectDates :exec
DELETE FROM actual_income_daily
WHERE project_id = $1
  AND income_date = ANY($2::date[]);
