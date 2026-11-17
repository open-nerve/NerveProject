-- name: ProjectFacts :one
-- access's ProjectAccess (M3 design 6.5): the undeleted project, archived or not, whether it shows its guests every
-- work item (M4 design 3.4), and the user's active membership of it, if any.
SELECT p.workspace_id, p.network, p.guest_view_all_features, m.role AS member_role
FROM projects p
LEFT JOIN project_members m
       ON m.project_id = p.id AND m.member_id = sqlc.arg(user_id) AND m.is_active AND m.deleted_at IS NULL
WHERE p.id = sqlc.arg(project_id) AND p.deleted_at IS NULL;

-- name: ProjectFactsMany :many
-- ProjectFacts for each of the accounts ids at once, access's DecideMany (M7 P1b spec 2.7): the undeleted project,
-- archived or not, once per active membership of it among the accounts, in member order, or once with no member when
-- none of them has one; no row for a project that does not exist or is deleted.
SELECT p.workspace_id, p.network, p.guest_view_all_features, m.member_id, m.role AS member_role
FROM projects p
LEFT JOIN project_members m
       ON m.project_id = p.id AND m.member_id = ANY (sqlc.arg(ids)::uuid[]) AND m.is_active AND m.deleted_at IS NULL
WHERE p.id = sqlc.arg(project_id) AND p.deleted_at IS NULL
ORDER BY m.member_id;
