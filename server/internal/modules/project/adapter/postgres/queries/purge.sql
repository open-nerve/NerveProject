-- name: PurgeProjectUserProperties :many
DELETE FROM project_user_properties
WHERE id IN (
    SELECT candidate.id FROM project_user_properties AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
)
RETURNING id;

-- name: PurgeProjectMembers :many
DELETE FROM project_members
WHERE id IN (
    SELECT candidate.id FROM project_members AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
)
RETURNING id;

-- name: PurgeLabels :many
DELETE FROM labels
WHERE id IN (
    SELECT candidate.id FROM labels AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
)
RETURNING id;

-- name: PurgeStates :many
DELETE FROM states
WHERE id IN (
    SELECT candidate.id FROM states AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
)
RETURNING id;

-- name: PurgeProjects :many
DELETE FROM projects
WHERE id IN (
    SELECT candidate.id FROM projects AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
)
RETURNING id;

