-- name: RelatedIssues :many
-- Which of the work items ids relate to the work item id (M4 design 3.10): an undeleted relation between the two, of
-- any type, either way, in the other's id order. Read under the locks of both ends' projects, one of which every write
-- of a relation between them holds at least (3.5, convention 7), it stays as read until the write commits. The indexes
-- on issue_id and on related_issue_id serve it.
SELECT (CASE WHEN r.issue_id = sqlc.arg(id)::uuid THEN r.related_issue_id ELSE r.issue_id END)::uuid AS other
FROM issue_relations r
WHERE r.deleted_at IS NULL
  AND ((r.issue_id = sqlc.arg(id)::uuid AND r.related_issue_id = ANY (sqlc.arg(ids)::uuid[]))
    OR (r.related_issue_id = sqlc.arg(id)::uuid AND r.issue_id = ANY (sqlc.arg(ids)::uuid[])))
ORDER BY 1;

-- name: InsertRelations :exec
-- New relations of work items (M4 design 3.10), one statement: ids[i] the row of the i-th, in the order given (the
-- other work items' id order, convention 5's second half), each with its project, its ends and its type as stored, at
-- the moment and by the account given.
INSERT INTO issue_relations (id, workspace_id, project_id, issue_id, related_issue_id, relation_type, created_by_id, updated_by_id,
                             created_at, updated_at)
SELECT unnest(sqlc.arg(ids)::uuid[]), sqlc.arg(workspace_id), unnest(sqlc.arg(project_ids)::uuid[]),
       unnest(sqlc.arg(issue_ids)::uuid[]), unnest(sqlc.arg(related_issue_ids)::uuid[]), unnest(sqlc.arg(relation_types)::text[]),
       sqlc.arg(by)::uuid, sqlc.arg(by), sqlc.arg(at)::timestamptz, sqlc.arg(at);

-- name: MarkChanged :execrows
-- The work items a relation's addition, or its removal, changes last without a field of theirs (M4 design 3.7, 3.10):
-- both its ends, each last changed at the moment and by the account given; the undeleted ones among ids. Under the
-- locks of their projects, which every write of the projects' work items holds (3.5): one statement in scan order
-- (convention 5).
UPDATE issues
SET updated_by_id = sqlc.arg(by)::uuid, updated_at = sqlc.arg(at)
WHERE id = ANY (sqlc.arg(ids)::uuid[]) AND deleted_at IS NULL;

-- name: IssueRelations :many
-- The undeleted relations of the work item id, either way, each as stored, in the order they were made, (created_at,
-- id) (M4 design 3.10): listIssueRelations', read without a lock. The indexes on issue_id and on related_issue_id
-- serve it.
SELECT r.id, r.workspace_id, r.project_id, r.issue_id, r.related_issue_id, r.relation_type, r.created_by_id, r.created_at
FROM issue_relations r
WHERE (r.issue_id = sqlc.arg(id)::uuid OR r.related_issue_id = sqlc.arg(id)::uuid) AND r.deleted_at IS NULL
ORDER BY r.created_at, r.id;

-- name: GetRelation :one
-- The undeleted relation id, as stored, of the same columns as IssueRelations (M4 design 3.10): deleteIssueRelation's,
-- read without a lock first, then again under the locks of both its work items' projects, one of which every write of
-- it holds at least (3.5, convention 7).
SELECT r.id, r.workspace_id, r.project_id, r.issue_id, r.related_issue_id, r.relation_type, r.created_by_id, r.created_at
FROM issue_relations r
WHERE r.id = sqlc.arg(id)::uuid AND r.deleted_at IS NULL;

-- name: DeleteRelation :execrows
-- deleteIssueRelation's soft deletion of the undeleted relation id, at the moment and by the account given (M4 design
-- 3.10), under the locks of both its work items' projects: one row, by its key.
UPDATE issue_relations
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(by)::uuid
WHERE id = sqlc.arg(id)::uuid AND deleted_at IS NULL;
