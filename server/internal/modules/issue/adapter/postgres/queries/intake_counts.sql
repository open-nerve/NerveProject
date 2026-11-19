-- name: CountPendingIntakeIssues :many
-- getIntakeCounts' numbers (M7 P2b spec 2.6), in the request's read transaction: for each project given, each once, in
-- id order, one with none too, its undeleted intake rows of undeleted work items whose effective status at the
-- request's moment is pending, a snooze whose time has come too (R3); of a project where the caller sees only his own
-- work items, those he created alone, an unknown creator none, the registered copy of the guest rule (list_issues.sql).
-- Which projects, those he sees with their intake turned on, is the use case's: the issue module reads no project.
-- intake_issues_project_id_status_idx serves each project's probe of the two statuses.
SELECT p.id::uuid AS project_id, count(i.id)::bigint AS pending
FROM ROWS FROM (unnest(sqlc.arg(project_ids)::uuid[]), unnest(sqlc.arg(project_own)::boolean[])) AS p(id, own_only)
LEFT JOIN intake_issues ii ON ii.project_id = p.id AND ii.deleted_at IS NULL AND ii.status IN ('pending', 'snoozed')
     AND (ii.status = 'pending' OR ii.snoozed_till <= sqlc.arg(now)::timestamptz)
LEFT JOIN issues i ON i.id = ii.issue_id AND i.deleted_at IS NULL
     AND (NOT p.own_only OR i.created_by_id = sqlc.arg(actor_id)::uuid)
GROUP BY p.id
ORDER BY p.id;
