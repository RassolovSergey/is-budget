ALTER TABLE forecast_runs
  DROP CONSTRAINT IF EXISTS forecast_runs_horizon_days_check;

ALTER TABLE forecast_runs
  ADD CONSTRAINT forecast_runs_horizon_days_check
  CHECK (horizon_days BETWEEN 1 AND 30);
