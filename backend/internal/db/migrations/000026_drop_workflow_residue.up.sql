DROP TABLE IF EXISTS workflow_task_links;
DROP TABLE IF EXISTS workflow_tasks;

ALTER TABLE projects
DROP CONSTRAINT IF EXISTS projects_workflow_distribution_level_check,
DROP COLUMN IF EXISTS workflow_distribution_level;
