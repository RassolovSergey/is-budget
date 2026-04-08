ALTER TABLE payout_snapshots
ADD COLUMN settlement_status TEXT NOT NULL DEFAULT 'draft'
  CHECK (settlement_status IN ('draft', 'approved', 'ready_for_settlement')),
ADD COLUMN settlement_note TEXT,
ADD COLUMN approved_by UUID REFERENCES users(id) ON DELETE SET NULL,
ADD COLUMN approved_by_name TEXT,
ADD COLUMN approved_at TIMESTAMPTZ;

CREATE INDEX idx_payout_snapshots_project_status_created
ON payout_snapshots(project_id, settlement_status, created_at DESC);
