CREATE TABLE workflow_nodes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL CHECK (status IN ('pending', 'ready', 'completed')),
  assignee_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  contribution_points INTEGER NOT NULL DEFAULT 1 CHECK (contribution_points >= 0),
  completed_at TIMESTAMPTZ,
  completed_by UUID REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE workflow_dependencies (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  parent_node_id UUID NOT NULL REFERENCES workflow_nodes(id) ON DELETE CASCADE,
  child_node_id UUID NOT NULL REFERENCES workflow_nodes(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT workflow_dependencies_unique UNIQUE (parent_node_id, child_node_id),
  CONSTRAINT workflow_dependencies_not_self CHECK (parent_node_id <> child_node_id)
);

CREATE TABLE contribution_ledger (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  node_id UUID NOT NULL UNIQUE REFERENCES workflow_nodes(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id),
  points INTEGER NOT NULL CHECK (points >= 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_workflow_nodes_project_id ON workflow_nodes(project_id, created_at);
CREATE INDEX idx_workflow_nodes_assignee_user_id ON workflow_nodes(assignee_user_id);
CREATE INDEX idx_workflow_dependencies_parent_node_id ON workflow_dependencies(parent_node_id);
CREATE INDEX idx_workflow_dependencies_child_node_id ON workflow_dependencies(child_node_id);
CREATE INDEX idx_contribution_ledger_project_id ON contribution_ledger(project_id, created_at);
CREATE INDEX idx_contribution_ledger_user_id ON contribution_ledger(user_id, created_at);
