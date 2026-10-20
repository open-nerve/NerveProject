-- name: UpdateIssue :exec
-- updateIssue's work item (M4 design 3.7), under the project's FOR NO KEY UPDATE, read again under it: its name, state,
-- priority and completion as the use case gives them, those it changes and the others as they are, last changed at the
-- moment and by the account given. Every write of a work item's row holds its project's lock first (3.5), so the row is
-- as the use case read it; it is not locked by id first (M3 design 3.6, convention 5).
UPDATE issues
SET name = sqlc.arg(name), state_id = sqlc.arg(state_id), priority = sqlc.arg(priority), completed_at = sqlc.narg(completed_at),
    updated_by_id = sqlc.arg(by)::uuid, updated_at = sqlc.arg(at)::timestamptz
WHERE id = sqlc.arg(id);

-- name: SetArchivedAt :exec
-- archiveIssue's and unarchiveIssue's work item (M4 design 3.8), under the project's FOR NO KEY UPDATE, read again under
-- it: its archived_at, the moment or NULL, last changed at the moment and by the account given; every other column as
-- it is. It is not locked by id first, as UpdateIssue.
UPDATE issues
SET archived_at = sqlc.narg(archived_at), updated_by_id = sqlc.arg(by)::uuid, updated_at = sqlc.arg(at)::timestamptz
WHERE id = sqlc.arg(id);

-- name: RemoveAssignees :exec
-- The assignees updateIssue removes (M4 design 3.7): the work item's undeleted rows of the accounts, soft-deleted at
-- the moment and by the account given, in one statement. Every write of these rows holds the project's FOR NO KEY
-- UPDATE first (3.5): no other write reaches them meanwhile, so they are not locked by id first (M3 design 3.6,
-- convention 5). The partial unique key on (issue_id, assignee_id) serves it.
UPDATE issue_assignees
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(by)::uuid
WHERE issue_id = sqlc.arg(issue_id) AND assignee_id = ANY (sqlc.arg(assignee_ids)::uuid[]) AND deleted_at IS NULL;

-- name: RemoveLabels :exec
-- The labels updateIssue removes, as RemoveAssignees; the partial unique key on (issue_id, label_id) serves it.
UPDATE issue_labels
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(by)::uuid
WHERE issue_id = sqlc.arg(issue_id) AND label_id = ANY (sqlc.arg(label_ids)::uuid[]) AND deleted_at IS NULL;
