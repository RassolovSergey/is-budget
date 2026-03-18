DROP INDEX IF EXISTS idx_historical_campaigns_period;

ALTER TABLE historical_campaigns
  DROP CONSTRAINT IF EXISTS chk_historical_campaign_period_valid;

ALTER TABLE historical_campaigns
  DROP COLUMN IF EXISTS started_at,
  DROP COLUMN IF EXISTS ended_at;
