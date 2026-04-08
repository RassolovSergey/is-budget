-- name: ListProjectMembers :many
SELECT pm.id,
       pm.project_id,
       pm.user_id,
       pm.role,
       pm.created_at,
       u.email,
       u.first_name,
       u.last_name,
       u.full_name,
       u.nickname,
       u.avatar_path,
       u.default_currency_code
FROM project_members pm
JOIN users u ON u.id = pm.user_id
WHERE pm.project_id = $1
ORDER BY pm.created_at ASC;

-- name: AddProjectMember :one
INSERT INTO project_members (id, project_id, user_id, role)
VALUES (gen_random_uuid(), $1, $2, $3)
RETURNING id, project_id, user_id, role, created_at;

-- name: DeleteProjectMember :exec
DELETE FROM project_members
WHERE project_id = $1 AND id = $2;

-- name: DeleteProjectMemberByUser :exec
DELETE FROM project_members
WHERE project_id = $1 AND user_id = $2;

-- name: GetProjectMemberByUser :one
SELECT id, project_id, user_id, role, created_at
FROM project_members
WHERE project_id = $1 AND user_id = $2;

-- name: CreateProjectInvite :one
INSERT INTO project_invites (id, project_id, token, invited_role, invited_by, invited_user_id, expires_at)
VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6)
RETURNING id, project_id, token, invited_role, invited_by, invited_user_id, accepted_by, accepted_at, declined_by, declined_at, expires_at, created_at;

-- name: ListProjectInvitesWithUsersByProject :many
SELECT pi.id,
       pi.project_id,
       pi.token,
       pi.invited_role,
       pi.invited_by,
       pi.invited_user_id,
       pi.accepted_by,
       pi.accepted_at,
       pi.declined_by,
       pi.declined_at,
       pi.expires_at,
       pi.created_at,
       invited.email AS invited_user_email,
       invited.full_name AS invited_user_full_name,
       invited.nickname AS invited_user_nickname,
       inviter.full_name AS invited_by_full_name
FROM project_invites pi
LEFT JOIN users invited ON invited.id = pi.invited_user_id
LEFT JOIN users inviter ON inviter.id = pi.invited_by
WHERE pi.project_id = $1
  AND pi.invited_user_id IS NOT NULL
ORDER BY pi.created_at DESC;

-- name: GetProjectInviteByToken :one
SELECT id, project_id, token, invited_role, invited_by, invited_user_id, accepted_by, accepted_at, declined_by, declined_at, expires_at, created_at
FROM project_invites
WHERE token = $1;

-- name: MarkProjectInviteAccepted :one
UPDATE project_invites
SET accepted_by = $2,
    accepted_at = $3,
    declined_by = NULL,
    declined_at = NULL
WHERE id = $1
RETURNING id, project_id, token, invited_role, invited_by, invited_user_id, accepted_by, accepted_at, declined_by, declined_at, expires_at, created_at;

-- name: MarkProjectInviteDeclined :one
UPDATE project_invites
SET declined_by = $2,
    declined_at = $3
WHERE id = $1
RETURNING id, project_id, token, invited_role, invited_by, invited_user_id, accepted_by, accepted_at, declined_by, declined_at, expires_at, created_at;

-- name: GetPendingProjectInviteForUserInProject :one
SELECT id, project_id, token, invited_role, invited_by, invited_user_id, accepted_by, accepted_at, declined_by, declined_at, expires_at, created_at
FROM project_invites
WHERE project_id = $1
  AND invited_user_id = $2
  AND accepted_at IS NULL
  AND declined_at IS NULL
  AND expires_at > NOW()
ORDER BY created_at DESC
LIMIT 1;

-- name: ListIncomingPendingProjectInvitesForUser :many
SELECT pi.id,
       pi.project_id,
       pi.token,
       pi.invited_role,
       pi.invited_by,
       pi.invited_user_id,
       pi.accepted_by,
       pi.accepted_at,
       pi.declined_by,
       pi.declined_at,
       pi.expires_at,
       pi.created_at,
       p.name AS project_name,
       p.description AS project_description,
       p.status AS project_status,
       p.currency_code AS project_currency_code,
       inviter.full_name AS invited_by_full_name
FROM project_invites pi
JOIN projects p ON p.id = pi.project_id
LEFT JOIN users inviter ON inviter.id = pi.invited_by
WHERE pi.invited_user_id = $1
  AND pi.accepted_at IS NULL
  AND pi.declined_at IS NULL
  AND pi.expires_at > NOW()
ORDER BY pi.created_at DESC;
