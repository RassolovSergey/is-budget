DROP INDEX IF EXISTS idx_project_invites_pending_unique;
DROP INDEX IF EXISTS idx_project_invites_invited_user_created;

ALTER TABLE project_invites
DROP COLUMN IF EXISTS declined_at,
DROP COLUMN IF EXISTS declined_by,
DROP COLUMN IF EXISTS invited_user_id;
