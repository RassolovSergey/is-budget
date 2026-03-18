ALTER TABLE historical_campaign_daily_metrics
  DROP CONSTRAINT IF EXISTS chk_historical_campaign_daily_metrics_income_nonnegative;

ALTER TABLE historical_campaign_daily_metrics
  ADD CONSTRAINT chk_historical_campaign_daily_metrics_income_nonnegative
  CHECK (income_amount >= 0);

