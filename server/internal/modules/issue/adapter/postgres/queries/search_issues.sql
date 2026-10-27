-- name: SearchIssues :many
-- Scope, dates and the entire relationship exclusion precede LIMIT. Recursive
-- paths intentionally traverse invisible, archived and deleted intermediate rows.
WITH RECURSIVE projects AS (
    SELECT * FROM ROWS FROM(unnest(sqlc.arg(project_ids)::uuid[]), unnest(sqlc.arg(project_own)::boolean[])) AS p(id, own_only)
), source AS (
    SELECT id, parent_id FROM issues WHERE id=sqlc.narg(source_id)::uuid
), descendants(id) AS (
    SELECT id FROM source WHERE sqlc.arg(picker)::text='parent'
    UNION
    SELECT i.id FROM issues i JOIN descendants d ON i.parent_id=d.id
), ancestors(id) AS (
    SELECT id FROM source WHERE sqlc.arg(picker)::text='sub_issue'
    UNION
    SELECT i.parent_id FROM issues i JOIN ancestors a ON i.id=a.id WHERE i.parent_id IS NOT NULL
), candidates AS (
    SELECT i.id, i.created_at, i.updated_at,
           CASE WHEN i.project_id=sqlc.narg(exact_project_id)::uuid AND i.sequence_id=sqlc.narg(identifier_number)::integer THEN 0
                WHEN i.sequence_id=sqlc.narg(sequence_number)::integer THEN 1 ELSE 2 END AS rank,
           similarity(i.name, @query) AS score
    FROM issues i JOIN projects p ON p.id=i.project_id
    WHERE i.deleted_at IS NULL AND i.archived_at IS NULL
      AND (NOT p.own_only OR i.created_by_id=sqlc.arg(actor_id)::uuid)
      AND (sqlc.arg(query)::text='' OR i.project_id=sqlc.narg(exact_project_id)::uuid AND i.sequence_id=sqlc.narg(identifier_number)::integer
           OR i.sequence_id=sqlc.narg(sequence_number)::integer OR i.name ILIKE sqlc.arg(pattern)::text ESCAPE E'\\')
      AND NOT EXISTS(SELECT 1 FROM ROWS FROM(unnest(sqlc.arg(start_none)::boolean[]), unnest(sqlc.arg(start_from)::text[]), unnest(sqlc.arg(start_to)::text[])) AS d(none, lo, hi)
                     WHERE NOT (CASE WHEN d.none THEN i.start_date IS NULL ELSE i.start_date IS NOT NULL AND (d.lo='' OR i.start_date>=NULLIF(d.lo,'')::date) AND (d.hi='' OR i.start_date<=NULLIF(d.hi,'')::date) END))
      AND NOT EXISTS(SELECT 1 FROM ROWS FROM(unnest(sqlc.arg(target_none)::boolean[]), unnest(sqlc.arg(target_from)::text[]), unnest(sqlc.arg(target_to)::text[])) AS d(none, lo, hi)
                     WHERE NOT (CASE WHEN d.none THEN i.target_date IS NULL ELSE i.target_date IS NOT NULL AND (d.lo='' OR i.target_date>=NULLIF(d.lo,'')::date) AND (d.hi='' OR i.target_date<=NULLIF(d.hi,'')::date) END))
      AND (sqlc.arg(picker)::text='' OR i.id<>sqlc.narg(source_id)::uuid)
      AND (sqlc.arg(picker)::text<>'parent' OR
           (NOT EXISTS(SELECT 1 FROM descendants d WHERE d.id=i.id) AND NOT EXISTS(SELECT 1 FROM source s WHERE s.parent_id=i.id)))
      AND (sqlc.arg(picker)::text<>'sub_issue' OR
           (i.parent_id IS NULL AND NOT EXISTS(SELECT 1 FROM ancestors a WHERE a.id=i.id)))
      AND (sqlc.arg(picker)::text<>'relation' OR NOT EXISTS(
          SELECT 1 FROM issue_relations r WHERE r.deleted_at IS NULL
            AND ((r.issue_id=sqlc.narg(source_id)::uuid AND r.related_issue_id=i.id)
                 OR (r.related_issue_id=sqlc.narg(source_id)::uuid AND r.issue_id=i.id))))
), paged AS MATERIALIZED (
    SELECT * FROM candidates c
    ORDER BY CASE WHEN sqlc.arg(query)::text='' THEN c.updated_at END DESC,
             CASE WHEN sqlc.arg(query)::text<>'' THEN c.rank END,
             CASE WHEN sqlc.arg(query)::text<>'' THEN c.score END DESC,
             CASE WHEN sqlc.arg(query)::text<>'' THEN c.created_at END DESC, c.id DESC
    LIMIT sqlc.arg(page_size)::integer
)
SELECT i.id, i.workspace_id, i.project_id, i.sequence_id, i.name, i.description_html, i.description_stripped,
       i.state_id, i.priority, i.start_date, i.target_date, i.parent_id, i.sort_order, i.completed_at, i.archived_at,
       i.created_by_id, i.updated_by_id, i.created_at, i.updated_at,
       (SELECT array_agg(a.assignee_id ORDER BY a.assignee_id) FROM issue_assignees a WHERE a.issue_id=i.id AND a.deleted_at IS NULL)::uuid[] AS assignee_ids,
       (SELECT array_agg(l.label_id ORDER BY l.label_id) FROM issue_labels l WHERE l.issue_id=i.id AND l.deleted_at IS NULL)::uuid[] AS label_ids,
       (SELECT count(*) FROM issue_links l WHERE l.issue_id=i.id AND l.deleted_at IS NULL)::bigint AS link_count,
       similarity(i.name, @query) AS similarity
FROM paged c JOIN issues i ON i.id=c.id
ORDER BY CASE WHEN sqlc.arg(query)::text='' THEN c.updated_at END DESC,
         CASE WHEN sqlc.arg(query)::text<>'' THEN c.rank END,
         CASE WHEN sqlc.arg(query)::text<>'' THEN c.score END DESC,
         CASE WHEN sqlc.arg(query)::text<>'' THEN c.created_at END DESC, c.id DESC;
