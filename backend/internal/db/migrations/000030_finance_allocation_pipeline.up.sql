CREATE TABLE allocation_blocks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  type TEXT NOT NULL CHECK (
    type IN (
      'tax_percent',
      'fixed_cost',
      'custom_fee',
      'studio_percent',
      'admin_percent',
      'team_pool_percent',
      'team_pool_remainder'
    )
  ),
  order_index INTEGER NOT NULL CHECK (order_index >= 1),
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  value_type TEXT NOT NULL CHECK (value_type IN ('percent', 'fixed')),
  value NUMERIC(12,2) NOT NULL CHECK (value >= 0),
  applies_to TEXT NOT NULL CHECK (applies_to IN ('gross', 'current_balance')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_allocation_blocks_project_order
ON allocation_blocks(project_id, order_index, created_at);
