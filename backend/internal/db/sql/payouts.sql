-- name: CreatePayout :one
INSERT INTO payouts (
  id, project_id, rule_id, calculated_at, recipient_name, amount, currency_code, amount_converted, conversion_rate, status
)
VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, project_id, rule_id, calculated_at, recipient_name, amount::double precision AS amount, currency_code, amount_converted::double precision AS amount_converted, conversion_rate::double precision AS conversion_rate, status;

-- name: ListPayoutsByProject :many
SELECT id, project_id, rule_id, calculated_at, recipient_name, amount::double precision AS amount, currency_code, amount_converted::double precision AS amount_converted, conversion_rate::double precision AS conversion_rate, status
FROM payouts
WHERE project_id = $1
ORDER BY calculated_at DESC;
