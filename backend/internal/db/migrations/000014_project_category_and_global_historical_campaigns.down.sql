DROP INDEX IF EXISTS idx_historical_campaigns_owner_category;
DROP INDEX IF EXISTS idx_projects_created_by_category;

ALTER TABLE historical_campaigns
DROP CONSTRAINT IF EXISTS historical_campaigns_owner_user_id_fkey;

DELETE FROM historical_campaigns
WHERE project_id IS NULL;

ALTER TABLE historical_campaigns
ALTER COLUMN project_id SET NOT NULL;

ALTER TABLE historical_campaigns
DROP COLUMN IF EXISTS owner_user_id;

ALTER TABLE historical_campaigns
DROP COLUMN IF EXISTS category;

ALTER TABLE projects
DROP COLUMN IF EXISTS category;
