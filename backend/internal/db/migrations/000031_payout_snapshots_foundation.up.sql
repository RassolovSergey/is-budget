CREATE TABLE payout_snapshots (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  created_by_name TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  project_currency_code TEXT NOT NULL CHECK (project_currency_code IN ('EUR', 'USD', 'RUB')),
  gross_amount NUMERIC(12,2) NOT NULL CHECK (gross_amount >= 0),
  gross_amount_source TEXT NOT NULL,
  final_team_pool_amount NUMERIC(12,2) NOT NULL CHECK (final_team_pool_amount >= 0),
  total_points INTEGER NOT NULL CHECK (total_points >= 0),
  eligible_members_count INTEGER NOT NULL CHECK (eligible_members_count >= 0),
  member_count INTEGER NOT NULL CHECK (member_count >= 0),
  rounding_policy TEXT NOT NULL
);

CREATE INDEX idx_payout_snapshots_project_created
ON payout_snapshots(project_id, created_at DESC);

CREATE TABLE payout_snapshot_members (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  snapshot_id UUID NOT NULL REFERENCES payout_snapshots(id) ON DELETE CASCADE,
  user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  user_name TEXT NOT NULL,
  total_points INTEGER NOT NULL CHECK (total_points >= 0),
  percentage_share NUMERIC(12,6) NOT NULL CHECK (percentage_share >= 0),
  payout_amount NUMERIC(12,2) NOT NULL CHECK (payout_amount >= 0),
  completed_nodes_count INTEGER NOT NULL DEFAULT 0 CHECK (completed_nodes_count >= 0),
  order_index INTEGER NOT NULL CHECK (order_index >= 1)
);

CREATE INDEX idx_payout_snapshot_members_snapshot_order
ON payout_snapshot_members(snapshot_id, order_index);

CREATE TABLE payout_snapshot_blocks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  snapshot_id UUID NOT NULL REFERENCES payout_snapshots(id) ON DELETE CASCADE,
  source_block_id UUID,
  name TEXT NOT NULL,
  type TEXT NOT NULL,
  order_index INTEGER NOT NULL CHECK (order_index >= 1),
  value_type TEXT NOT NULL,
  value NUMERIC(12,2) NOT NULL CHECK (value >= 0),
  applies_to TEXT NOT NULL,
  input_amount NUMERIC(12,2) NOT NULL CHECK (input_amount >= 0),
  allocated_amount NUMERIC(12,2) NOT NULL CHECK (allocated_amount >= 0),
  remaining_amount NUMERIC(12,2) NOT NULL CHECK (remaining_amount >= 0),
  is_team_pool BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX idx_payout_snapshot_blocks_snapshot_order
ON payout_snapshot_blocks(snapshot_id, order_index);
