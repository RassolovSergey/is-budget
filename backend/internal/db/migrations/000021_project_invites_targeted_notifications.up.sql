ALTER TABLE project_invites
ADD COLUMN IF NOT EXISTS invited_user_id UUID REFERENCES users(id) ON DELETE CASCADE,
ADD COLUMN IF NOT EXISTS declined_by UUID REFERENCES users(id) ON DELETE SET NULL,
ADD COLUMN IF NOT EXISTS declined_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_project_invites_invited_user_created
ON project_invites(invited_user_id, created_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS idx_project_invites_pending_unique
ON project_invites(project_id, invited_user_id)
WHERE invited_user_id IS NOT NULL AND accepted_at IS NULL AND declined_at IS NULL;
