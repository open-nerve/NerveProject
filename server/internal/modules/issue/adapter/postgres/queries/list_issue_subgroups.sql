-- name: SubgroupIssues :many
-- Directories are paired arrays, already scoped in the same read snapshot.
-- Only the issue module's tables are read. All filters precede pagination.
WITH projects AS (
    SELECT * FROM ROWS FROM(unnest(sqlc.arg(project_ids)::uuid[]), unnest(sqlc.arg(project_own)::boolean[])) AS p(id, own_only)
), states AS (
    SELECT * FROM ROWS FROM(unnest(sqlc.arg(state_ids)::uuid[]), unnest(sqlc.arg(state_groups)::text[]), unnest(sqlc.arg(state_keys)::text[])) AS s(id, state_group, name_key)
), labels AS (
    SELECT * FROM ROWS FROM(unnest(sqlc.arg(label_ids)::uuid[]), unnest(sqlc.arg(label_keys)::text[])) AS l(id, name_key)
), assignees AS (
    SELECT * FROM ROWS FROM(unnest(sqlc.arg(assignee_ids)::uuid[]), unnest(sqlc.arg(assignee_keys)::text[])) AS a(id, name_key)
), filtered AS MATERIALIZED (
    SELECT i.id, p.own_only, s.name_key AS state_key
    FROM issues i JOIN projects p ON p.id = i.project_id JOIN states s ON s.id = i.state_id
    WHERE i.deleted_at IS NULL AND (i.archived_at IS NOT NULL) = sqlc.arg(archived)::boolean
      AND (NOT p.own_only OR i.created_by_id = sqlc.arg(actor_id)::uuid)
      AND (sqlc.arg(sub_issues)::boolean OR i.parent_id IS NULL)
      AND (coalesce(cardinality(sqlc.arg(filter_projects)::uuid[]),0)=0 OR i.project_id=ANY(sqlc.arg(filter_projects)::uuid[]))
      AND (coalesce(cardinality(sqlc.arg(filter_states)::uuid[]),0)=0 OR i.state_id=ANY(sqlc.arg(filter_states)::uuid[]))
      AND (coalesce(cardinality(sqlc.arg(filter_groups)::text[]),0)=0 OR s.state_group=ANY(sqlc.arg(filter_groups)::text[]))
      AND (coalesce(cardinality(sqlc.arg(filter_priorities)::text[]),0)=0 OR i.priority=ANY(sqlc.arg(filter_priorities)::text[]))
      AND (coalesce(cardinality(sqlc.arg(filter_creators)::uuid[]),0)=0 OR i.created_by_id=ANY(sqlc.arg(filter_creators)::uuid[]))
      AND (coalesce(cardinality(sqlc.arg(filter_mentions)::uuid[]),0)=0 OR EXISTS(SELECT 1 FROM issue_mentions m WHERE m.issue_id=i.id AND m.deleted_at IS NULL AND m.mention_id=ANY(sqlc.arg(filter_mentions)::uuid[])))
      AND (coalesce(cardinality(sqlc.arg(filter_subscribers)::uuid[]),0)=0 OR EXISTS(SELECT 1 FROM issue_subscribers s WHERE s.issue_id=i.id AND s.deleted_at IS NULL AND s.subscriber_id=ANY(sqlc.arg(filter_subscribers)::uuid[])))
      AND ((coalesce(cardinality(sqlc.arg(filter_labels)::uuid[]),0)=0 AND NOT sqlc.arg(labels_none)::boolean)
           OR EXISTS(SELECT 1 FROM issue_labels l WHERE l.issue_id=i.id AND l.deleted_at IS NULL AND l.label_id=ANY(sqlc.arg(filter_labels)::uuid[]))
           OR (sqlc.arg(labels_none)::boolean AND NOT EXISTS(SELECT 1 FROM issue_labels l WHERE l.issue_id=i.id AND l.deleted_at IS NULL)))
      AND ((coalesce(cardinality(sqlc.arg(filter_assignees)::uuid[]),0)=0 AND NOT sqlc.arg(assignees_none)::boolean)
           OR EXISTS(SELECT 1 FROM issue_assignees a WHERE a.issue_id=i.id AND a.deleted_at IS NULL AND a.assignee_id=ANY(sqlc.arg(filter_assignees)::uuid[]))
           OR (sqlc.arg(assignees_none)::boolean AND NOT EXISTS(SELECT 1 FROM issue_assignees a WHERE a.issue_id=i.id AND a.deleted_at IS NULL)))
      AND NOT EXISTS(SELECT 1 FROM ROWS FROM(unnest(sqlc.arg(start_none)::boolean[]), unnest(sqlc.arg(start_from)::text[]), unnest(sqlc.arg(start_to)::text[])) AS d(none, lo, hi)
                     WHERE NOT (CASE WHEN d.none THEN i.start_date IS NULL ELSE i.start_date IS NOT NULL AND (d.lo='' OR i.start_date>=NULLIF(d.lo,'')::date) AND (d.hi='' OR i.start_date<=NULLIF(d.hi,'')::date) END))
      AND NOT EXISTS(SELECT 1 FROM ROWS FROM(unnest(sqlc.arg(target_none)::boolean[]), unnest(sqlc.arg(target_from)::text[]), unnest(sqlc.arg(target_to)::text[])) AS d(none, lo, hi)
                     WHERE NOT (CASE WHEN d.none THEN i.target_date IS NULL ELSE i.target_date IS NOT NULL AND (d.lo='' OR i.target_date>=NULLIF(d.lo,'')::date) AND (d.hi='' OR i.target_date<=NULLIF(d.hi,'')::date) END))
      AND (sqlc.arg(relation)::text = ''
           OR (sqlc.arg(relation)::text='created' AND i.created_by_id=sqlc.narg(relation_user_id)::uuid)
           OR (sqlc.arg(relation)::text='assigned' AND EXISTS(SELECT 1 FROM issue_assignees a WHERE a.issue_id=i.id AND a.deleted_at IS NULL AND a.assignee_id=sqlc.narg(relation_user_id)::uuid))
           OR (sqlc.arg(relation)::text='subscribed' AND EXISTS(SELECT 1 FROM issue_subscribers s WHERE s.issue_id=i.id AND s.deleted_at IS NULL AND s.subscriber_id=sqlc.narg(relation_user_id)::uuid)))
), counted AS (
    SELECT f.*, count(*) OVER () AS total FROM filtered f
), source_keys AS (
    SELECT f.id, f.own_only, f.total, i.created_at, i.updated_at, i.start_date, i.target_date, i.priority,
           CASE WHEN sqlc.arg(order_key)::text='sort_order' THEN pg_catalog.float8send(i.sort_order) END AS bits,
           CASE WHEN sqlc.arg(order_key)::text='sub_issues_count' THEN
               (SELECT count(*) FROM issues c WHERE c.parent_id=i.id AND c.deleted_at IS NULL AND c.archived_at IS NULL
                AND (NOT f.own_only OR c.created_by_id=sqlc.arg(actor_id)::uuid)) END AS child_count,
           CASE sqlc.arg(order_key)::text
             WHEN 'state_name' THEN f.state_key
             WHEN 'label_name' THEN (SELECT min(l.name_key COLLATE "C") FROM issue_labels r JOIN labels l ON l.id=r.label_id WHERE r.issue_id=i.id AND r.deleted_at IS NULL)
             WHEN 'assignee_name' THEN (SELECT min(a.name_key COLLATE "C") FROM issue_assignees r JOIN assignees a ON a.id=r.assignee_id WHERE r.issue_id=i.id AND r.deleted_at IS NULL)
           END AS k_text
    FROM counted f JOIN issues i ON i.id=f.id
), numeric_keys AS (
    SELECT s.id, s.own_only, s.total, s.child_count, s.k_text,
           CASE sqlc.arg(order_key)::text
             WHEN 'sort_order' THEN ((CASE WHEN get_byte(s.bits,0)>=128 THEN -1 ELSE 1 END)::bigint *
                 ((get_byte(s.bits,0)&127)::bigint*72057594037927936::bigint + get_byte(s.bits,1)::bigint*281474976710656::bigint
                 + get_byte(s.bits,2)::bigint*1099511627776::bigint + get_byte(s.bits,3)::bigint*4294967296::bigint
                 + get_byte(s.bits,4)::bigint*16777216::bigint + get_byte(s.bits,5)::bigint*65536::bigint
                 + get_byte(s.bits,6)::bigint*256::bigint + get_byte(s.bits,7)::bigint))::numeric
             WHEN 'created_at' THEN extract(epoch FROM s.created_at)*1000000::numeric
             WHEN 'updated_at' THEN extract(epoch FROM s.updated_at)*1000000::numeric
             WHEN 'start_date' THEN (s.start_date-DATE '1970-01-01')::numeric
             WHEN 'target_date' THEN (s.target_date-DATE '1970-01-01')::numeric
             WHEN 'priority' THEN (CASE s.priority WHEN 'urgent' THEN 4 WHEN 'high' THEN 3 WHEN 'medium' THEN 2 WHEN 'low' THEN 1 ELSE 0 END)::numeric
             WHEN 'sub_issues_count' THEN s.child_count::numeric
           END AS k_num
    FROM source_keys s
), keyed AS (
    SELECT n.*, (n.k_num IS NULL AND n.k_text IS NULL) AS k_null FROM numeric_keys n
), expanded AS (
    SELECT k.*, coalesce(g.key,'none')::text AS group_key
    FROM keyed k JOIN issues i ON i.id=k.id JOIN states s ON s.id=i.state_id
    LEFT JOIN LATERAL unnest(CASE sqlc.arg(group_by)::text
        WHEN 'label_id' THEN ARRAY(SELECT r.label_id::text FROM issue_labels r WHERE r.issue_id=i.id AND r.deleted_at IS NULL)
        WHEN 'assignee_id' THEN ARRAY(SELECT r.assignee_id::text FROM issue_assignees r WHERE r.issue_id=i.id AND r.deleted_at IS NULL)
        WHEN 'state_id' THEN ARRAY[i.state_id::text]
        WHEN 'state_group' THEN ARRAY[s.state_group]
        WHEN 'priority' THEN ARRAY[i.priority::text]
        WHEN 'created_by_id' THEN ARRAY[i.created_by_id::text]
        WHEN 'target_date' THEN ARRAY[to_char(i.target_date,'YYYY-MM-DD')]
        WHEN 'project_id' THEN ARRAY[i.project_id::text]
    END) AS g(key) ON true
), group_counts AS (
    SELECT e.*, count(*) OVER (PARTITION BY e.group_key) AS group_total FROM expanded e
), subgroup_expanded AS (
    SELECT k.id, coalesce(g.key,'none')::text AS sub_group_key
    FROM keyed k JOIN issues i ON i.id=k.id JOIN states s ON s.id=i.state_id
    LEFT JOIN LATERAL unnest(CASE sqlc.arg(sub_group_by)::text
        WHEN 'label_id' THEN ARRAY(SELECT r.label_id::text FROM issue_labels r WHERE r.issue_id=i.id AND r.deleted_at IS NULL)
        WHEN 'assignee_id' THEN ARRAY(SELECT r.assignee_id::text FROM issue_assignees r WHERE r.issue_id=i.id AND r.deleted_at IS NULL)
        WHEN 'state_id' THEN ARRAY[i.state_id::text]
        WHEN 'state_group' THEN ARRAY[s.state_group]
        WHEN 'priority' THEN ARRAY[i.priority::text]
        WHEN 'created_by_id' THEN ARRAY[i.created_by_id::text]
        WHEN 'target_date' THEN ARRAY[to_char(i.target_date,'YYYY-MM-DD')]
    END) AS g(key) ON true
), subgroup_counts AS (
    SELECT e.*, count(*) OVER (PARTITION BY e.sub_group_key) AS subgroup_total FROM subgroup_expanded e
), cells AS (
    SELECT g.*, s.sub_group_key, s.subgroup_total,
           count(*) OVER (PARTITION BY g.group_key, s.sub_group_key) AS cell_total
    FROM group_counts g JOIN subgroup_counts s ON s.id=g.id
), ranked AS (
    SELECT k.*, row_number() OVER (PARTITION BY k.group_key, k.sub_group_key
        ORDER BY k.k_null,
         CASE WHEN NOT sqlc.arg(descending)::boolean THEN k.k_num END ASC,
         CASE WHEN sqlc.arg(descending)::boolean THEN k.k_num END DESC,
         CASE WHEN NOT sqlc.arg(descending)::boolean THEN k.k_text END COLLATE "C" ASC,
         CASE WHEN sqlc.arg(descending)::boolean THEN k.k_text END COLLATE "C" DESC,
         CASE WHEN NOT sqlc.arg(descending)::boolean THEN k.id END ASC,
         CASE WHEN sqlc.arg(descending)::boolean THEN k.id END DESC) AS row_number
    FROM cells k
    WHERE (sqlc.narg(group_key)::text IS NULL OR k.group_key=sqlc.narg(group_key)::text)
      AND (sqlc.narg(sub_group_key)::text IS NULL OR k.sub_group_key=sqlc.narg(sub_group_key)::text)
      AND (NOT sqlc.arg(has_cursor)::boolean OR
      (k.k_null AND NOT sqlc.arg(cursor_null)::boolean) OR
      (k.k_null=sqlc.arg(cursor_null)::boolean AND
       (CASE WHEN sqlc.arg(descending)::boolean THEN
           (k.k_num<sqlc.narg(cursor_num)::bigint::numeric OR k.k_text COLLATE "C"<sqlc.narg(cursor_text)::text COLLATE "C")
         ELSE (k.k_num>sqlc.narg(cursor_num)::bigint::numeric OR k.k_text COLLATE "C">sqlc.narg(cursor_text)::text COLLATE "C") END
        OR ((k.k_null OR k.k_num=sqlc.narg(cursor_num)::bigint::numeric OR k.k_text COLLATE "C"=sqlc.narg(cursor_text)::text COLLATE "C")
            AND CASE WHEN sqlc.arg(descending)::boolean THEN k.id<sqlc.narg(cursor_id)::uuid ELSE k.id>sqlc.narg(cursor_id)::uuid END))))
)
SELECT i.id, i.workspace_id, i.project_id, i.sequence_id, i.name, i.state_id, i.priority,
       i.start_date, i.target_date, i.parent_id, i.sort_order, i.completed_at, i.archived_at,
       i.created_by_id, i.updated_by_id, i.created_at, i.updated_at, k.total::bigint AS total,
       k.group_key::text AS group_key, k.group_total::bigint AS group_total,
       k.sub_group_key::text AS sub_group_key, k.subgroup_total::bigint AS subgroup_total, k.cell_total::bigint AS cell_total,
       coalesce(k.k_text,'')::text AS name_key, (k.k_text IS NULL)::boolean AS name_null,
       coalesce(k.child_count,(SELECT count(*) FROM issues c WHERE c.parent_id=i.id AND c.deleted_at IS NULL AND c.archived_at IS NULL
                              AND (NOT k.own_only OR c.created_by_id=sqlc.arg(actor_id)::uuid)))::bigint AS sub_issues_count,
       coalesce((SELECT array_agg(a.assignee_id ORDER BY a.assignee_id) FROM issue_assignees a WHERE a.issue_id=i.id AND a.deleted_at IS NULL),'{}'::uuid[])::uuid[] AS assignee_ids,
       coalesce((SELECT array_agg(l.label_id ORDER BY l.label_id) FROM issue_labels l WHERE l.issue_id=i.id AND l.deleted_at IS NULL),'{}'::uuid[])::uuid[] AS label_ids
FROM ranked k JOIN issues i ON i.id=k.id
WHERE k.row_number <= sqlc.arg(page_size)::integer
ORDER BY (k.group_key='none'),
         CASE sqlc.arg(group_by)::text
           WHEN 'priority' THEN CASE k.group_key WHEN 'urgent' THEN 0 WHEN 'high' THEN 1 WHEN 'medium' THEN 2 WHEN 'low' THEN 3 ELSE 4 END
           WHEN 'state_group' THEN CASE k.group_key WHEN 'backlog' THEN 0 WHEN 'unstarted' THEN 1 WHEN 'started' THEN 2 WHEN 'completed' THEN 3 ELSE 4 END
           ELSE 0 END,
         k.group_key COLLATE "C", (k.sub_group_key='none'),
         CASE sqlc.arg(sub_group_by)::text
           WHEN 'priority' THEN CASE k.sub_group_key WHEN 'urgent' THEN 0 WHEN 'high' THEN 1 WHEN 'medium' THEN 2 WHEN 'low' THEN 3 ELSE 4 END
           WHEN 'state_group' THEN CASE k.sub_group_key WHEN 'backlog' THEN 0 WHEN 'unstarted' THEN 1 WHEN 'started' THEN 2 WHEN 'completed' THEN 3 ELSE 4 END
           ELSE 0 END,
         k.sub_group_key COLLATE "C", k.row_number;
