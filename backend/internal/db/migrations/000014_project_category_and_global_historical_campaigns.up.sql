ALTER TABLE projects
ADD COLUMN IF NOT EXISTS category TEXT NOT NULL DEFAULT 'Общее';

UPDATE projects
SET category = 'Общее'
WHERE TRIM(category) = '';

ALTER TABLE historical_campaigns
ADD COLUMN IF NOT EXISTS owner_user_id UUID;

ALTER TABLE historical_campaigns
ADD COLUMN IF NOT EXISTS category TEXT NOT NULL DEFAULT 'Общее';

UPDATE historical_campaigns hc
SET owner_user_id = p.created_by,
    category = COALESCE(NULLIF(TRIM(p.category), ''), 'Общее')
FROM projects p
WHERE hc.project_id = p.id
  AND hc.owner_user_id IS NULL;

UPDATE historical_campaigns
SET category = 'Общее'
WHERE TRIM(category) = '';

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'historical_campaigns_owner_user_id_fkey'
  ) THEN
    ALTER TABLE historical_campaigns
    ADD CONSTRAINT historical_campaigns_owner_user_id_fkey
      FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE CASCADE;
  END IF;
END $$;

ALTER TABLE historical_campaigns
ALTER COLUMN owner_user_id SET NOT NULL;

ALTER TABLE historical_campaigns
ALTER COLUMN project_id DROP NOT NULL;

CREATE INDEX IF NOT EXISTS idx_projects_created_by_category
ON projects(created_by, lower(category));

CREATE INDEX IF NOT EXISTS idx_historical_campaigns_owner_category
ON historical_campaigns(owner_user_id, lower(category), created_at DESC);
