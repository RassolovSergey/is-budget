ALTER TABLE allocation_blocks
  ADD COLUMN IF NOT EXISTS level_index INTEGER NOT NULL DEFAULT 1 CHECK (level_index >= 1);

UPDATE allocation_blocks
SET type = CASE type
  WHEN 'tax_percent' THEN 'tax'
  WHEN 'fixed_cost' THEN 'actual_expense'
  WHEN 'custom_fee' THEN 'custom_fee'
  WHEN 'studio_percent' THEN 'studio_distribution'
  WHEN 'admin_percent' THEN 'admin_distribution'
  WHEN 'team_pool_percent' THEN 'team_distribution'
  WHEN 'team_pool_remainder' THEN 'team_distribution'
  ELSE type
END;

ALTER TABLE allocation_blocks
  DROP CONSTRAINT IF EXISTS allocation_blocks_type_check;

ALTER TABLE allocation_blocks
  ADD CONSTRAINT allocation_blocks_type_check CHECK (
    type IN (
      'tax',
      'actual_expense',
      'custom_fee',
      'admin_distribution',
      'studio_distribution',
      'team_distribution'
    )
  );

DROP INDEX IF EXISTS idx_allocation_blocks_project_order;

CREATE INDEX idx_allocation_blocks_project_level_order
ON allocation_blocks(project_id, level_index, order_index, created_at);

ALTER TABLE payout_snapshot_blocks
  ADD COLUMN IF NOT EXISTS level_index INTEGER NOT NULL DEFAULT 1 CHECK (level_index >= 1);

DROP INDEX IF EXISTS idx_payout_snapshot_blocks_snapshot_order;

CREATE INDEX idx_payout_snapshot_blocks_snapshot_level_order
ON payout_snapshot_blocks(snapshot_id, level_index, order_index);
