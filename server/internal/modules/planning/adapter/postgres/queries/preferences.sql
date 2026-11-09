-- name: DeleteCyclePreferenceRows :exec
-- A deleted cycle's page's display settings (M6 P3b spec 2.5): each account's undeleted row of it, locked FOR NO KEY
-- UPDATE in id order, then soft-deleted at the moment and by the account given, in one statement (M3 design 3.6,
-- convention 5; as DeleteModuleMemberRows); a row deleted before keeps its deletion. cycle_user_properties_cycle_id_idx
-- serves it.
UPDATE cycle_user_properties AS t
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(by)::uuid
WHERE t.id IN (
    SELECT r.id FROM cycle_user_properties r
    WHERE r.cycle_id = sqlc.arg(cycle_id) AND r.deleted_at IS NULL
    ORDER BY r.id FOR NO KEY UPDATE);

-- name: DeleteModulePreferenceRows :exec
-- DeleteCyclePreferenceRows' statement, of a deleted module's page. module_user_properties_module_id_idx serves it.
UPDATE module_user_properties AS t
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(by)::uuid
WHERE t.id IN (
    SELECT r.id FROM module_user_properties r
    WHERE r.module_id = sqlc.arg(module_id) AND r.deleted_at IS NULL
    ORDER BY r.id FOR NO KEY UPDATE);

-- name: CyclePreferences :one
-- getCyclePreferences (M6 P3b spec 2.3): the account's undeleted display settings on the cycle's page, if any.
-- cycle_user_properties_cycle_id_user_id_key serves it.
SELECT filters, display_filters, display_properties
FROM cycle_user_properties
WHERE cycle_id = sqlc.arg(cycle_id) AND user_id = sqlc.arg(user_id) AND deleted_at IS NULL;

-- name: UpsertCyclePreferences :one
-- updateCyclePreferences, under the project's FOR SHARE (M6 P3b spec 2.3, R6): an account without an undeleted row gets
-- one, with the values given, the defaults with the change applied; one with a row has the fields that are set changed,
-- each whole. The account makes and last changes his own row. The conflict target is the partial unique index, so a
-- deleted row does not count (as updateProjectPreferences).
INSERT INTO cycle_user_properties AS p (id, workspace_id, project_id, cycle_id, user_id, filters, display_filters, display_properties,
                                         created_by_id, updated_by_id, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(project_id), sqlc.arg(cycle_id), sqlc.arg(user_id), sqlc.arg(filters),
        sqlc.arg(display_filters), sqlc.arg(display_properties), sqlc.arg(user_id), sqlc.arg(user_id), sqlc.arg(now), sqlc.arg(now))
ON CONFLICT (cycle_id, user_id) WHERE deleted_at IS NULL DO UPDATE
SET filters            = CASE WHEN sqlc.arg(set_filters)::boolean THEN EXCLUDED.filters ELSE p.filters END,
    display_filters    = CASE WHEN sqlc.arg(set_display_filters)::boolean THEN EXCLUDED.display_filters ELSE p.display_filters END,
    display_properties = CASE WHEN sqlc.arg(set_display_properties)::boolean THEN EXCLUDED.display_properties ELSE p.display_properties END,
    updated_by_id      = EXCLUDED.updated_by_id,
    updated_at         = EXCLUDED.updated_at
RETURNING filters, display_filters, display_properties;

-- name: ModulePreferences :one
-- CyclePreferences' statement, of a module's page. module_user_properties_module_id_user_id_key serves it.
SELECT filters, display_filters, display_properties
FROM module_user_properties
WHERE module_id = sqlc.arg(module_id) AND user_id = sqlc.arg(user_id) AND deleted_at IS NULL;

-- name: UpsertModulePreferences :one
-- UpsertCyclePreferences' statement, of a module's page.
INSERT INTO module_user_properties AS p (id, workspace_id, project_id, module_id, user_id, filters, display_filters, display_properties,
                                          created_by_id, updated_by_id, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(project_id), sqlc.arg(module_id), sqlc.arg(user_id), sqlc.arg(filters),
        sqlc.arg(display_filters), sqlc.arg(display_properties), sqlc.arg(user_id), sqlc.arg(user_id), sqlc.arg(now), sqlc.arg(now))
ON CONFLICT (module_id, user_id) WHERE deleted_at IS NULL DO UPDATE
SET filters            = CASE WHEN sqlc.arg(set_filters)::boolean THEN EXCLUDED.filters ELSE p.filters END,
    display_filters    = CASE WHEN sqlc.arg(set_display_filters)::boolean THEN EXCLUDED.display_filters ELSE p.display_filters END,
    display_properties = CASE WHEN sqlc.arg(set_display_properties)::boolean THEN EXCLUDED.display_properties ELSE p.display_properties END,
    updated_by_id      = EXCLUDED.updated_by_id,
    updated_at         = EXCLUDED.updated_at
RETURNING filters, display_filters, display_properties;
