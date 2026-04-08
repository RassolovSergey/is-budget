UPDATE workflow_nodes
SET status = CASE status
  WHEN 'pending' THEN 'planned'
  WHEN 'completed' THEN 'done'
  ELSE status
END
WHERE status IN ('pending', 'completed');

ALTER TABLE workflow_nodes
DROP CONSTRAINT IF EXISTS workflow_nodes_status_check;

ALTER TABLE workflow_nodes
ADD CONSTRAINT workflow_nodes_status_check
CHECK (status IN ('planned', 'ready', 'in_progress', 'done'));

ALTER TABLE workflow_nodes
DROP CONSTRAINT IF EXISTS workflow_nodes_contribution_points_check;

ALTER TABLE workflow_nodes
ADD CONSTRAINT workflow_nodes_contribution_points_check
CHECK (contribution_points > 0);
