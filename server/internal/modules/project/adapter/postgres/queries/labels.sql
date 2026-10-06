-- name: CreateLabel :one
-- createLabel, under the project's FOR NO KEY UPDATE (M3 design 3.16): the row as stored.
INSERT INTO labels (id, workspace_id, project_id, parent_id, name, color, sort_order, created_by_id, updated_by_id, created_at,
                    updated_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(project_id), sqlc.narg(parent_id), sqlc.arg(name), sqlc.arg(color),
        sqlc.arg(sort_order), sqlc.arg(created_by), sqlc.arg(created_by), sqlc.arg(now), sqlc.arg(now))
RETURNING id, workspace_id, project_id, parent_id, name, color, sort_order, created_at, updated_at;

-- name: LabelByID :one
-- A write on a label named by its id (M3 design 3.6 convention 2, 6.7), read first without a lock for its project and
-- the project's workspace, then again under their locks; and a label named as a parent, under the project's lock: the
-- undeleted label.
SELECT id, workspace_id, project_id, parent_id, name, color, sort_order, created_at, updated_at
FROM labels
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;

-- name: ListLabels :many
-- listLabels (M3 design 3.16, 3.19): the project's undeleted labels, parents and children alike, by sort order, then id;
-- an archived project's too.
SELECT id, workspace_id, project_id, parent_id, name, color, sort_order, created_at, updated_at
FROM labels
WHERE project_id = sqlc.arg(project_id) AND deleted_at IS NULL
ORDER BY sort_order, id;

-- name: GreatestSortOrder :one
-- createLabel, under the project's FOR NO KEY UPDATE (M3 design 4.10): the greatest sort order of the project's
-- undeleted labels, parents and children alike, as Plane's Label.save() reads it; no row when it has none.
SELECT sort_order
FROM labels
WHERE project_id = sqlc.arg(project_id) AND deleted_at IS NULL
ORDER BY sort_order DESC
LIMIT 1;

-- name: HasChildren :one
-- updateLabel, under the project's FOR NO KEY UPDATE (M3 design 3.16): whether an undeleted label has the label as its
-- parent.
SELECT EXISTS (SELECT 1 FROM labels WHERE parent_id = sqlc.arg(id)::uuid AND deleted_at IS NULL);

-- name: UpdateLabel :one
-- updateLabel, under the project's FOR NO KEY UPDATE (M3 design 3.16, 6.7): a field left out, null here, keeps its value;
-- the parent changes when set_parent is true, to none for a null parent_id. A deleted label is not written.
UPDATE labels l
SET name          = coalesce(sqlc.narg(name)::text, l.name),
    color         = coalesce(sqlc.narg(color)::text, l.color),
    parent_id     = CASE WHEN sqlc.arg(set_parent)::boolean THEN sqlc.narg(parent_id)::uuid ELSE l.parent_id END,
    sort_order    = coalesce(sqlc.narg(sort_order)::double precision, l.sort_order),
    updated_by_id = sqlc.arg(updated_by)::uuid,
    updated_at    = sqlc.arg(now)
WHERE l.id = sqlc.arg(id) AND l.deleted_at IS NULL
RETURNING l.id, l.workspace_id, l.project_id, l.parent_id, l.name, l.color, l.sort_order, l.created_at, l.updated_at;

-- name: DeleteLabel :exec
-- deleteLabel, one statement under the project's FOR NO KEY UPDATE (M3 design 3.16, 3.6 convention 5): the undeleted
-- label and the undeleted labels under it, at the moment and by the account given. A label's children are of its
-- project (3.16), so that lock covers every row the statement writes.
UPDATE labels
SET deleted_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE (id = sqlc.arg(id) OR parent_id = sqlc.arg(id)) AND deleted_at IS NULL;
