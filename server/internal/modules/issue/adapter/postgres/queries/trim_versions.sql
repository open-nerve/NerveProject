-- name: TrimDescriptionVersions :execrows
-- The caller holds the undeleted project's NO KEY UPDATE lock. Archived
-- issues remain eligible; soft-deleted issues and versions belong to purge.
WITH ranked AS (
    SELECT v.id, row_number() OVER (
        PARTITION BY v.issue_id ORDER BY v.last_saved_at DESC, v.id DESC
    ) AS position
    FROM issue_description_versions v
    JOIN issues i ON i.id = v.issue_id
    WHERE i.project_id = sqlc.arg(project_id)
      AND v.project_id = sqlc.arg(project_id)
      AND i.deleted_at IS NULL
      AND v.deleted_at IS NULL
)
DELETE FROM issue_description_versions v
USING ranked r
WHERE v.id = r.id AND r.position > 20;
