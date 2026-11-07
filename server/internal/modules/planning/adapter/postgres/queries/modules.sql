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
