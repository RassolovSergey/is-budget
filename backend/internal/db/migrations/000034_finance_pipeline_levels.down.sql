DROP INDEX IF EXISTS idx_payout_snapshot_blocks_snapshot_level_order;

CREATE INDEX idx_payout_snapshot_blocks_snapshot_order
ON payout_snapshot_blocks(snapshot_id, order_index);

ALTER TABLE payout_snapshot_blocks
  DROP COLUMN IF EXISTS level_index;

DROP INDEX IF EXISTS idx_allocation_blocks_project_level_order;

CREATE INDEX idx_allocation_blocks_project_order
ON allocation_blocks(project_id, order_index, created_at);

ALTER TABLE allocation_blocks
  DROP CONSTRAINT IF EXISTS allocation_blocks_type_check;

UPDATE allocation_blocks
SET type = CASE type
  WHEN 'tax' THEN 'tax_percent'
  WHEN 'actual_expense' THEN 'fixed_cost'
  WHEN 'custom_fee' THEN 'custom_fee'
  WHEN 'studio_distribution' THEN 'studio_percent'
  WHEN 'admin_distribution' THEN 'admin_percent'
  WHEN 'team_distribution' THEN CASE
    WHEN value_type = 'fixed' AND value = 0 THEN 'team_pool_remainder'
    ELSE 'team_pool_percent'
  END
  ELSE type
END;

ALTER TABLE allocation_blocks
  ADD CONSTRAINT allocation_blocks_type_check CHECK (
    type IN (
      'tax_percent',
      'fixed_cost',
      'custom_fee',
      'studio_percent',
      'admin_percent',
      'team_pool_percent',
      'team_pool_remainder'
    )
  );

ALTER TABLE allocation_blocks
  DROP COLUMN IF EXISTS level_index;
