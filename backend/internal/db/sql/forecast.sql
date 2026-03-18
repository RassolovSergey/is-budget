-- name: CreateForecastRun :one
INSERT INTO forecast_runs (
  id, project_id, created_at, horizon_days, model_version, input_hash, seed
)
VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6)
RETURNING id, project_id, created_at, horizon_days, model_version, input_hash, seed;

-- name: CreateForecastPoint :one
INSERT INTO forecast_points (
  id, run_id, forecast_date, yhat, p10, p90
)
VALUES (gen_random_uuid(), $1, $2, $3, $4, $5)
RETURNING id, run_id, forecast_date, yhat::double precision AS yhat, p10::double precision AS p10, p90::double precision AS p90;

-- name: GetForecastRunByID :one
SELECT id, project_id, created_at, horizon_days, model_version, input_hash, seed
FROM forecast_runs
WHERE id = $1;

-- name: GetLatestForecastRunByProject :one
SELECT id, project_id, created_at, horizon_days, model_version, input_hash, seed
FROM forecast_runs
WHERE project_id = $1
ORDER BY created_at DESC
LIMIT 1;

-- name: ListForecastPointsByRun :many
SELECT id, run_id, forecast_date, yhat::double precision AS yhat, p10::double precision AS p10, p90::double precision AS p90
FROM forecast_points
WHERE run_id = $1
ORDER BY forecast_date ASC;
