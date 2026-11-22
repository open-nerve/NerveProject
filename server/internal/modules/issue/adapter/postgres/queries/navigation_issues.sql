-- name: NavigationIssuePlaces :many
-- Where the undeleted work items among ids are (M7 P3d spec 2.7, R3): each one's workspace and project, archived or
-- not, another workspace's too. A recording of a recent visit reads it before its locks, to know the workspace it
-- decides on and the project it locks FOR SHARE, and again under them. A deleted work item and an id of none read no
-- row. A plain read of what is committed, without a lock, in id order; the primary key serves it.
SELECT id, workspace_id, project_id
FROM issues
WHERE id = ANY (sqlc.arg(ids)::uuid[]) AND deleted_at IS NULL
ORDER BY id;

-- name: NavigationIssueRows :many
-- The undeleted work items among ids, archived or not, of any project, each with what a recent visit is answered with
-- now (spec 2.4, 2.5, R3): its project, its number, its name, its state by id, its priority and its current assignees,
-- the accounts of its undeleted assignees' rows (GetIssue's), each once by the partial unique key on (issue_id,
-- assignee_id), in id order, none for none; its creator, NULL when not known, which the guest rule asks; and its
-- undeleted intake row's status, NULL for none, which InTriage asks: the condition is in the ON, so the join stays a
-- left join and intake_issues_issue_id_key, one undeleted row a work item at most, serves it (GetIssue's). The module
-- answers those a caller sees now (NavigationIssues), so the statement reads any project's, another workspace's too. A
-- plain read of what is committed, without a lock, in id order; the primary key serves it.
SELECT i.id, i.project_id, i.sequence_id, i.name, i.state_id, i.priority, i.created_by_id,
       array(SELECT a.assignee_id FROM issue_assignees a WHERE a.issue_id = i.id AND a.deleted_at IS NULL
             ORDER BY a.assignee_id)::uuid[] AS assignee_ids,
       ii.status AS intake_status
FROM issues i LEFT JOIN intake_issues ii ON ii.issue_id = i.id AND ii.deleted_at IS NULL
WHERE i.id = ANY (sqlc.arg(ids)::uuid[]) AND i.deleted_at IS NULL
ORDER BY i.id;
