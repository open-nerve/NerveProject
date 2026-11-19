-- name: ListIntakeIssues :many
-- listIntakeIssues' page and the row after it (M7 P2b spec 2.3), in the request's read transaction: the project's
-- undeleted work items with an undeleted intake row, accepted and archived ones too (intake_issues_project_id_status_idx
-- finds the rows). A caller who sees only his own work items lists those he created alone, an unknown creator none, the
-- registered copy of the guest rule (list_issues.sql). A status is the row's effective one at the request's moment, a
-- snooze whose time has come pending (R3); the two time ranges are half-open instants the use case gives, so no day is
-- taken in the session's time zone (R8). All filters precede the total and the cursor. The order's key, the work item's
-- created_at or updated_at in microseconds or its number, then its id, the same way. Each row as a list row (FlatIssues),
-- its sub-items counted as the project list counts them, and its intake row as stored.
WITH filtered AS MATERIALIZED (
    SELECT i.id, i.created_at, i.updated_at, i.sequence_id
    FROM intake_issues ii JOIN issues i ON i.id = ii.issue_id AND i.deleted_at IS NULL
    WHERE ii.project_id = sqlc.arg(project_id)::uuid AND ii.deleted_at IS NULL
      AND (NOT sqlc.arg(own_only)::boolean OR i.created_by_id = sqlc.arg(actor_id)::uuid)
      AND (coalesce(cardinality(sqlc.arg(statuses)::text[]),0)=0
           OR (CASE WHEN ii.status = 'snoozed' AND ii.snoozed_till <= sqlc.arg(now)::timestamptz THEN 'pending' ELSE ii.status END)
              = ANY(sqlc.arg(statuses)::text[]))
      AND (coalesce(cardinality(sqlc.arg(priorities)::text[]),0)=0 OR i.priority=ANY(sqlc.arg(priorities)::text[]))
      AND (coalesce(cardinality(sqlc.arg(creators)::uuid[]),0)=0 OR i.created_by_id=ANY(sqlc.arg(creators)::uuid[]))
      AND ((coalesce(cardinality(sqlc.arg(labels)::uuid[]),0)=0 AND NOT sqlc.arg(labels_none)::boolean)
           OR EXISTS(SELECT 1 FROM issue_labels l WHERE l.issue_id=i.id AND l.deleted_at IS NULL AND l.label_id=ANY(sqlc.arg(labels)::uuid[]))
           OR (sqlc.arg(labels_none)::boolean AND NOT EXISTS(SELECT 1 FROM issue_labels l WHERE l.issue_id=i.id AND l.deleted_at IS NULL)))
      AND ((coalesce(cardinality(sqlc.arg(assignees)::uuid[]),0)=0 AND NOT sqlc.arg(assignees_none)::boolean)
           OR EXISTS(SELECT 1 FROM issue_assignees a WHERE a.issue_id=i.id AND a.deleted_at IS NULL AND a.assignee_id=ANY(sqlc.arg(assignees)::uuid[]))
           OR (sqlc.arg(assignees_none)::boolean AND NOT EXISTS(SELECT 1 FROM issue_assignees a WHERE a.issue_id=i.id AND a.deleted_at IS NULL)))
      AND (sqlc.narg(created_from)::timestamptz IS NULL OR i.created_at >= sqlc.narg(created_from)::timestamptz)
      AND (sqlc.narg(created_to)::timestamptz IS NULL OR i.created_at < sqlc.narg(created_to)::timestamptz)
      AND (sqlc.narg(updated_from)::timestamptz IS NULL OR i.updated_at >= sqlc.narg(updated_from)::timestamptz)
      AND (sqlc.narg(updated_to)::timestamptz IS NULL OR i.updated_at < sqlc.narg(updated_to)::timestamptz)
), keyed AS (
    SELECT f.id, count(*) OVER () AS total,
           CASE sqlc.arg(order_key)::text
             WHEN 'created_at' THEN extract(epoch FROM f.created_at)*1000000::numeric
             WHEN 'updated_at' THEN extract(epoch FROM f.updated_at)*1000000::numeric
             WHEN 'sequence_id' THEN f.sequence_id::numeric
           END AS k
    FROM filtered f
), ranked AS (
    SELECT k.id, k.total, row_number() OVER (
        ORDER BY CASE WHEN NOT sqlc.arg(descending)::boolean THEN k.k END ASC,
                 CASE WHEN sqlc.arg(descending)::boolean THEN k.k END DESC,
                 CASE WHEN NOT sqlc.arg(descending)::boolean THEN k.id END ASC,
                 CASE WHEN sqlc.arg(descending)::boolean THEN k.id END DESC) AS page_position
    FROM keyed k
    WHERE NOT sqlc.arg(has_cursor)::boolean OR
          (CASE WHEN sqlc.arg(descending)::boolean
             THEN k.k < sqlc.narg(cursor_key)::bigint::numeric OR (k.k = sqlc.narg(cursor_key)::bigint::numeric AND k.id < sqlc.narg(cursor_id)::uuid)
             ELSE k.k > sqlc.narg(cursor_key)::bigint::numeric OR (k.k = sqlc.narg(cursor_key)::bigint::numeric AND k.id > sqlc.narg(cursor_id)::uuid) END)
), paged AS MATERIALIZED (
    SELECT * FROM ranked WHERE page_position <= sqlc.arg(page_size)::integer
)
SELECT i.id, i.workspace_id, i.project_id, i.sequence_id, i.name, i.state_id, i.priority,
       i.start_date, i.target_date, i.parent_id, i.sort_order, i.completed_at, i.archived_at,
       i.created_by_id, i.updated_by_id, i.created_at, i.updated_at, p.total::bigint AS total,
       (SELECT count(*) FROM issues c WHERE c.parent_id=i.id AND c.deleted_at IS NULL AND c.archived_at IS NULL
        AND (NOT sqlc.arg(own_only)::boolean OR c.created_by_id=sqlc.arg(actor_id)::uuid))::bigint AS sub_issues_count,
       (SELECT count(*) FROM issue_links l WHERE l.issue_id=i.id AND l.deleted_at IS NULL)::bigint AS link_count,
       coalesce((SELECT array_agg(a.assignee_id ORDER BY a.assignee_id) FROM issue_assignees a WHERE a.issue_id=i.id AND a.deleted_at IS NULL),'{}'::uuid[])::uuid[] AS assignee_ids,
       coalesce((SELECT array_agg(l.label_id ORDER BY l.label_id) FROM issue_labels l WHERE l.issue_id=i.id AND l.deleted_at IS NULL),'{}'::uuid[])::uuid[] AS label_ids,
       ci.cycle_id,
       coalesce((SELECT array_agg(m.module_id ORDER BY m.module_id) FROM module_issues m WHERE m.issue_id=i.id AND m.deleted_at IS NULL),'{}'::uuid[])::uuid[] AS module_ids,
       ii.status AS intake_status, ii.snoozed_till AS intake_snoozed_till, ii.duplicate_to_id AS intake_duplicate_to_id
FROM paged p JOIN issues i ON i.id = p.id
     JOIN intake_issues ii ON ii.issue_id = i.id AND ii.deleted_at IS NULL
     LEFT JOIN cycle_issues ci ON ci.issue_id = i.id AND ci.deleted_at IS NULL
ORDER BY p.page_position;
