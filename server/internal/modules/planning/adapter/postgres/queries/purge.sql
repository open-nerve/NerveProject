-- name: PurgeCycles :one
-- One bounded page of the 60-day purge (M6 P1 spec 2.4): the cycles deleted before the cutoff after the cursor, in id
-- order, deleted in one statement; it returns how many it read and deleted and the page's greatest id.
WITH candidates AS MATERIALIZED (
    SELECT candidate.id FROM cycles AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
), deleted AS (
    DELETE FROM cycles AS target WHERE target.deleted_at < sqlc.arg(cutoff) AND target.id IN (SELECT candidates.id FROM candidates) RETURNING target.id
)
SELECT (SELECT count(*)::integer FROM candidates) AS scanned,
       (SELECT count(*)::integer FROM deleted) AS deleted,
       coalesce((SELECT id FROM candidates ORDER BY id DESC LIMIT 1), sqlc.arg(after_id)::uuid)::uuid AS after_id;
