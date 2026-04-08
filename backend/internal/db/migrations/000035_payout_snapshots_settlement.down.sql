DROP INDEX IF EXISTS idx_payout_snapshots_project_status_created;

ALTER TABLE payout_snapshots
DROP COLUMN IF EXISTS approved_at,
DROP COLUMN IF EXISTS approved_by_name,
DROP COLUMN IF EXISTS approved_by,
DROP COLUMN IF EXISTS settlement_note,
DROP COLUMN IF EXISTS settlement_status;
