ALTER TABLE projects
ADD COLUMN IF NOT EXISTS workflow_distribution_level INTEGER;

ALTER TABLE projects
DROP CONSTRAINT IF EXISTS projects_workflow_distribution_level_check;

ALTER TABLE projects
ADD CONSTRAINT projects_workflow_distribution_level_check
CHECK (workflow_distribution_level IS NULL OR workflow_distribution_level BETWEEN 1 AND 20);

CREATE TABLE IF NOT EXISTS workflow_tasks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  phase TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  weight NUMERIC(12,2) NOT NULL DEFAULT 1,
  assignee_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  status TEXT NOT NULL DEFAULT 'todo',
  done_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  done_at TIMESTAMPTZ,
  created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  position_x INTEGER NOT NULL DEFAULT 0,
  position_y INTEGER NOT NULL DEFAULT 0,
  CONSTRAINT workflow_tasks_status_check CHECK (status IN ('todo', 'in_progress', 'done')),
  CONSTRAINT workflow_tasks_weight_check CHECK (weight >= 0),
  CONSTRAINT workflow_tasks_position_x_check CHECK (position_x BETWEEN 0 AND 10000),
  CONSTRAINT workflow_tasks_position_y_check CHECK (position_y BETWEEN 0 AND 10000)
);

CREATE INDEX IF NOT EXISTS idx_workflow_tasks_project_status
  ON workflow_tasks(project_id, status, created_at);

CREATE INDEX IF NOT EXISTS idx_workflow_tasks_project_done
  ON workflow_tasks(project_id, done_by_user_id, status);

CREATE TABLE IF NOT EXISTS workflow_task_links (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  from_task_id UUID NOT NULL REFERENCES workflow_tasks(id) ON DELETE CASCADE,
  to_task_id UUID NOT NULL REFERENCES workflow_tasks(id) ON DELETE CASCADE,
  label TEXT NOT NULL DEFAULT '',
  created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT workflow_task_links_not_self CHECK (from_task_id <> to_task_id),
  CONSTRAINT workflow_task_links_unique UNIQUE (project_id, from_task_id, to_task_id)
);

CREATE INDEX IF NOT EXISTS idx_workflow_task_links_project
  ON workflow_task_links(project_id, created_at);

CREATE INDEX IF NOT EXISTS idx_workflow_task_links_from_to
  ON workflow_task_links(project_id, from_task_id, to_task_id);
