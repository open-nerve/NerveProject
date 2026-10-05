-- name: CreateState :one
-- createProject's six states and createState's one (M3 design 3.17): the row as stored.
INSERT INTO states (id, workspace_id, project_id, name, description, color, sequence, "group", "default", created_by_id,
                    updated_by_id, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(project_id), sqlc.arg(name), sqlc.arg(description), sqlc.arg(color),
        sqlc.arg(sequence), sqlc.arg(state_group), sqlc.arg(is_default), sqlc.arg(created_by), sqlc.arg(created_by), sqlc.arg(now),
        sqlc.arg(now))
RETURNING id, workspace_id, project_id, name, description, color, "group", "default", sequence, created_at, updated_at;

-- name: StateByID :one
-- A write on a state named by its id (M3 design 3.6 convention 2, 6.7): the undeleted state, read first without a lock
-- for its project and the project's workspace, then again under their locks. The triage state is none (3.17).
SELECT id, workspace_id, project_id, name, description, color, "group", "default", sequence, created_at, updated_at
FROM states
WHERE id = sqlc.arg(id) AND "group" <> 'triage' AND deleted_at IS NULL;

-- name: ListStates :many
-- listStates (M3 design 3.12, 3.17): the project's undeleted states but its triage state, by sequence, then id; none
-- while the project is archived (Plane's views/state/base.py:37).
SELECT s.id, s.workspace_id, s.project_id, s.name, s.description, s.color, s."group", s."default", s.sequence, s.created_at,
       s.updated_at
FROM states s
         JOIN projects p ON p.id = s.project_id
WHERE s.project_id = sqlc.arg(project_id) AND s."group" <> 'triage' AND s.deleted_at IS NULL AND p.archived_at IS NULL
ORDER BY s.sequence, s.id;

-- name: GreatestSequence :one
-- createState, under the project's FOR NO KEY UPDATE (M3 design 3.17): the greatest sequence of the project's undeleted
-- states but its triage state, which Plane's State.objects leaves out (db/models/state.py:65-68); no row when it has
-- none.
SELECT sequence
FROM states
WHERE project_id = sqlc.arg(project_id) AND "group" <> 'triage' AND deleted_at IS NULL
ORDER BY sequence DESC
LIMIT 1;

-- name: UpdateState :one
-- updateState, under the project's FOR NO KEY UPDATE (M3 design 3.17, 6.7): a field left out, null here, keeps its
-- value. A deleted state and the triage state are not written.
UPDATE states s
SET name          = coalesce(sqlc.narg(name)::text, s.name),
    description   = coalesce(sqlc.narg(description)::text, s.description),
    color         = coalesce(sqlc.narg(color)::text, s.color),
    "group"       = coalesce(sqlc.narg(state_group)::text, s."group"),
    sequence      = coalesce(sqlc.narg(sequence)::double precision, s.sequence),
    updated_by_id = sqlc.arg(updated_by)::uuid,
    updated_at    = sqlc.arg(now)
WHERE s.id = sqlc.arg(id) AND s."group" <> 'triage' AND s.deleted_at IS NULL
RETURNING s.id, s.workspace_id, s.project_id, s.name, s.description, s.color, s."group", s."default", s.sequence, s.created_at,
    s.updated_at;

-- name: CountGroupStates :one
-- updateState and deleteState, under the project's FOR NO KEY UPDATE (M3 design 3.17): how many undeleted states the
-- project's group has.
SELECT count(*)
FROM states
WHERE project_id = sqlc.arg(project_id) AND "group" = sqlc.arg(state_group) AND deleted_at IS NULL;

-- name: DeleteState :execrows
-- deleteState, under the project's FOR NO KEY UPDATE (M3 design 3.17): a guarded write. The undeleted state, unless it
-- is the project's default or its triage state, deleted at the moment and by the account given; no row otherwise.
UPDATE states
SET deleted_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE id = sqlc.arg(id) AND NOT "default" AND "group" <> 'triage' AND deleted_at IS NULL;

-- name: ClearDefaultState :exec
-- markDefaultState's first statement, under the project's FOR NO KEY UPDATE (M3 design 3.17): the project's undeleted
-- default state the default no longer, at the moment and by the account given. states_project_id_default_key holds at
-- most one.
UPDATE states
SET "default" = false, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(updated_by)::uuid
WHERE project_id = sqlc.arg(project_id) AND "default" AND deleted_at IS NULL;

-- name: SetDefaultState :execrows
-- markDefaultState's second statement: the project's undeleted state the default, unless it is the triage state, at the
-- moment and by the account given; no row otherwise, and the caller rolls the first statement back.
UPDATE states
SET "default" = true, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(updated_by)::uuid
WHERE id = sqlc.arg(id) AND project_id = sqlc.arg(project_id) AND "group" <> 'triage' AND deleted_at IS NULL;

-- name: ListWorkspaceStates :many
-- listWorkspaceStates (M3 design 3.12, 3.17, 5.1): the undeleted states but the triage states of the workspace's
-- undeleted, unarchived projects the account is an active member of, by project id, then sequence, then id.
SELECT s.id, s.workspace_id, s.project_id, s.name, s.description, s.color, s."group", s."default", s.sequence, s.created_at,
       s.updated_at
FROM states s
         JOIN projects p ON p.id = s.project_id
         JOIN project_members m ON m.project_id = p.id
WHERE p.workspace_id = sqlc.arg(workspace_id) AND p.deleted_at IS NULL AND p.archived_at IS NULL
  AND m.member_id = sqlc.arg(user_id) AND m.is_active AND m.deleted_at IS NULL
  AND s."group" <> 'triage' AND s.deleted_at IS NULL
ORDER BY s.project_id, s.sequence, s.id;
