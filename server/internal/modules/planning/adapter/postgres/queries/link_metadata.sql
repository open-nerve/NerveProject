-- name: SetModuleLinkMetadata :execrows
-- A link's crawl (M6 P2 spec R11): the metadata it found written, alone, only while the link is undeleted and has the
-- address it crawled, as it is spelt; a link deleted or given another address since is not written. It takes no lock of
-- the project and writes no audit: the link's row alone, which a deletion of its module locks in id order
-- (DeleteModuleLinkRows) and a change of the link writes under the project's lock after reading it again.
UPDATE module_links
SET metadata = sqlc.arg(metadata)
WHERE id = sqlc.arg(id) AND url = sqlc.arg(url) AND deleted_at IS NULL;
