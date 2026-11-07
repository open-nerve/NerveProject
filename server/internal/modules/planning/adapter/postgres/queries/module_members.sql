-- name: InsertModuleMembers :exec
-- A module's new members' rows, one statement, in the order given: the i-th id is the row of the i-th account, each
-- written at one moment by one account (M6 P2 spec R5).
INSERT INTO module_members (id, workspace_id, project_id, module_id, member_id, created_by_id, updated_by_id, created_at,
                            updated_at)
SELECT unnest(sqlc.arg(ids)::uuid[]), sqlc.arg(workspace_id), sqlc.arg(project_id), sqlc.arg(module_id),
       unnest(sqlc.arg(member_ids)::uuid[]), sqlc.arg(by)::uuid, sqlc.arg(by), sqlc.arg(at)::timestamptz, sqlc.arg(at);
