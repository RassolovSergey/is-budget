-- name: GetCurrencyRate :one
SELECT from_currency_code,
       to_currency_code,
       rate::double precision AS rate,
       source,
       updated_at
FROM currency_rates
WHERE from_currency_code = $1
  AND to_currency_code = $2;

-- name: UpsertCurrencyRate :one
INSERT INTO currency_rates (from_currency_code, to_currency_code, rate, source, updated_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (from_currency_code, to_currency_code)
DO UPDATE SET rate = EXCLUDED.rate,
              source = EXCLUDED.source,
              updated_at = EXCLUDED.updated_at
RETURNING from_currency_code,
          to_currency_code,
          rate::double precision AS rate,
          source,
          updated_at;
