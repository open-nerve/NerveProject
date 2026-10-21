-- name: Ancestry :many
-- A work item's ancestry (M4 design 3.9): the work item and those above it, its parent, its parent's parent and so on,
-- deleted and archived ones too, in no order; a work item in it may not be put under it. Read under the project's FOR
-- NO KEY UPDATE, which every write of a parent holds (3.5), so it stays as read until the write commits. UNION reads
-- each work item once, so a line that loops, which no write makes, ends. The primary key serves it.
WITH RECURSIVE line (id, parent_id) AS (
    SELECT i.id, i.parent_id FROM issues i WHERE i.id = sqlc.arg(id)
    UNION
    SELECT p.id, p.parent_id FROM issues p JOIN line l ON p.id = l.parent_id
)
SELECT id FROM line;
