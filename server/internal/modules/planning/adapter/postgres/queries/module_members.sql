-- name: InsertModuleMembers :exec
-- A module's new members' rows, one statement, in the order given: the i-th id is the row of the i-th account, each
-- written at one moment by one account (M6 P2 spec R5).
INSERT INTO module_members (id, workspace_id, project_id, module_id, member_id, created_by_id, updated_by_id, created_at,
                            updated_at)
SELECT unnest(sqlc.arg(ids)::uuid[]), sqlc.arg(workspace_id), sqlc.arg(project_id), sqlc.arg(module_id),
       unnest(sqlc.arg(member_ids)::uuid[]), sqlc.arg(by)::uuid, sqlc.arg(by), sqlc.arg(at)::timestamptz, sqlc.arg(at);

-- name: RemoveModuleMembers :exec
-- The members a change of a module removes (M6 P2 spec R5): the module's undeleted rows of the accounts, soft-deleted at
-- the moment and by the account given, in one statement; a member it keeps, its row as it is. Every write of these rows
-- through the API holds the project's FOR NO KEY UPDATE first, and the deletion job's reach a deleted project's alone,
-- which no change reaches: no other write reaches them meanwhile, so they are not locked by id first (M3 design 3.6,
-- convention 5; as the work items' RemoveAssignees). module_members_module_id_member_id_key serves it.
UPDATE module_members
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(by)::uuid
WHERE module_id = sqlc.arg(module_id) AND member_id = ANY (sqlc.arg(member_ids)::uuid[]) AND deleted_at IS NULL;

-- name: DeleteModuleMemberRows :exec
-- A deleted module's members (M6 P2 spec 2.4, R15): its undeleted rows, locked FOR NO KEY UPDATE in id order, then
-- soft-deleted at the moment and by the account given, in one statement (M3 design 3.6, convention 5); a row deleted
-- before keeps its deletion. module_members_module_id_idx serves it.
UPDATE module_members AS t
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(by)::uuid
WHERE t.id IN (
    SELECT r.id FROM module_members r
    WHERE r.module_id = sqlc.arg(module_id) AND r.deleted_at IS NULL
    ORDER BY r.id FOR NO KEY UPDATE);
