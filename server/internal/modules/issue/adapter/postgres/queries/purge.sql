-- name: PurgeCommentReactions :many
DELETE FROM comment_reactions
WHERE id IN (
    SELECT candidate.id FROM comment_reactions AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
)
RETURNING id;

-- name: PurgeIssueReactions :many
DELETE FROM issue_reactions
WHERE id IN (
    SELECT candidate.id FROM issue_reactions AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
)
RETURNING id;

-- name: PurgeIssueMentions :many
DELETE FROM issue_mentions
WHERE id IN (
    SELECT candidate.id FROM issue_mentions AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
)
RETURNING id;

-- name: PurgeIssueDescriptionVersions :many
DELETE FROM issue_description_versions
WHERE id IN (
    SELECT candidate.id FROM issue_description_versions AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
)
RETURNING id;

-- name: PurgeIssueActivities :many
DELETE FROM issue_activities
WHERE id IN (
    SELECT candidate.id FROM issue_activities AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
)
RETURNING id;

-- name: PurgeIssueComments :many
DELETE FROM issue_comments
WHERE id IN (
    SELECT candidate.id FROM issue_comments AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
)
RETURNING id;

-- name: PurgeIssueLinks :many
DELETE FROM issue_links
WHERE id IN (
    SELECT candidate.id FROM issue_links AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
)
RETURNING id;

-- name: PurgeIssueRelations :many
DELETE FROM issue_relations
WHERE id IN (
    SELECT candidate.id FROM issue_relations AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
)
RETURNING id;

-- name: PurgeIssueAssignees :many
DELETE FROM issue_assignees
WHERE id IN (
    SELECT candidate.id FROM issue_assignees AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
)
RETURNING id;

-- name: PurgeIssueLabels :many
DELETE FROM issue_labels
WHERE id IN (
    SELECT candidate.id FROM issue_labels AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
)
RETURNING id;

-- name: PurgeIssueSubscribers :many
DELETE FROM issue_subscribers
WHERE id IN (
    SELECT candidate.id FROM issue_subscribers AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
)
RETURNING id;

-- name: PurgeDraftIssues :many
DELETE FROM draft_issues
WHERE id IN (
    SELECT candidate.id FROM draft_issues AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
)
RETURNING id;

-- name: PurgeIssues :many
DELETE FROM issues
WHERE id IN (
    SELECT candidate.id FROM issues AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
)
RETURNING id;
