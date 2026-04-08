CREATE TABLE import_runs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  uploaded_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  uploaded_by_name TEXT NOT NULL,
  import_type TEXT NOT NULL DEFAULT 'kickstarter_csv',
  file_name TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('pending', 'success', 'failed')),
  error_message TEXT,
  rows_processed INTEGER NOT NULL DEFAULT 0,
  rows_succeeded INTEGER NOT NULL DEFAULT 0,
  rows_failed INTEGER NOT NULL DEFAULT 0,
  rows_skipped INTEGER NOT NULL DEFAULT 0,
  total_pledged NUMERIC(14,2),
  backer_count INTEGER,
  reward_tiers_count INTEGER,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  completed_at TIMESTAMPTZ
);

CREATE INDEX idx_import_runs_project_created
ON import_runs(project_id, created_at DESC);

CREATE TABLE import_records (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  run_id UUID NOT NULL REFERENCES import_runs(id) ON DELETE CASCADE,
  row_number INTEGER NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('success', 'failed', 'skipped')),
  error_message TEXT,
  pledged_at DATE,
  pledged_amount NUMERIC(14,2),
  backer_identifier TEXT,
  reward_tier TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (run_id, row_number)
);

CREATE INDEX idx_import_records_run
ON import_records(run_id, row_number ASC);
