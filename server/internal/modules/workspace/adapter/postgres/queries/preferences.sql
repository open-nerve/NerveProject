-- name: Preferences :one
-- getWorkspacePreferences: the account's undeleted row, if any (M3 design 3.18).
SELECT navigation_control_preference, navigation_project_limit
FROM workspace_user_properties
WHERE workspace_id = sqlc.arg(workspace_id) AND user_id = sqlc.arg(user_id) AND deleted_at IS NULL;

-- name: UpsertPreferences :one
-- updateWorkspacePreferences, under the workspace's FOR SHARE (M3 design 3.18): the first change inserts the row with
-- the values given, the defaults with the change applied; later ones change only the fields that are set. The
-- conflict target is the partial unique index, so a deleted row does not count.
INSERT INTO workspace_user_properties AS p (id, workspace_id, user_id, navigation_control_preference,
                                             navigation_project_limit, created_by_id, updated_by_id, created_at,
                                             updated_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(user_id), sqlc.arg(navigation_control_preference),
        sqlc.arg(navigation_project_limit), sqlc.arg(user_id), sqlc.arg(user_id), sqlc.arg(now), sqlc.arg(now))
ON CONFLICT (workspace_id, user_id) WHERE deleted_at IS NULL DO UPDATE
SET navigation_control_preference = CASE WHEN sqlc.arg(set_navigation_control)::boolean
                                         THEN EXCLUDED.navigation_control_preference
                                         ELSE p.navigation_control_preference END,
    navigation_project_limit      = CASE WHEN sqlc.arg(set_navigation_project_limit)::boolean
                                         THEN EXCLUDED.navigation_project_limit
                                         ELSE p.navigation_project_limit END,
    updated_by_id                 = EXCLUDED.updated_by_id,
    updated_at                    = EXCLUDED.updated_at
RETURNING navigation_control_preference, navigation_project_limit;
