-- name: GetModule :one
-- The undeleted module id, archived or not (M6 P2 spec 2.3), with the accounts of its undeleted members' rows in id
-- order, none as an empty array; read in the caller's transaction: under the project's lock for a write, which every
-- write on the project's modules takes first, so it stays as read until the write commits.
-- module_members_module_id_member_id_key (module_id, member_id) WHERE deleted_at IS NULL serves the members.
SELECT m.id, m.workspace_id, m.project_id, m.name, m.description, m.start_date, m.target_date, m.status, m.lead_id,
       m.sort_order, m.archived_at, m.created_at, m.updated_at, m.created_by_id, m.updated_by_id,
       ARRAY(SELECT mm.member_id FROM module_members mm
             WHERE mm.module_id = m.id AND mm.deleted_at IS NULL
             ORDER BY mm.member_id)::uuid[] AS member_ids
FROM modules m
WHERE m.id = sqlc.arg(id) AND m.deleted_at IS NULL;

-- name: LowestModuleSortOrder :one
-- The lowest sort order of the project's undeleted modules, archived ones too, no row when it has none: a new module
-- goes before it (M6 P2 spec 2.2). The list index (project_id, sort_order, created_at, id) WHERE deleted_at IS NULL
-- serves it.
SELECT sort_order
FROM modules
WHERE project_id = sqlc.arg(project_id) AND deleted_at IS NULL
ORDER BY sort_order
LIMIT 1;

-- name: CreateModule :exec
-- A new module, written at one moment by one account. A name another undeleted module of the project has, in any case,
-- breaks modules_project_id_name_key (M6 P2 spec R8).
INSERT INTO modules (id, workspace_id, project_id, name, description, start_date, target_date, status, lead_id,
                     sort_order, created_at, updated_at, created_by_id, updated_by_id)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(project_id), sqlc.arg(name), sqlc.arg(description),
        sqlc.narg(start_date), sqlc.narg(target_date), sqlc.arg(status), sqlc.narg(lead_id), sqlc.arg(sort_order),
        sqlc.arg(created_at), sqlc.arg(updated_at), sqlc.narg(created_by_id), sqlc.narg(updated_by_id));
