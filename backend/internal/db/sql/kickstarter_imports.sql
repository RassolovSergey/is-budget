-- name: CreateImportRun :one
INSERT INTO import_runs (
  id,
  project_id,
  uploaded_by,
  uploaded_by_name,
  import_type,
  file_name,
  status
)
VALUES (
  gen_random_uuid(),
  $1,
  $2,
  $3,
  $4,
  $5,
  $6
)
RETURNING id,
          project_id,
          uploaded_by,
          uploaded_by_name,
          import_type,
          file_name,
          status,
          error_message,
          rows_processed,
          rows_succeeded,
          rows_failed,
          rows_skipped,
          total_pledged,
          backer_count,
          reward_tiers_count,
          created_at,
          completed_at;

-- name: UpdateImportRunResult :one
UPDATE import_runs
SET status = $2,
    error_message = $3,
    rows_processed = $4,
    rows_succeeded = $5,
    rows_failed = $6,
    rows_skipped = $7,
    total_pledged = $8,
    backer_count = $9,
    reward_tiers_count = $10,
    completed_at = $11
WHERE id = $1
RETURNING id,
          project_id,
          uploaded_by,
          uploaded_by_name,
          import_type,
          file_name,
          status,
          error_message,
          rows_processed,
          rows_succeeded,
          rows_failed,
          rows_skipped,
          total_pledged,
          backer_count,
          reward_tiers_count,
          created_at,
          completed_at;

-- name: CreateImportRecord :one
INSERT INTO import_records (
  id,
  run_id,
  row_number,
  status,
  error_message,
  pledged_at,
  pledged_amount,
  backer_identifier,
  reward_tier
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
          run_id,
          row_number,
          status,
          error_message,
          pledged_at,
          pledged_amount,
          backer_identifier,
          reward_tier,
          created_at;

-- name: ListImportRunsByProject :many
SELECT id,
       project_id,
       uploaded_by,
       uploaded_by_name,
       import_type,
       file_name,
       status,
       error_message,
       rows_processed,
       rows_succeeded,
       rows_failed,
       rows_skipped,
       total_pledged,
       backer_count,
       reward_tiers_count,
       created_at,
       completed_at
FROM import_runs
WHERE project_id = $1
ORDER BY created_at DESC;

-- name: GetImportRunByID :one
SELECT id,
       project_id,
       uploaded_by,
       uploaded_by_name,
       import_type,
       file_name,
       status,
       error_message,
       rows_processed,
       rows_succeeded,
       rows_failed,
       rows_skipped,
       total_pledged,
       backer_count,
       reward_tiers_count,
       created_at,
       completed_at
FROM import_runs
WHERE id = $1;

-- name: ListImportRecordsByRun :many
SELECT id,
       run_id,
       row_number,
       status,
       error_message,
       pledged_at,
       pledged_amount,
       backer_identifier,
       reward_tier,
       created_at
FROM import_records
WHERE run_id = $1
ORDER BY row_number ASC;
