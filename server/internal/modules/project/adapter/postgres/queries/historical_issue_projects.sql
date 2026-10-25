-- name: HistoricalIssueProjects :many
-- Historical references retain active memberships even after project/member
-- soft deletion or project archival. Normal target authorization is separate.
SELECT p.id, pm.role,
       (pm.role = 5 AND NOT p.guest_view_all_features)::boolean AS own_only
FROM projects p
JOIN project_members pm ON pm.project_id = p.id
WHERE p.workspace_id = sqlc.arg(workspace_id)
  AND pm.member_id = sqlc.arg(actor_id)
  AND pm.is_active = true
ORDER BY p.id;
