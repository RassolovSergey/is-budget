ALTER TABLE projects
ADD COLUMN activated_at TIMESTAMPTZ;

UPDATE projects
SET activated_at = updated_at
WHERE status = 'active'
  AND activated_at IS NULL;
