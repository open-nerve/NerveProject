-- name: PurgeProjectUserProperties :one
WITH candidates AS MATERIALIZED (
    SELECT candidate.id FROM project_user_properties AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
), deleted AS (
    DELETE FROM project_user_properties AS target WHERE target.deleted_at < sqlc.arg(cutoff) AND target.id IN (SELECT candidates.id FROM candidates) RETURNING target.id
)
SELECT (SELECT count(*)::integer FROM candidates) AS scanned,
       (SELECT count(*)::integer FROM deleted) AS deleted,
       coalesce((SELECT id FROM candidates ORDER BY id DESC LIMIT 1), sqlc.arg(after_id)::uuid)::uuid AS after_id;

-- name: PurgeProjectMembers :one
WITH candidates AS MATERIALIZED (
    SELECT candidate.id FROM project_members AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
), deleted AS (
    DELETE FROM project_members AS target WHERE target.deleted_at < sqlc.arg(cutoff) AND target.id IN (SELECT candidates.id FROM candidates) RETURNING target.id
)
SELECT (SELECT count(*)::integer FROM candidates) AS scanned,
       (SELECT count(*)::integer FROM deleted) AS deleted,
       coalesce((SELECT id FROM candidates ORDER BY id DESC LIMIT 1), sqlc.arg(after_id)::uuid)::uuid AS after_id;

-- name: PurgeLabels :one
WITH candidates AS MATERIALIZED (
    SELECT candidate.id FROM labels AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
), deleted AS (
    DELETE FROM labels AS target WHERE target.deleted_at < sqlc.arg(cutoff) AND target.id IN (SELECT candidates.id FROM candidates) RETURNING target.id
)
SELECT (SELECT count(*)::integer FROM candidates) AS scanned,
       (SELECT count(*)::integer FROM deleted) AS deleted,
       coalesce((SELECT id FROM candidates ORDER BY id DESC LIMIT 1), sqlc.arg(after_id)::uuid)::uuid AS after_id;

-- name: PurgeStatesCandidates :many
SELECT id FROM states
WHERE deleted_at < sqlc.arg(cutoff) AND id > sqlc.arg(after_id)
ORDER BY id LIMIT sqlc.arg(batch_limit);

-- name: DeletePurgeStates :many
DELETE FROM states
WHERE deleted_at < sqlc.arg(cutoff) AND id = ANY(sqlc.arg(ids)::uuid[])
RETURNING id;

-- name: PurgeProjectsCandidates :many
SELECT id FROM projects
WHERE deleted_at < sqlc.arg(cutoff) AND id > sqlc.arg(after_id)
ORDER BY id LIMIT sqlc.arg(batch_limit);

-- name: DeletePurgeProjects :many
DELETE FROM projects
WHERE deleted_at < sqlc.arg(cutoff) AND id = ANY(sqlc.arg(ids)::uuid[])
RETURNING id;
