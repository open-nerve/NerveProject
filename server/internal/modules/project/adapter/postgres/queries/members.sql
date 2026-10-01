-- name: CreateMember :exec
INSERT INTO project_members (id, workspace_id, project_id, member_id, role, created_by_id, updated_by_id, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(project_id), sqlc.arg(member_id), sqlc.arg(role), sqlc.arg(created_by),
        sqlc.arg(created_by), sqlc.arg(now), sqlc.arg(now));
