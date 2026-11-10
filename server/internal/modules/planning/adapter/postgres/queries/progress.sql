-- name: CycleSnapshot :one
-- The frozen progress of the undeleted cycle id, archived or not, null when it is not frozen (M6 P4 spec R5): the whole
-- document, for the cycle's progress. Read in the caller's snapshot, with the cycle's other facts. The primary key
-- serves it.
SELECT progress_snapshot
FROM cycles
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;

-- name: CycleSnapshotCounts :many
-- Of the cycles ids, the undeleted frozen ones, archived or not, in id order, each with the six counts and the moment
-- of its frozen progress alone (M6 P4 spec R5): a list row does not read the distributions and the burndown, so they
-- are taken out here, in one statement for the page. The primary key serves the ids.
SELECT id, (progress_snapshot - '{assignees,labels,burndown}'::text[])::jsonb AS counts
FROM cycles
WHERE id = ANY(sqlc.arg(ids)::uuid[]) AND deleted_at IS NULL AND progress_snapshot IS NOT NULL
ORDER BY id;

-- name: FreezeCycle :execrows
-- The cycle's progress frozen, once (M6 P4 spec R5): written only while there is none, so the first writer, the job or a
-- transfer, each under the project's lock, stays; a deleted cycle is not frozen. One row when it wrote, none when it did
-- not. The cycle's last change stays (spec R6): a freeze is a record, not a write of anyone's.
UPDATE cycles
SET progress_snapshot = sqlc.arg(snapshot)::jsonb
WHERE id = sqlc.arg(id) AND deleted_at IS NULL AND progress_snapshot IS NULL;

-- name: FreezeDueProjects :many
-- The freeze job's candidates, read without a lock (M6 P4 spec 2.5): the projects, in id order and each once, with an
-- undeleted cycle not frozen, archived or not, that ends on utc_today or before it. Today in any time zone is at most a
-- day after the UTC date, so these are a superset of the projects with an ended cycle; each is decided under its lock.
-- The partial index cycles_freeze_due_idx (end_date) serves it.
SELECT DISTINCT project_id
FROM cycles
WHERE deleted_at IS NULL AND progress_snapshot IS NULL AND end_date IS NOT NULL AND end_date <= sqlc.arg(utc_today)::date
ORDER BY project_id;

-- name: CyclesToFreeze :many
-- The project's undeleted cycles not frozen, archived or not, that ended before today, the project's day, in id order
-- (M6 P4 spec 2.5): their status on today is completed (domain.StatusOf: completed is after the end day), and a draft
-- has no end. Read under the project's lock the job holds, which every write on the project's cycles takes first. The
-- list index (project_id, sort_order, created_at, id) WHERE deleted_at IS NULL serves it.
SELECT id, workspace_id, project_id, name, description, start_date, end_date, owned_by_id, sort_order, archived_at,
       created_at, updated_at, created_by_id, updated_by_id
FROM cycles
WHERE project_id = sqlc.arg(project_id) AND deleted_at IS NULL AND progress_snapshot IS NULL
  AND end_date < sqlc.arg(today)::date
ORDER BY id;
