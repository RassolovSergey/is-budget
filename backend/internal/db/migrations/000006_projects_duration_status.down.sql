ALTER TABLE projects
DROP CONSTRAINT IF EXISTS projects_status_check,
DROP CONSTRAINT IF EXISTS projects_duration_days_check,
DROP COLUMN IF EXISTS status,
DROP COLUMN IF EXISTS duration_days;
