-- The steps of deleting a workspace's projects (M3 design 3.3, 3.6), each one statement under the workspace's FOR NO
-- KEY UPDATE, which deleteWorkspace took (convention 5): the rows of the workspace not deleted before, at the moment
-- and by the account of the workspace's deletion. Rows deleted before keep their moment.

-- name: DeleteWorkspaceProjects :exec
UPDATE projects
SET deleted_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE workspace_id = sqlc.arg(workspace_id) AND deleted_at IS NULL;

-- name: DeleteWorkspaceProjectMembers :exec
-- Active memberships and ended ones alike.
UPDATE project_members
SET deleted_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE workspace_id = sqlc.arg(workspace_id) AND deleted_at IS NULL;

-- name: DeleteWorkspaceProjectPreferences :exec
UPDATE project_user_properties
SET deleted_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE workspace_id = sqlc.arg(workspace_id) AND deleted_at IS NULL;

-- name: DeleteWorkspaceStates :exec
-- The triage states too.
UPDATE states
SET deleted_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE workspace_id = sqlc.arg(workspace_id) AND deleted_at IS NULL;
