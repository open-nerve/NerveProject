-- name: ListProjectViews :many
-- The project's undeleted views, the newest first, then by id (M7 P3b spec 2.4); with owned_by, only the views it owns,
-- an OwnOnly grant's list (spec 2.2). The partial index issue_views_project_id_created_at_idx serves it.
SELECT id, workspace_id, project_id, name, description, logo_props, filters, display_filters, display_properties,
       owned_by_id, created_at, updated_at
FROM issue_views
WHERE project_id = sqlc.arg(project_id)::uuid AND deleted_at IS NULL
  AND (sqlc.narg(owned_by)::uuid IS NULL OR owned_by_id = sqlc.narg(owned_by))
ORDER BY created_at DESC, id DESC;

-- name: ListWorkspaceViews :many
-- The workspace's own undeleted views, those without a project, the newest first, then by id (spec 2.4); with owned_by,
-- only the views it owns. The partial index issue_views_workspace_id_created_at_idx serves it.
SELECT id, workspace_id, project_id, name, description, logo_props, filters, display_filters, display_properties,
       owned_by_id, created_at, updated_at
FROM issue_views
WHERE workspace_id = sqlc.arg(workspace_id) AND project_id IS NULL AND deleted_at IS NULL
  AND (sqlc.narg(owned_by)::uuid IS NULL OR owned_by_id = sqlc.narg(owned_by))
ORDER BY created_at DESC, id DESC;

-- name: GetView :one
-- The undeleted view id, of either kind, read without a lock: /views/{view_id} reads it first to choose its actions
-- (spec 2.5).
SELECT id, workspace_id, project_id, name, description, logo_props, filters, display_filters, display_properties,
       owned_by_id, created_at, updated_at
FROM issue_views
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;

-- name: LockView :one
-- The undeleted view id, locked FOR UPDATE after its workspace's and project's FOR SHARE (spec 2.6): a write's read
-- under its locks.
SELECT id, workspace_id, project_id, name, description, logo_props, filters, display_filters, display_properties,
       owned_by_id, created_at, updated_at
FROM issue_views
WHERE id = sqlc.arg(id) AND deleted_at IS NULL
FOR UPDATE;

-- name: InsertView :exec
-- A new view, made by one account at one moment.
INSERT INTO issue_views (id, workspace_id, project_id, name, description, logo_props, filters, display_filters,
                         display_properties, owned_by_id, created_by_id, updated_by_id, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.narg(project_id), sqlc.arg(name), sqlc.arg(description),
        sqlc.arg(logo_props), sqlc.arg(filters), sqlc.arg(display_filters), sqlc.arg(display_properties),
        sqlc.arg(owned_by_id), sqlc.arg(by)::uuid, sqlc.arg(by)::uuid, sqlc.arg(created_at), sqlc.arg(updated_at));

-- name: UpdateView :exec
-- A view's content replaced whole, last changed by one account at one moment (spec 2.4). Its kind, owner and creation
-- are not a change's.
UPDATE issue_views
SET name               = sqlc.arg(name),
    description        = sqlc.arg(description),
    logo_props         = sqlc.arg(logo_props),
    filters            = sqlc.arg(filters),
    display_filters    = sqlc.arg(display_filters),
    display_properties = sqlc.arg(display_properties),
    updated_at         = sqlc.arg(updated_at),
    updated_by_id      = sqlc.arg(by)::uuid
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;

-- name: DeleteView :exec
-- A view soft-deleted by one account at one moment, its last change the same; one deleted already keeps its deletion.
UPDATE issue_views
SET deleted_at    = sqlc.arg(at)::timestamptz,
    updated_at    = sqlc.arg(at),
    updated_by_id = sqlc.arg(by)::uuid
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;

-- name: ViewTargets :many
-- The undeleted views among the ids asked, of either kind and any workspace, read without a lock: the views favorites
-- name (M7 P3c spec 2.4, 2.5). A deleted view and an id of none read no row. The primary key serves it.
SELECT id, workspace_id, project_id, name, description, logo_props, filters, display_filters, display_properties,
       owned_by_id, created_at, updated_at
FROM issue_views
WHERE id = ANY(sqlc.arg(ids)::uuid[]) AND deleted_at IS NULL;

-- name: ShareView :exec
-- The undeleted view a favorite names, locked FOR SHARE after its workspace's and its project's FOR SHARE (P3c spec
-- 2.8): its deletion and its change, which lock it FOR UPDATE, wait for the favorite's write, and it for them. After a
-- wait Postgres evaluates deleted_at IS NULL again on the row's newest version, so a view deleted meanwhile is not
-- locked, which the write's reads under its locks find.
SELECT id
FROM issue_views
WHERE id = sqlc.arg(id) AND deleted_at IS NULL
FOR SHARE;
