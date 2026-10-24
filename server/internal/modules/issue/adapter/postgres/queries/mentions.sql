-- name: MentionIDs :many
SELECT mention_id FROM issue_mentions WHERE issue_id = $1 AND deleted_at IS NULL ORDER BY mention_id;

-- name: InsertMentions :exec
-- A work item's new mentions' rows, one statement: ids[i] the row of mention_ids[i], in the order given (the
-- accounts' id order, convention 5's second half), at the moment and by the account given.
INSERT INTO issue_mentions (id, workspace_id, project_id, issue_id, mention_id, created_by_id, updated_by_id, created_at,
                             updated_at)
SELECT unnest(sqlc.arg(ids)::uuid[]), sqlc.arg(workspace_id), sqlc.arg(project_id), sqlc.arg(issue_id),
       unnest(sqlc.arg(mention_ids)::uuid[]), sqlc.arg(by)::uuid, sqlc.arg(by), sqlc.arg(at)::timestamptz, sqlc.arg(at);

-- name: RemoveMentions :exec
-- The mentions updateIssue removes (M4 design 3.7): the work item's undeleted rows of the accounts, soft-deleted at
-- the moment and by the account given, in one statement. Every write of these rows holds the project's FOR NO KEY
-- UPDATE first (3.5): no other write reaches them meanwhile, so they are not locked by id first (M3 design 3.6,
-- convention 5). The partial unique key on (issue_id, mention_id) serves it.
UPDATE issue_mentions
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(by)::uuid
WHERE issue_id = sqlc.arg(issue_id) AND mention_id = ANY (sqlc.arg(mention_ids)::uuid[]) AND deleted_at IS NULL;
