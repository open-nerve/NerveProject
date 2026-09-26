-- name: CreateProfile :exec
-- Every preference takes its default (Plane's model defaults, M2 design 4.3).
INSERT INTO profiles (id, user_id, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(user_id), sqlc.arg(now), sqlc.arg(now));

-- name: GetProfile :one
SELECT theme, language, start_of_the_week, onboarding_step, is_onboarded, is_tour_completed, last_workspace_id, updated_at
FROM profiles
WHERE user_id = sqlc.arg(user_id);

-- name: UpdateProfile :one
-- PATCH /me/profile: only the fields that are set change, and the steps are merged into the stored object in the
-- statement (M2 design 3.14): a concurrent update of other steps waits for the row lock and is merged into the new
-- row, not lost. An empty patch, {}, merges nothing.
UPDATE profiles
SET updated_at        = sqlc.arg(now),
    theme             = CASE WHEN sqlc.arg(set_theme)::boolean THEN sqlc.arg(theme)::text ELSE theme END,
    language          = CASE WHEN sqlc.arg(set_language)::boolean THEN sqlc.arg(language)::text ELSE language END,
    start_of_the_week = CASE WHEN sqlc.arg(set_start_of_the_week)::boolean THEN sqlc.arg(start_of_the_week)::smallint ELSE start_of_the_week END,
    onboarding_step   = onboarding_step || sqlc.arg(onboarding_step_patch)::jsonb,
    is_onboarded      = CASE WHEN sqlc.arg(set_is_onboarded)::boolean THEN sqlc.arg(is_onboarded)::boolean ELSE is_onboarded END,
    is_tour_completed = CASE WHEN sqlc.arg(set_is_tour_completed)::boolean THEN sqlc.arg(is_tour_completed)::boolean ELSE is_tour_completed END,
    last_workspace_id = CASE WHEN sqlc.arg(set_last_workspace_id)::boolean THEN sqlc.narg(last_workspace_id)::uuid ELSE last_workspace_id END
WHERE user_id = sqlc.arg(user_id)
RETURNING theme, language, start_of_the_week, onboarding_step, is_onboarded, is_tour_completed, last_workspace_id, updated_at;
