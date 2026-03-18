ALTER TABLE historical_campaigns
  ADD COLUMN started_at DATE,
  ADD COLUMN ended_at DATE;

UPDATE historical_campaigns
SET
  started_at = COALESCE(started_at, created_at::date),
  ended_at = COALESCE(ended_at, (created_at::date + (duration_days - 1)));

ALTER TABLE historical_campaigns
  ALTER COLUMN started_at SET NOT NULL,
  ALTER COLUMN ended_at SET NOT NULL;

ALTER TABLE historical_campaigns
  ADD CONSTRAINT chk_historical_campaign_period_valid
  CHECK (ended_at >= started_at);

CREATE INDEX IF NOT EXISTS idx_historical_campaigns_period
ON historical_campaigns(project_id, started_at DESC, ended_at DESC);
