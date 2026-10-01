-- name: CreatePreferences :exec
-- The navigation takes the column's default (M3 design 4.8); the place in the sidebar is the use case's (3.18).
INSERT INTO project_user_properties (id, workspace_id, project_id, user_id, sort_order, created_by_id, updated_by_id, created_at,
                                     updated_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(project_id), sqlc.arg(user_id), sqlc.arg(sort_order), sqlc.arg(created_by),
        sqlc.arg(created_by), sqlc.arg(now), sqlc.arg(now));

-- name: LowestSortOrder :one
-- The least place of the user's in his sidebar among the workspace's projects, over his undeleted display settings,
-- the ones of projects he left too (Plane's ProjectMember.save, M3 design 3.18); no row when he has none.
SELECT sort_order
FROM project_user_properties
WHERE workspace_id = sqlc.arg(workspace_id) AND user_id = sqlc.arg(user_id) AND deleted_at IS NULL
ORDER BY sort_order
LIMIT 1;
