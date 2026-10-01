-- name: CreateProject :exec
-- createProject (M3 design 3.6): the audit columns come from the use case's clock (M2 design 3.13); the columns the
-- insert does not name take their defaults.
INSERT INTO projects (id, workspace_id, name, description, identifier, network, project_lead_id, logo_props, timezone,
                      created_by_id, updated_by_id, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(name), sqlc.arg(description), sqlc.arg(identifier), sqlc.arg(network),
        sqlc.narg(project_lead_id), sqlc.arg(logo_props), sqlc.arg(timezone), sqlc.arg(created_by), sqlc.arg(created_by),
        sqlc.arg(now), sqlc.arg(now));

-- name: IdentifierTaken :one
-- checkProjectIdentifier: whether an undeleted project of the workspace has the identifier.
SELECT EXISTS (SELECT 1 FROM projects
               WHERE workspace_id = sqlc.arg(workspace_id) AND identifier = sqlc.arg(identifier) AND deleted_at IS NULL);

-- name: GetProject :one
-- The undeleted project, archived or not, as the user sees it (M3 design 3.19, 5.2): his project role and his place in
-- his sidebar while his membership is active, null otherwise; and the active members' accounts, in the order they
-- became members (3.12).
SELECT p.id, p.workspace_id, p.name, p.description, p.identifier, p.network, p.project_lead_id, p.default_assignee_id,
       p.cycle_view, p.module_view, p.issue_views_view, p.intake_view, p.guest_view_all_features, p.archive_in,
       p.archived_at, p.logo_props, p.timezone, p.created_at, p.updated_at, m.role AS member_role, u.sort_order,
       ARRAY(SELECT a.member_id FROM project_members a
             WHERE a.project_id = p.id AND a.is_active AND a.deleted_at IS NULL
             ORDER BY a.created_at, a.id)::uuid[] AS member_ids
FROM projects p
LEFT JOIN project_members m
       ON m.project_id = p.id AND m.member_id = sqlc.arg(user_id) AND m.is_active AND m.deleted_at IS NULL
LEFT JOIN project_user_properties u
       ON u.project_id = m.project_id AND u.user_id = m.member_id AND u.deleted_at IS NULL
WHERE p.id = sqlc.arg(id) AND p.deleted_at IS NULL;

-- name: ListProjects :many
-- listProjects (M3 design 3.4, 3.12, 3.19): the workspace's undeleted projects that the user sees, the archived ones or
-- the others, each as GetProject reads it (the same columns, so the rows convert); sees_all and sees_public are the
-- user's workspace role's domain.Visibility. By the project's place in the user's sidebar, the projects without a
-- place in it last, then by name, which is unique among the workspace's undeleted projects.
SELECT p.id, p.workspace_id, p.name, p.description, p.identifier, p.network, p.project_lead_id, p.default_assignee_id,
       p.cycle_view, p.module_view, p.issue_views_view, p.intake_view, p.guest_view_all_features, p.archive_in,
       p.archived_at, p.logo_props, p.timezone, p.created_at, p.updated_at, m.role AS member_role, u.sort_order,
       ARRAY(SELECT a.member_id FROM project_members a
             WHERE a.project_id = p.id AND a.is_active AND a.deleted_at IS NULL
             ORDER BY a.created_at, a.id)::uuid[] AS member_ids
FROM projects p
LEFT JOIN project_members m
       ON m.project_id = p.id AND m.member_id = sqlc.arg(user_id) AND m.is_active AND m.deleted_at IS NULL
LEFT JOIN project_user_properties u
       ON u.project_id = m.project_id AND u.user_id = m.member_id AND u.deleted_at IS NULL
WHERE p.workspace_id = sqlc.arg(workspace_id) AND p.deleted_at IS NULL
  AND (p.archived_at IS NOT NULL) = sqlc.arg(archived)::boolean
  AND (sqlc.arg(sees_all)::boolean OR m.id IS NOT NULL OR (sqlc.arg(sees_public)::boolean AND p.network = 2))
ORDER BY u.sort_order NULLS LAST, p.name;

-- name: LockProject :one
-- The parent lock of a write that changes the project row or its memberships (M3 design 3.6 convention 2): FOR NO KEY
-- UPDATE waits for another FOR NO KEY UPDATE and for FOR SHARE, not for a foreign key's FOR KEY SHARE. After a wait,
-- Postgres evaluates deleted_at IS NULL again on the row's newest version, so a project deleted meanwhile reads no row.
SELECT workspace_id, (archived_at IS NOT NULL)::boolean AS archived
FROM projects
WHERE id = sqlc.arg(id) AND deleted_at IS NULL
FOR NO KEY UPDATE;

-- name: ProjectWorkspace :one
-- The workspace of the undeleted project, archived or not, without a lock: what a read decides on (M3 design 6.4), and
-- what a write on the project reads first, to lock the workspace before the project (3.6 convention 2).
SELECT workspace_id
FROM projects
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;

-- name: UpdateProject :exec
-- updateProject, under the project's FOR NO KEY UPDATE (M3 design 5.2): a field left out, null here, keeps its value;
-- the lead and the default assignee change when their flags are set, to null too.
UPDATE projects p
SET name                    = coalesce(sqlc.narg(name)::text, p.name),
    description             = coalesce(sqlc.narg(description)::text, p.description),
    identifier              = coalesce(sqlc.narg(identifier)::text, p.identifier),
    network                 = coalesce(sqlc.narg(network)::smallint, p.network),
    project_lead_id         = CASE WHEN sqlc.arg(set_lead)::boolean THEN sqlc.narg(project_lead_id)::uuid
                                   ELSE p.project_lead_id END,
    default_assignee_id     = CASE WHEN sqlc.arg(set_default_assignee)::boolean THEN sqlc.narg(default_assignee_id)::uuid
                                   ELSE p.default_assignee_id END,
    cycle_view              = coalesce(sqlc.narg(cycle_view)::boolean, p.cycle_view),
    module_view             = coalesce(sqlc.narg(module_view)::boolean, p.module_view),
    issue_views_view        = coalesce(sqlc.narg(issue_views_view)::boolean, p.issue_views_view),
    intake_view             = coalesce(sqlc.narg(intake_view)::boolean, p.intake_view),
    guest_view_all_features = coalesce(sqlc.narg(guest_view_all_features)::boolean, p.guest_view_all_features),
    archive_in              = coalesce(sqlc.narg(archive_in)::integer, p.archive_in),
    logo_props              = coalesce(sqlc.narg(logo_props)::jsonb, p.logo_props),
    timezone                = coalesce(sqlc.narg(timezone)::text, p.timezone),
    updated_by_id           = sqlc.arg(updated_by)::uuid,
    updated_at              = sqlc.arg(now)
WHERE p.id = sqlc.arg(id);
