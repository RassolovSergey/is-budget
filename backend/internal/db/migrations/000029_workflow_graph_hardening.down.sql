UPDATE workflow_nodes
SET status = CASE status
  WHEN 'planned' THEN 'pending'
  WHEN 'done' THEN 'completed'
  ELSE status
END
WHERE status IN ('planned', 'done');

ALTER TABLE workflow_nodes
DROP CONSTRAINT IF EXISTS workflow_nodes_status_check;

ALTER TABLE workflow_nodes
ADD CONSTRAINT workflow_nodes_status_check
CHECK (status IN ('pending', 'ready', 'completed'));

ALTER TABLE workflow_nodes
DROP CONSTRAINT IF EXISTS workflow_nodes_contribution_points_check;

ALTER TABLE workflow_nodes
ADD CONSTRAINT workflow_nodes_contribution_points_check
CHECK (contribution_points >= 0);
