-- name: NavigationProjectFacts :one
-- What project.Provide offers the navigation module (M7 P3b spec 2.1), read without a lock: the undeleted project's
-- workspace and its views' toggle, an archived project's as ever (R2). A deleted project reads no row.
SELECT workspace_id, issue_views_view
FROM projects
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;

-- name: ShareNavigationProject :one
-- NavigationProjectFacts under the project's lock of a view's write (spec 2.6): the project's row FOR SHARE, after the
-- write's FOR SHARE of its workspace, which this does not lock. It waits for the FOR NO KEY UPDATE of a change of the
-- toggle, of the project's members, of an archiving and of a deletion, and they for it. After a wait Postgres evaluates
-- deleted_at IS NULL again on the row's newest version, so a project deleted meanwhile reads no row.
SELECT workspace_id, issue_views_view
FROM projects
WHERE id = sqlc.arg(id) AND deleted_at IS NULL
FOR SHARE;

-- name: NavigationTargets :many
-- The projects favorites name (M7 P3c spec 2.4, 2.5), read by the ids asked, without a lock: each undeleted one's
-- workspace, name, icon and whether it is archived, another workspace's too. A deleted project and an id of none read
-- no row. The primary key serves it.
SELECT id, workspace_id, name, logo_props, (archived_at IS NOT NULL)::boolean AS archived
FROM projects
WHERE id = ANY(sqlc.arg(ids)::uuid[]) AND deleted_at IS NULL;
