-- name: PurgeCommentReactions :one
WITH candidates AS MATERIALIZED (
    SELECT candidate.id FROM comment_reactions AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
), deleted AS (
    DELETE FROM comment_reactions AS target WHERE target.deleted_at < sqlc.arg(cutoff) AND target.id IN (SELECT candidates.id FROM candidates) RETURNING target.id
)
SELECT (SELECT count(*)::integer FROM candidates) AS scanned,
       (SELECT count(*)::integer FROM deleted) AS deleted,
       coalesce((SELECT id FROM candidates ORDER BY id DESC LIMIT 1), sqlc.arg(after_id)::uuid)::uuid AS after_id;

-- name: PurgeIssueReactions :one
WITH candidates AS MATERIALIZED (
    SELECT candidate.id FROM issue_reactions AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
), deleted AS (
    DELETE FROM issue_reactions AS target WHERE target.deleted_at < sqlc.arg(cutoff) AND target.id IN (SELECT candidates.id FROM candidates) RETURNING target.id
)
SELECT (SELECT count(*)::integer FROM candidates) AS scanned,
       (SELECT count(*)::integer FROM deleted) AS deleted,
       coalesce((SELECT id FROM candidates ORDER BY id DESC LIMIT 1), sqlc.arg(after_id)::uuid)::uuid AS after_id;

-- name: PurgeIssueMentions :one
WITH candidates AS MATERIALIZED (
    SELECT candidate.id FROM issue_mentions AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
), deleted AS (
    DELETE FROM issue_mentions AS target WHERE target.deleted_at < sqlc.arg(cutoff) AND target.id IN (SELECT candidates.id FROM candidates) RETURNING target.id
)
SELECT (SELECT count(*)::integer FROM candidates) AS scanned,
       (SELECT count(*)::integer FROM deleted) AS deleted,
       coalesce((SELECT id FROM candidates ORDER BY id DESC LIMIT 1), sqlc.arg(after_id)::uuid)::uuid AS after_id;

-- name: PurgeIssueDescriptionVersions :one
WITH candidates AS MATERIALIZED (
    SELECT candidate.id FROM issue_description_versions AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
), deleted AS (
    DELETE FROM issue_description_versions AS target WHERE target.deleted_at < sqlc.arg(cutoff) AND target.id IN (SELECT candidates.id FROM candidates) RETURNING target.id
)
SELECT (SELECT count(*)::integer FROM candidates) AS scanned,
       (SELECT count(*)::integer FROM deleted) AS deleted,
       coalesce((SELECT id FROM candidates ORDER BY id DESC LIMIT 1), sqlc.arg(after_id)::uuid)::uuid AS after_id;

-- name: PurgeIssueActivities :one
WITH candidates AS MATERIALIZED (
    SELECT candidate.id FROM issue_activities AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
), deleted AS (
    DELETE FROM issue_activities AS target WHERE target.deleted_at < sqlc.arg(cutoff) AND target.id IN (SELECT candidates.id FROM candidates) RETURNING target.id
)
SELECT (SELECT count(*)::integer FROM candidates) AS scanned,
       (SELECT count(*)::integer FROM deleted) AS deleted,
       coalesce((SELECT id FROM candidates ORDER BY id DESC LIMIT 1), sqlc.arg(after_id)::uuid)::uuid AS after_id;

-- name: PurgeIssueCommentsCandidates :many
SELECT id FROM issue_comments
WHERE deleted_at < sqlc.arg(cutoff) AND id > sqlc.arg(after_id)
ORDER BY id LIMIT sqlc.arg(batch_limit);

-- name: DeletePurgeIssueComments :many
DELETE FROM issue_comments
WHERE deleted_at < sqlc.arg(cutoff) AND id = ANY(sqlc.arg(ids)::uuid[])
RETURNING id;

-- name: PurgeIssueLinks :one
WITH candidates AS MATERIALIZED (
    SELECT candidate.id FROM issue_links AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
), deleted AS (
    DELETE FROM issue_links AS target WHERE target.deleted_at < sqlc.arg(cutoff) AND target.id IN (SELECT candidates.id FROM candidates) RETURNING target.id
)
SELECT (SELECT count(*)::integer FROM candidates) AS scanned,
       (SELECT count(*)::integer FROM deleted) AS deleted,
       coalesce((SELECT id FROM candidates ORDER BY id DESC LIMIT 1), sqlc.arg(after_id)::uuid)::uuid AS after_id;

-- name: PurgeIssueRelations :one
WITH candidates AS MATERIALIZED (
    SELECT candidate.id FROM issue_relations AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
), deleted AS (
    DELETE FROM issue_relations AS target WHERE target.deleted_at < sqlc.arg(cutoff) AND target.id IN (SELECT candidates.id FROM candidates) RETURNING target.id
)
SELECT (SELECT count(*)::integer FROM candidates) AS scanned,
       (SELECT count(*)::integer FROM deleted) AS deleted,
       coalesce((SELECT id FROM candidates ORDER BY id DESC LIMIT 1), sqlc.arg(after_id)::uuid)::uuid AS after_id;

-- name: PurgeIssueAssignees :one
WITH candidates AS MATERIALIZED (
    SELECT candidate.id FROM issue_assignees AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
), deleted AS (
    DELETE FROM issue_assignees AS target WHERE target.deleted_at < sqlc.arg(cutoff) AND target.id IN (SELECT candidates.id FROM candidates) RETURNING target.id
)
SELECT (SELECT count(*)::integer FROM candidates) AS scanned,
       (SELECT count(*)::integer FROM deleted) AS deleted,
       coalesce((SELECT id FROM candidates ORDER BY id DESC LIMIT 1), sqlc.arg(after_id)::uuid)::uuid AS after_id;

-- name: PurgeIssueLabels :one
WITH candidates AS MATERIALIZED (
    SELECT candidate.id FROM issue_labels AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
), deleted AS (
    DELETE FROM issue_labels AS target WHERE target.deleted_at < sqlc.arg(cutoff) AND target.id IN (SELECT candidates.id FROM candidates) RETURNING target.id
)
SELECT (SELECT count(*)::integer FROM candidates) AS scanned,
       (SELECT count(*)::integer FROM deleted) AS deleted,
       coalesce((SELECT id FROM candidates ORDER BY id DESC LIMIT 1), sqlc.arg(after_id)::uuid)::uuid AS after_id;

-- name: PurgeIssueSubscribers :one
WITH candidates AS MATERIALIZED (
    SELECT candidate.id FROM issue_subscribers AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
), deleted AS (
    DELETE FROM issue_subscribers AS target WHERE target.deleted_at < sqlc.arg(cutoff) AND target.id IN (SELECT candidates.id FROM candidates) RETURNING target.id
)
SELECT (SELECT count(*)::integer FROM candidates) AS scanned,
       (SELECT count(*)::integer FROM deleted) AS deleted,
       coalesce((SELECT id FROM candidates ORDER BY id DESC LIMIT 1), sqlc.arg(after_id)::uuid)::uuid AS after_id;

-- name: PurgeModuleIssues :one
WITH candidates AS MATERIALIZED (
    SELECT candidate.id FROM module_issues AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
), deleted AS (
    DELETE FROM module_issues AS target WHERE target.deleted_at < sqlc.arg(cutoff) AND target.id IN (SELECT candidates.id FROM candidates) RETURNING target.id
)
SELECT (SELECT count(*)::integer FROM candidates) AS scanned,
       (SELECT count(*)::integer FROM deleted) AS deleted,
       coalesce((SELECT id FROM candidates ORDER BY id DESC LIMIT 1), sqlc.arg(after_id)::uuid)::uuid AS after_id;

-- name: PurgeCycleIssues :one
WITH candidates AS MATERIALIZED (
    SELECT candidate.id FROM cycle_issues AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
), deleted AS (
    DELETE FROM cycle_issues AS target WHERE target.deleted_at < sqlc.arg(cutoff) AND target.id IN (SELECT candidates.id FROM candidates) RETURNING target.id
)
SELECT (SELECT count(*)::integer FROM candidates) AS scanned,
       (SELECT count(*)::integer FROM deleted) AS deleted,
       coalesce((SELECT id FROM candidates ORDER BY id DESC LIMIT 1), sqlc.arg(after_id)::uuid)::uuid AS after_id;

-- name: PurgeDraftIssuesCandidates :many
SELECT id FROM draft_issues
WHERE deleted_at < sqlc.arg(cutoff) AND id > sqlc.arg(after_id)
ORDER BY id LIMIT sqlc.arg(batch_limit);

-- name: DeletePurgeDraftIssues :many
DELETE FROM draft_issues
WHERE deleted_at < sqlc.arg(cutoff) AND id = ANY(sqlc.arg(ids)::uuid[])
RETURNING id;

-- name: PurgeIssuesCandidates :many
SELECT id FROM issues
WHERE deleted_at < sqlc.arg(cutoff) AND id > sqlc.arg(after_id)
ORDER BY id LIMIT sqlc.arg(batch_limit);

-- name: DeletePurgeIssues :many
DELETE FROM issues
WHERE deleted_at < sqlc.arg(cutoff) AND id = ANY(sqlc.arg(ids)::uuid[])
RETURNING id;
