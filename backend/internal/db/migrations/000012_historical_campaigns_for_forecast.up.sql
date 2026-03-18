CREATE TABLE IF NOT EXISTS historical_campaigns (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  duration_days INTEGER NOT NULL CHECK (duration_days BETWEEN 3 AND 60),
  currency_code TEXT NOT NULL DEFAULT 'EUR' CHECK (currency_code IN ('EUR', 'USD', 'RUB')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS historical_campaign_daily_metrics (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  campaign_id UUID NOT NULL REFERENCES historical_campaigns(id) ON DELETE CASCADE,
  day_number INTEGER NOT NULL CHECK (day_number >= 1 AND day_number <= 120),
  income_amount NUMERIC(12,2) NOT NULL CHECK (income_amount >= 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (campaign_id, day_number)
);

CREATE TABLE IF NOT EXISTS historical_campaign_daily_expenses (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  metric_id UUID NOT NULL REFERENCES historical_campaign_daily_metrics(id) ON DELETE CASCADE,
  category TEXT NOT NULL,
  amount NUMERIC(12,2) NOT NULL CHECK (amount >= 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_historical_campaigns_project_id
ON historical_campaigns(project_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_historical_campaign_daily_metrics_campaign_day
ON historical_campaign_daily_metrics(campaign_id, day_number);

CREATE INDEX IF NOT EXISTS idx_historical_campaign_daily_expenses_metric
ON historical_campaign_daily_expenses(metric_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_hist_campaign_expenses_metric_category_unique
ON historical_campaign_daily_expenses(metric_id, lower(category));
