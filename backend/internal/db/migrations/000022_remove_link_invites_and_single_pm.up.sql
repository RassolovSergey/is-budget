-- Remove legacy invite-link records (without concrete recipient user).
DELETE FROM project_invites
WHERE invited_user_id IS NULL;

-- PM invitations are not allowed anymore (creator is the only PM).
UPDATE project_invites
SET invited_role = 'TEAM'::project_role
WHERE invited_role = 'PM'::project_role
  AND invited_user_id IS NOT NULL
  AND accepted_at IS NULL
  AND declined_at IS NULL;

-- Ensure project creator is always a PM member.
INSERT INTO project_members (id, project_id, user_id, role)
SELECT gen_random_uuid(), p.id, p.created_by, 'PM'::project_role
FROM projects p
LEFT JOIN project_members pm
  ON pm.project_id = p.id AND pm.user_id = p.created_by
WHERE pm.id IS NULL;

UPDATE project_members pm
SET role = 'PM'::project_role
FROM projects p
WHERE pm.project_id = p.id
  AND pm.user_id = p.created_by
  AND pm.role <> 'PM'::project_role;

-- If there were extra PM users from older logic, downgrade them to TEAM.
UPDATE project_members pm
SET role = 'TEAM'::project_role
FROM projects p
WHERE pm.project_id = p.id
  AND pm.role = 'PM'::project_role
  AND pm.user_id <> p.created_by;

-- Enforce one PM per project at DB level.
CREATE UNIQUE INDEX IF NOT EXISTS idx_project_members_single_pm_per_project
ON project_members(project_id)
WHERE role = 'PM'::project_role;
