CREATE TABLE campaign_configs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL UNIQUE REFERENCES projects(id) ON DELETE CASCADE,
  current_stage TEXT NOT NULL DEFAULT 'pre_launch' CHECK (current_stage IN ('pre_launch', 'live_campaign', 'post_campaign', 'closed')),
  launch_at TIMESTAMPTZ,
  late_pledges_enabled BOOLEAN NOT NULL DEFAULT FALSE,
  pledge_over_time_enabled BOOLEAN NOT NULL DEFAULT FALSE,
  pledge_manager_enabled BOOLEAN NOT NULL DEFAULT FALSE,
  digital_only_fulfillment BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE campaign_stage_snapshots (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  stage TEXT NOT NULL CHECK (stage IN ('pre_launch', 'live_campaign', 'post_campaign', 'closed')),
  created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  created_by_name TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  followers_count INTEGER CHECK (followers_count >= 0),
  notes TEXT,
  gross_amount_snapshot NUMERIC(12,2) CHECK (gross_amount_snapshot >= 0),
  team_pool_snapshot NUMERIC(12,2) CHECK (team_pool_snapshot >= 0)
);

CREATE INDEX idx_campaign_stage_snapshots_project_created
ON campaign_stage_snapshots(project_id, created_at DESC);
