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

-- name: Preferences :one
-- getProjectPreferences: the account's undeleted display settings in the project, if any (M3 design 3.18).
SELECT preferences, sort_order
FROM project_user_properties
WHERE project_id = sqlc.arg(project_id) AND user_id = sqlc.arg(user_id) AND deleted_at IS NULL;

-- name: UpsertPreferences :one
-- updateProjectPreferences, under the project's FOR SHARE (M3 design 3.6, 3.18): an account without an undeleted row
-- gets one, with the values given, the defaults with the change applied; one with a row has the fields that are set
-- changed, the navigation whole. The conflict target is the partial unique index, so a deleted row does not count.
INSERT INTO project_user_properties AS p (id, workspace_id, project_id, user_id, preferences, sort_order, created_by_id,
                                           updated_by_id, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(project_id), sqlc.arg(user_id), sqlc.arg(preferences), sqlc.arg(sort_order),
        sqlc.arg(user_id), sqlc.arg(user_id), sqlc.arg(now), sqlc.arg(now))
ON CONFLICT (project_id, user_id) WHERE deleted_at IS NULL DO UPDATE
SET preferences   = CASE WHEN sqlc.arg(set_navigation)::boolean THEN EXCLUDED.preferences ELSE p.preferences END,
    sort_order    = CASE WHEN sqlc.arg(set_sort_order)::boolean THEN EXCLUDED.sort_order ELSE p.sort_order END,
    updated_by_id = EXCLUDED.updated_by_id,
    updated_at    = EXCLUDED.updated_at
RETURNING preferences, sort_order;

-- name: EnsurePreferences :exec
-- addProjectMembers and joinProject (M3 design 3.18): the account's display settings in the project, made with the
-- place given unless he has undeleted ones there already, which a restored membership keeps as they are. The conflict
-- target is the partial unique index, so a deleted row does not count.
INSERT INTO project_user_properties (id, workspace_id, project_id, user_id, sort_order, created_by_id, updated_by_id, created_at,
                                     updated_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(project_id), sqlc.arg(user_id), sqlc.arg(sort_order), sqlc.arg(created_by),
        sqlc.arg(created_by), sqlc.arg(now), sqlc.arg(now))
ON CONFLICT (project_id, user_id) WHERE deleted_at IS NULL DO NOTHING;
