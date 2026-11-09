-- name: PurgeCycles :one
-- One bounded page of the 60-day purge (M6 P1 spec 2.4, P2 spec 2.4): the cycles deleted before the cutoff after the
-- cursor, in id order, deleted in one statement; it returns how many it read and deleted and the page's greatest id.
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

-- name: PurgeModules :one
-- PurgeCycles' page, of the modules. Their members and links are purged before them (module.go), so the CASCADE of a
-- module's deletion finds none of its own rows to take.
WITH candidates AS MATERIALIZED (
    SELECT candidate.id FROM modules AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
), deleted AS (
    DELETE FROM modules AS target WHERE target.deleted_at < sqlc.arg(cutoff) AND target.id IN (SELECT candidates.id FROM candidates) RETURNING target.id
)
SELECT (SELECT count(*)::integer FROM candidates) AS scanned,
       (SELECT count(*)::integer FROM deleted) AS deleted,
       coalesce((SELECT id FROM candidates ORDER BY id DESC LIMIT 1), sqlc.arg(after_id)::uuid)::uuid AS after_id;

-- name: PurgeModuleMembers :one
-- PurgeCycles' page, of the modules' members.
WITH candidates AS MATERIALIZED (
    SELECT candidate.id FROM module_members AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
), deleted AS (
    DELETE FROM module_members AS target WHERE target.deleted_at < sqlc.arg(cutoff) AND target.id IN (SELECT candidates.id FROM candidates) RETURNING target.id
)
SELECT (SELECT count(*)::integer FROM candidates) AS scanned,
       (SELECT count(*)::integer FROM deleted) AS deleted,
       coalesce((SELECT id FROM candidates ORDER BY id DESC LIMIT 1), sqlc.arg(after_id)::uuid)::uuid AS after_id;

-- name: PurgeModuleLinks :one
-- PurgeCycles' page, of the modules' links.
WITH candidates AS MATERIALIZED (
    SELECT candidate.id FROM module_links AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
), deleted AS (
    DELETE FROM module_links AS target WHERE target.deleted_at < sqlc.arg(cutoff) AND target.id IN (SELECT candidates.id FROM candidates) RETURNING target.id
)
SELECT (SELECT count(*)::integer FROM candidates) AS scanned,
       (SELECT count(*)::integer FROM deleted) AS deleted,
       coalesce((SELECT id FROM candidates ORDER BY id DESC LIMIT 1), sqlc.arg(after_id)::uuid)::uuid AS after_id;

-- name: PurgeCycleUserProperties :one
-- PurgeCycles' page, of the accounts' display settings of the cycles' pages (M6/P3b spec 2.5). They are purged before
-- the cycles (module.go), so the CASCADE of a cycle's deletion finds none of them to take.
WITH candidates AS MATERIALIZED (
    SELECT candidate.id FROM cycle_user_properties AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
), deleted AS (
    DELETE FROM cycle_user_properties AS target WHERE target.deleted_at < sqlc.arg(cutoff) AND target.id IN (SELECT candidates.id FROM candidates) RETURNING target.id
)
SELECT (SELECT count(*)::integer FROM candidates) AS scanned,
       (SELECT count(*)::integer FROM deleted) AS deleted,
       coalesce((SELECT id FROM candidates ORDER BY id DESC LIMIT 1), sqlc.arg(after_id)::uuid)::uuid AS after_id;

-- name: PurgeModuleUserProperties :one
-- PurgeCycles' page, of the accounts' display settings of the modules' pages, purged before the modules.
WITH candidates AS MATERIALIZED (
    SELECT candidate.id FROM module_user_properties AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
), deleted AS (
    DELETE FROM module_user_properties AS target WHERE target.deleted_at < sqlc.arg(cutoff) AND target.id IN (SELECT candidates.id FROM candidates) RETURNING target.id
)
SELECT (SELECT count(*)::integer FROM candidates) AS scanned,
       (SELECT count(*)::integer FROM deleted) AS deleted,
       coalesce((SELECT id FROM candidates ORDER BY id DESC LIMIT 1), sqlc.arg(after_id)::uuid)::uuid AS after_id;
