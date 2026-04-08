-- name: GetCampaignConfigByProject :one
SELECT id,
       project_id,
       current_stage,
       launch_at,
       late_pledges_enabled,
       pledge_over_time_enabled,
       pledge_manager_enabled,
       digital_only_fulfillment,
       created_at,
       updated_at
FROM campaign_configs
WHERE project_id = $1;

-- name: UpsertCampaignConfig :one
INSERT INTO campaign_configs (
  id,
  project_id,
  current_stage,
  launch_at,
  late_pledges_enabled,
  pledge_over_time_enabled,
  pledge_manager_enabled,
  digital_only_fulfillment
)
VALUES (
  gen_random_uuid(),
  $1,
  $2,
  $3,
  $4,
  $5,
  $6,
  $7
)
ON CONFLICT (project_id)
DO UPDATE SET current_stage = EXCLUDED.current_stage,
              launch_at = EXCLUDED.launch_at,
              late_pledges_enabled = EXCLUDED.late_pledges_enabled,
              pledge_over_time_enabled = EXCLUDED.pledge_over_time_enabled,
              pledge_manager_enabled = EXCLUDED.pledge_manager_enabled,
              digital_only_fulfillment = EXCLUDED.digital_only_fulfillment,
              updated_at = NOW()
RETURNING id,
          project_id,
          current_stage,
          launch_at,
          late_pledges_enabled,
          pledge_over_time_enabled,
          pledge_manager_enabled,
          digital_only_fulfillment,
          created_at,
          updated_at;

-- name: CreateCampaignStageSnapshot :one
INSERT INTO campaign_stage_snapshots (
  id,
  project_id,
  stage,
  created_by,
  created_by_name,
  created_at,
  followers_count,
  notes,
  gross_amount_snapshot,
  team_pool_snapshot
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
  $9
)
RETURNING id,
          project_id,
          stage,
          created_by,
          created_by_name,
          created_at,
          followers_count,
          notes,
          gross_amount_snapshot,
          team_pool_snapshot;

-- name: ListCampaignStageSnapshotsByProject :many
SELECT id,
       project_id,
       stage,
       created_by,
       created_by_name,
       created_at,
       followers_count,
       notes,
       gross_amount_snapshot,
       team_pool_snapshot
FROM campaign_stage_snapshots
WHERE project_id = $1
ORDER BY created_at DESC;

-- name: GetCampaignStageSnapshotByID :one
SELECT id,
       project_id,
       stage,
       created_by,
       created_by_name,
       created_at,
       followers_count,
       notes,
       gross_amount_snapshot,
       team_pool_snapshot
FROM campaign_stage_snapshots
WHERE id = $1;
