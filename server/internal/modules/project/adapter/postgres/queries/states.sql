-- name: CreateState :exec
INSERT INTO states (id, workspace_id, project_id, name, color, sequence, "group", "default", created_by_id, updated_by_id,
                    created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(project_id), sqlc.arg(name), sqlc.arg(color), sqlc.arg(sequence),
        sqlc.arg(state_group), sqlc.arg(is_default), sqlc.arg(created_by), sqlc.arg(created_by), sqlc.arg(now), sqlc.arg(now));
