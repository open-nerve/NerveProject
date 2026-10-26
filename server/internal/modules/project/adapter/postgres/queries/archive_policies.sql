-- name: ArchiveProjects :many
-- Candidate IDs only: the authoritative policy is read under the project lock.
SELECT id
FROM projects
WHERE deleted_at IS NULL AND archived_at IS NULL AND archive_in BETWEEN 1 AND 12
ORDER BY id;
