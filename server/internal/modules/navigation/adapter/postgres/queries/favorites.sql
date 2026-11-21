-- name: ListFavorites :many
-- One's undeleted favorites in a workspace, those in his folders too, by sequence descending, then by id (M7 P3c spec
-- 2.4): two made at once may share a sequence (spec 2.5). The partial index user_favorites_user_id_workspace_id_idx
-- serves it.
SELECT id, workspace_id, project_id, user_id, entity_type, entity_identifier, name, parent_id, sequence, created_at, updated_at
FROM user_favorites
WHERE user_id = sqlc.arg(user_id) AND workspace_id = sqlc.arg(workspace_id) AND deleted_at IS NULL
ORDER BY sequence DESC, id DESC;

-- name: GetFavorite :one
-- One's undeleted favorite id, read without a lock: /favorites/{favorite_id} reads it first (spec 2.7). Another's is
-- not found.
SELECT id, workspace_id, project_id, user_id, entity_type, entity_identifier, name, parent_id, sequence, created_at, updated_at
FROM user_favorites
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) AND deleted_at IS NULL;

-- name: GetFavoriteOfTarget :one
-- One's undeleted favorite of a target, which the insertion of another favorite of it met (spec 2.5): one his other
-- creation of it at once made after this creation's lock found none (ShareFavoriteOfTarget).
SELECT id, workspace_id, project_id, user_id, entity_type, entity_identifier, name, parent_id, sequence, created_at, updated_at
FROM user_favorites
WHERE user_id = sqlc.arg(user_id) AND entity_type = sqlc.arg(entity_type) AND entity_identifier = sqlc.narg(entity_identifier)
  AND deleted_at IS NULL;

-- name: GreatestFavoriteSequence :one
-- The greatest sequence of one's undeleted favorites in a workspace, a new one's step (spec 2.5); no row when he has
-- none. It takes no lock: two made at once may share a sequence, which the list's order by id tells apart.
SELECT sequence
FROM user_favorites
WHERE user_id = sqlc.arg(user_id) AND workspace_id = sqlc.arg(workspace_id) AND deleted_at IS NULL
ORDER BY sequence DESC
LIMIT 1;

-- name: InsertFavoriteItem :execrows
-- A new favorite of a target, made by one account at one moment, unless its owner has an undeleted favorite of the
-- target: then nothing, and the caller reads that one (spec 2.3, 2.5). The conflict target is that partial unique index
-- alone, so any other unique violation is raised, never passed over.
INSERT INTO user_favorites (id, workspace_id, project_id, user_id, entity_type, entity_identifier, name, parent_id, sequence,
                            created_by_id, updated_by_id, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.narg(project_id), sqlc.arg(user_id), sqlc.arg(entity_type),
        sqlc.narg(entity_identifier), sqlc.narg(name), sqlc.narg(parent_id), sqlc.arg(sequence), sqlc.arg(by)::uuid,
        sqlc.arg(by)::uuid, sqlc.arg(created_at), sqlc.arg(updated_at))
ON CONFLICT (user_id, entity_type, entity_identifier) WHERE deleted_at IS NULL DO NOTHING;

-- name: InsertFavoriteFolder :exec
-- A new folder, made by one account at one moment (spec 2.3): no conflict is passed over, so a name, in any case,
-- another of its owner's undeleted folders in the workspace has is the violation of
-- user_favorites_user_id_workspace_id_name_key, which the caller answers as 409.
INSERT INTO user_favorites (id, workspace_id, project_id, user_id, entity_type, entity_identifier, name, parent_id, sequence,
                            created_by_id, updated_by_id, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.narg(project_id), sqlc.arg(user_id), sqlc.arg(entity_type),
        sqlc.narg(entity_identifier), sqlc.narg(name), sqlc.narg(parent_id), sqlc.arg(sequence), sqlc.arg(by)::uuid,
        sqlc.arg(by)::uuid, sqlc.arg(created_at), sqlc.arg(updated_at));

-- name: ShareFolder :one
-- The parent a creation or a change names, locked FOR SHARE until the transaction ends, before the favorite's own row
-- (spec 2.5, 2.8): one's own undeleted folder in the workspace. Any other row is no parent, and is not locked:
-- another's folder, his folder of another workspace, his favorite of a target. So a parent's lock is always a folder
-- row's, which the total order of favorite rows takes before the item rows.
SELECT id
FROM user_favorites
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) AND workspace_id = sqlc.arg(workspace_id) AND entity_type = 'folder'
  AND deleted_at IS NULL
FOR SHARE;

-- name: ShareFavoriteOfTarget :one
-- One's undeleted favorite of a target, locked FOR SHARE until the transaction ends: a creation's of a target last
-- lock, after its parent's, an item row after the folder rows in the total order of favorite rows (spec 2.5, 2.8).
-- Found, the creation answers it as it is, and its deletion waits until the creation ends. Not found, the creation
-- inserts: so too once a deletion that held it has committed, for the row read again is then its deleted version,
-- which this predicate leaves out. Another's favorite of the target, his deleted one and his of another target are
-- not found, nor locked. The partial unique index user_favorites_user_id_entity_type_entity_identifier_key serves it.
SELECT id, workspace_id, project_id, user_id, entity_type, entity_identifier, name, parent_id, sequence, created_at, updated_at
FROM user_favorites
WHERE user_id = sqlc.arg(user_id) AND entity_type = sqlc.arg(entity_type) AND entity_identifier = sqlc.arg(entity_identifier)::uuid
  AND deleted_at IS NULL
FOR SHARE;

-- name: LockFavorite :one
-- One's undeleted favorite id, locked FOR UPDATE after its workspace's and project's FOR SHARE and its new parent's
-- (spec 2.8): a change's or a deletion's read under its locks. Another's is not found, nor locked.
SELECT id, workspace_id, project_id, user_id, entity_type, entity_identifier, name, parent_id, sequence, created_at, updated_at
FROM user_favorites
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) AND deleted_at IS NULL
FOR UPDATE;

-- name: LockFolderChildren :exec
-- The undeleted favorites a folder holds, locked FOR UPDATE in id order after the folder's own FOR UPDATE (spec 2.8): a
-- folder's deletion's locks before it writes them.
SELECT id
FROM user_favorites
WHERE parent_id = sqlc.arg(folder_id)::uuid AND deleted_at IS NULL
ORDER BY id
FOR UPDATE;

-- name: UpdateFavorite :exec
-- A favorite's parent, sequence and, a folder's, name replaced, last changed by one account at one moment (spec 2.4). A
-- name another of its owner's undeleted folders in the workspace has, in any case, is the violation of
-- user_favorites_user_id_workspace_id_name_key; its own in another case is not.
UPDATE user_favorites
SET parent_id     = sqlc.narg(parent_id),
    sequence      = sqlc.arg(sequence),
    name          = sqlc.narg(name),
    updated_at    = sqlc.arg(updated_at),
    updated_by_id = sqlc.arg(by)::uuid
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;

-- name: DeleteFavorite :exec
-- A favorite soft-deleted by one account at one moment, its last change the same, and with a folder the favorites it
-- holds, every row at the same moment by the same account (spec 2.5). Under a folder's deletion's locks: the folder,
-- then its favorites in id order (spec 2.8). One deleted already keeps its deletion.
UPDATE user_favorites
SET deleted_at    = sqlc.arg(at)::timestamptz,
    updated_at    = sqlc.arg(at),
    updated_by_id = sqlc.arg(by)::uuid
WHERE (id = sqlc.arg(id) OR parent_id = sqlc.arg(id)) AND deleted_at IS NULL;

-- name: DeleteFavoritesOfTarget :exec
-- The cascade of a target's deletion or archiving (spec 2.6): every account's undeleted favorites of the target, locked
-- FOR NO KEY UPDATE in id order and soft-deleted at the event's moment by its actor. A target's favorites are item rows
-- alone (spec 2.8).
UPDATE user_favorites AS t
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(by)::uuid
WHERE t.id IN (
    SELECT r.id FROM user_favorites r
    WHERE r.entity_type = sqlc.arg(entity_type) AND r.entity_identifier = sqlc.arg(entity_identifier)::uuid AND r.deleted_at IS NULL
    ORDER BY r.id FOR NO KEY UPDATE);

-- name: DeleteFavoritesOfProject :exec
-- The cascade of a project's archiving (spec 2.6): every account's undeleted favorites whose project it is, its own and
-- its cycles', modules' and views', as DeleteFavoritesOfTarget's. No folder has a project.
UPDATE user_favorites AS t
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(by)::uuid
WHERE t.id IN (
    SELECT r.id FROM user_favorites r
    WHERE r.project_id = sqlc.arg(project_id)::uuid AND r.deleted_at IS NULL
    ORDER BY r.id FOR NO KEY UPDATE);
