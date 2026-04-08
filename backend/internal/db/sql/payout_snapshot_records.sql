-- name: CreatePayoutSnapshot :one
INSERT INTO payout_snapshots (
  id,
  project_id,
  created_by,
  created_by_name,
  created_at,
  project_currency_code,
  gross_amount,
  gross_amount_source,
  final_team_pool_amount,
  total_points,
  eligible_members_count,
  member_count,
  rounding_policy,
  settlement_status,
  settlement_note,
  approved_by,
  approved_by_name,
  approved_at
)
VALUES (
  gen_random_uuid(),
  $1,
  $2,
  $3,
  $4,
  $5,
  $6,
  $7,
  $8,
  $9,
  $10,
  $11,
  $12,
  $13,
  $14,
  $15,
  $16,
  $17
)
RETURNING id,
          project_id,
          created_by,
          created_by_name,
          created_at,
          project_currency_code,
          gross_amount::double precision AS gross_amount,
          gross_amount_source,
          final_team_pool_amount::double precision AS final_team_pool_amount,
          total_points,
          eligible_members_count,
          member_count,
          rounding_policy,
          settlement_status,
          settlement_note,
          approved_by,
          approved_by_name,
          approved_at;

-- name: CreatePayoutSnapshotMember :one
INSERT INTO payout_snapshot_members (
  id,
  snapshot_id,
  user_id,
  user_name,
  total_points,
  percentage_share,
  payout_amount,
  completed_nodes_count,
  order_index
)
VALUES (
  gen_random_uuid(),
  $1,
  $2,
  $3,
  $4,
  $5,
  $6,
  $7,
  $8
)
RETURNING id,
          snapshot_id,
          user_id,
          user_name,
          total_points,
          percentage_share::double precision AS percentage_share,
          payout_amount::double precision AS payout_amount,
          completed_nodes_count,
          order_index;

-- name: CreatePayoutSnapshotBlock :one
INSERT INTO payout_snapshot_blocks (
  id,
  snapshot_id,
  source_block_id,
  name,
  type,
  level_index,
  order_index,
  value_type,
  value,
  applies_to,
  input_amount,
  allocated_amount,
  remaining_amount,
  is_team_pool
)
VALUES (
  gen_random_uuid(),
  $1,
  $2,
  $3,
  $4,
  $5,
  $6,
  $7,
  $8,
  $9,
  $10,
  $11,
  $12,
  $13
)
RETURNING id,
          snapshot_id,
          source_block_id,
          name,
          type,
          level_index,
          order_index,
          value_type,
          value::double precision AS value,
          applies_to,
          input_amount::double precision AS input_amount,
          allocated_amount::double precision AS allocated_amount,
          remaining_amount::double precision AS remaining_amount,
          is_team_pool;

-- name: ListPayoutSnapshotsByProject :many
SELECT id,
       project_id,
       created_by,
       created_by_name,
       created_at,
       project_currency_code,
       gross_amount::double precision AS gross_amount,
       gross_amount_source,
       final_team_pool_amount::double precision AS final_team_pool_amount,
       total_points,
       eligible_members_count,
       member_count,
       rounding_policy,
       settlement_status,
       settlement_note,
       approved_by,
       approved_by_name,
       approved_at
FROM payout_snapshots
WHERE project_id = $1
ORDER BY created_at DESC;

-- name: GetPayoutSnapshotByID :one
SELECT id,
       project_id,
       created_by,
       created_by_name,
       created_at,
       project_currency_code,
       gross_amount::double precision AS gross_amount,
       gross_amount_source,
       final_team_pool_amount::double precision AS final_team_pool_amount,
       total_points,
       eligible_members_count,
       member_count,
       rounding_policy,
       settlement_status,
       settlement_note,
       approved_by,
       approved_by_name,
       approved_at
FROM payout_snapshots
WHERE id = $1;

-- name: UpdatePayoutSnapshotSettlement :one
UPDATE payout_snapshots
SET settlement_status = $2,
    settlement_note = $3,
    approved_by = $4,
    approved_by_name = $5,
    approved_at = $6
WHERE id = $1
RETURNING id,
          project_id,
          created_by,
          created_by_name,
          created_at,
          project_currency_code,
          gross_amount::double precision AS gross_amount,
          gross_amount_source,
          final_team_pool_amount::double precision AS final_team_pool_amount,
          total_points,
          eligible_members_count,
          member_count,
          rounding_policy,
          settlement_status,
          settlement_note,
          approved_by,
          approved_by_name,
          approved_at;

-- name: ListPayoutSnapshotMembersBySnapshot :many
SELECT id,
       snapshot_id,
       user_id,
       user_name,
       total_points,
       percentage_share::double precision AS percentage_share,
       payout_amount::double precision AS payout_amount,
       completed_nodes_count,
       order_index
FROM payout_snapshot_members
WHERE snapshot_id = $1
ORDER BY order_index ASC;

-- name: ListPayoutSnapshotBlocksBySnapshot :many
SELECT id,
       snapshot_id,
       source_block_id,
       name,
       type,
       level_index,
       order_index,
       value_type,
       value::double precision AS value,
       applies_to,
       input_amount::double precision AS input_amount,
       allocated_amount::double precision AS allocated_amount,
       remaining_amount::double precision AS remaining_amount,
       is_team_pool
FROM payout_snapshot_blocks
WHERE snapshot_id = $1
ORDER BY level_index ASC, order_index ASC;
