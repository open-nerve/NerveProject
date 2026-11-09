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
