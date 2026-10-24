-- name: GreatestSortOrder :one
-- The greatest sort order of the project's undeleted work items in the state, archived ones too; no row when it has
-- none: a new work item's place after them (M4 design 3.7). Read under the project's FOR NO KEY UPDATE, which every
-- write of a work item holds, so it stays as read until the write commits (3.5).
SELECT sort_order
FROM issues
WHERE project_id = sqlc.arg(project_id) AND state_id = sqlc.arg(state_id) AND deleted_at IS NULL
ORDER BY sort_order DESC
LIMIT 1;

-- name: InsertIssue :exec
-- createIssue's work item (M4 design 3.5), under the project's FOR NO KEY UPDATE; created and last changed at the
-- moment and by the account given. Description values are explicit, including nullable legacy text.
INSERT INTO issues (id, workspace_id, project_id, state_id, name, priority, start_date, target_date, parent_id, sequence_id,
                    sort_order, completed_at, description_html, description_stripped, created_by_id, updated_by_id,
                    created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(project_id), sqlc.arg(state_id), sqlc.arg(name), sqlc.arg(priority),
        sqlc.narg(start_date), sqlc.narg(target_date), sqlc.narg(parent_id), sqlc.arg(sequence_id), sqlc.arg(sort_order),
        sqlc.narg(completed_at), sqlc.arg(description_html), sqlc.narg(description_stripped), sqlc.arg(by)::uuid, sqlc.arg(by),
        sqlc.arg(at)::timestamptz, sqlc.arg(at));

-- name: InsertAssignees :exec
-- A work item's new assignees' rows, one statement: ids[i] the row of assignee_ids[i], in the order given (the
-- accounts' id order, convention 5's second half), at the moment and by the account given.
INSERT INTO issue_assignees (id, workspace_id, project_id, issue_id, assignee_id, created_by_id, updated_by_id, created_at,
                             updated_at)
SELECT unnest(sqlc.arg(ids)::uuid[]), sqlc.arg(workspace_id), sqlc.arg(project_id), sqlc.arg(issue_id),
       unnest(sqlc.arg(assignee_ids)::uuid[]), sqlc.arg(by)::uuid, sqlc.arg(by), sqlc.arg(at)::timestamptz, sqlc.arg(at);

-- name: InsertLabels :exec
-- A work item's new labels' rows, one statement, as InsertAssignees.
INSERT INTO issue_labels (id, workspace_id, project_id, issue_id, label_id, created_by_id, updated_by_id, created_at, updated_at)
SELECT unnest(sqlc.arg(ids)::uuid[]), sqlc.arg(workspace_id), sqlc.arg(project_id), sqlc.arg(issue_id),
       unnest(sqlc.arg(label_ids)::uuid[]), sqlc.arg(by)::uuid, sqlc.arg(by), sqlc.arg(at)::timestamptz, sqlc.arg(at);

-- name: InsertSubscribers :exec
-- A work item's new subscribers' rows, one statement, as InsertAssignees; an account that already subscribes to it, or
-- that the accounts name twice, is subscribed once: the partial unique key skips the row (M4 design 3.12).
INSERT INTO issue_subscribers (id, workspace_id, project_id, issue_id, subscriber_id, created_by_id, updated_by_id, created_at,
                               updated_at)
SELECT unnest(sqlc.arg(ids)::uuid[]), sqlc.arg(workspace_id), sqlc.arg(project_id), sqlc.arg(issue_id),
       unnest(sqlc.arg(subscriber_ids)::uuid[]), sqlc.arg(by)::uuid, sqlc.arg(by), sqlc.arg(at)::timestamptz, sqlc.arg(at)
ON CONFLICT (issue_id, subscriber_id) WHERE deleted_at IS NULL DO NOTHING;

-- name: InsertActivity :exec
-- A row of a work item's activity (M4 design 3.13): its actor, who also creates it, at the moment given.
INSERT INTO issue_activities (id, workspace_id, project_id, issue_id, verb, field, old_value, new_value, old_identifier,
                              new_identifier, comment, actor_id, created_by_id, updated_by_id, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(project_id), sqlc.arg(issue_id), sqlc.arg(verb), sqlc.narg(field),
        sqlc.narg(old_value), sqlc.narg(new_value), sqlc.narg(old_identifier), sqlc.narg(new_identifier), sqlc.arg(comment),
        sqlc.arg(actor_id)::uuid, sqlc.arg(actor_id), sqlc.arg(actor_id), sqlc.arg(at)::timestamptz, sqlc.arg(at));

-- name: GetIssue :one
-- The undeleted work item, archived or not, as the API shows it (M4 design 5.2): with the accounts of its undeleted
-- assignees' rows and the labels of its undeleted labels' rows, in the accounts' and the labels' id order, NULL for
-- none; an assignee who left the project is still one (3.6).
SELECT i.id, i.workspace_id, i.project_id, i.sequence_id, i.name, i.description_html, i.description_stripped,
       i.state_id, i.priority, i.start_date, i.target_date, i.parent_id, i.sort_order, i.completed_at, i.archived_at, i.created_by_id, i.updated_by_id, i.created_at, i.updated_at,
       (SELECT array_agg(a.assignee_id ORDER BY a.assignee_id) FROM issue_assignees a
        WHERE a.issue_id = i.id AND a.deleted_at IS NULL)::uuid[] AS assignee_ids,
       (SELECT array_agg(l.label_id ORDER BY l.label_id) FROM issue_labels l
        WHERE l.issue_id = i.id AND l.deleted_at IS NULL)::uuid[] AS label_ids
FROM issues i
WHERE i.id = sqlc.arg(id) AND i.deleted_at IS NULL;

-- name: GetIssues :many
-- The undeleted work items among ids, archived or not, each as GetIssue reads it, in id order (M4 design 3.4, 3.9), in a
-- write's transaction, under its project's lock, or without a lock.
SELECT i.id, i.workspace_id, i.project_id, i.sequence_id, i.name, i.description_html, i.description_stripped,
       i.state_id, i.priority, i.start_date, i.target_date, i.parent_id, i.sort_order, i.completed_at, i.archived_at, i.created_by_id, i.updated_by_id, i.created_at, i.updated_at,
       (SELECT array_agg(a.assignee_id ORDER BY a.assignee_id) FROM issue_assignees a
        WHERE a.issue_id = i.id AND a.deleted_at IS NULL)::uuid[] AS assignee_ids,
       (SELECT array_agg(l.label_id ORDER BY l.label_id) FROM issue_labels l
        WHERE l.issue_id = i.id AND l.deleted_at IS NULL)::uuid[] AS label_ids
FROM issues i
WHERE i.id = ANY (sqlc.arg(ids)::uuid[]) AND i.deleted_at IS NULL
ORDER BY i.id;

-- name: IssueByNumber :one
-- The id of the project's undeleted work item, archived or not, of the number: getIssueByIdentifier's (M4 design 3.7),
-- read without a lock. A number is its project's once (issues_project_id_sequence_id_key), deleted work items' too.
SELECT id
FROM issues
WHERE project_id = sqlc.arg(project_id) AND sequence_id = sqlc.arg(sequence_id) AND deleted_at IS NULL;
