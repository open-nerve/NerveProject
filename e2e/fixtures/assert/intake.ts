import type { Database } from "../db";

// Database reads of the M7 intake stories, read-only: the page version and the API version of a story assert the same
// rows (M7 P5a spec 2.12).

/**
 * The live intake row of the work item of issueId: its status, the moment it is snoozed till (ISO) and the work item
 * it duplicates, none when it is not so, and who wrote it last, by id.
 */
export async function intakeRowOf(db: Database, issueId: string) {
  const [row] = await db.query<{
    status: string;
    snoozed_till: Date | null;
    duplicate_to_id: string | null;
    written_by: string | null;
  }>(
    `SELECT status, snoozed_till, duplicate_to_id::text AS duplicate_to_id, updated_by_id::text AS written_by
       FROM intake_issues WHERE issue_id = $1 AND deleted_at IS NULL`,
    [issueId]
  );
  if (!row) throw new Error(`no live intake row of ${issueId}`);
  return { ...row, snoozed_till: row.snoozed_till?.toISOString() ?? null };
}

/**
 * The content of the work item of issueId as nerve keeps it: its name, its description, its priority, who wrote it
 * last, by id, and whether it is deleted.
 */
export async function contentOf(db: Database, issueId: string) {
  const [row] = await db.query<{
    name: string;
    description_html: string;
    priority: string;
    written_by: string | null;
    deleted: boolean;
  }>(
    `SELECT name, description_html, priority, updated_by_id::text AS written_by, deleted_at IS NOT NULL AS deleted
       FROM issues WHERE id = $1`,
    [issueId]
  );
  if (!row) throw new Error(`no work item ${issueId}`);
  return row;
}

/**
 * The live notifications of the account of receiverId about the work item of issueId, in the order they were made:
 * each why he is sent it (sender), who made the change, by id, and the change it tells: its field, its verb and the
 * values it moved from and to.
 */
export async function notificationRowsOf(db: Database, issueId: string, receiverId: string) {
  return db.query<{
    sender: string;
    by: string | null;
    field: string | null;
    verb: string | null;
    old_value: string | null;
    new_value: string | null;
  }>(
    `SELECT sender, triggered_by_id::text AS by, data->'change'->>'field' AS field, data->'change'->>'verb' AS verb,
            data->'change'->>'old_value' AS old_value, data->'change'->>'new_value' AS new_value
       FROM notifications WHERE issue_id = $1 AND receiver_id = $2 AND deleted_at IS NULL
      ORDER BY created_at, change_seq, id`,
    [issueId, receiverId]
  );
}

/**
 * The triage activities of the work item of issueId, in the order they were made: each its verb, the new status's
 * name, the statuses it moved from and to, and its actor, by id.
 */
export async function triageActivities(db: Database, issueId: string) {
  return db.query<{ verb: string; old_value: string | null; new_value: string | null; actor: string | null }>(
    `SELECT verb, old_value, new_value, actor_id::text AS actor
       FROM issue_activities WHERE issue_id = $1 AND field = 'intake' AND deleted_at IS NULL
      ORDER BY created_at, id`,
    [issueId]
  );
}

/**
 * What a submission to an intake made of the work item of issueId beside its content (contentOf): who submitted it, by
 * id; its due date, YYYY-MM-DD (null for none); its live assignees and labels, by id, in id order.
 */
export async function submittedOf(db: Database, issueId: string) {
  const [row] = await db.query<{
    submitter: string;
    target_date: string | null;
    assignee_ids: string[];
    label_ids: string[];
  }>(
    `SELECT i.created_by_id::text AS submitter, to_char(i.target_date, 'YYYY-MM-DD') AS target_date,
            ARRAY(SELECT assignee_id::text FROM issue_assignees
                   WHERE issue_id = i.id AND deleted_at IS NULL ORDER BY assignee_id) AS assignee_ids,
            ARRAY(SELECT label_id::text FROM issue_labels
                   WHERE issue_id = i.id AND deleted_at IS NULL ORDER BY label_id) AS label_ids
       FROM issues i WHERE i.id = $1`,
    [issueId]
  );
  if (!row) throw new Error(`no work item ${issueId}`);
  return row;
}

/** The names of the work items in the intake of the project of projectId, their intake rows live, in name order. */
export async function intakeNamesOf(db: Database, projectId: string): Promise<string[]> {
  const rows = await db.query<{ name: string }>(
    `SELECT i.name FROM intake_issues ii JOIN issues i ON i.id = ii.issue_id
      WHERE ii.project_id = $1 AND ii.deleted_at IS NULL ORDER BY i.name COLLATE "C"`,
    [projectId]
  );
  return rows.map(({ name }) => name);
}

/**
 * The file of id as nerve keeps it, a work item's in an intake or once out of it: the key of its object in the storage
 * and its bytes' digest (SHA-256, hexadecimal), who uploaded it, by id, whether it is uploaded (confirmed) and bound to
 * its work item (an attachment as it is confirmed, an image as the description that holds it is saved), and whether it
 * is deleted and by whom.
 */
export async function intakeFileOf(db: Database, id: string) {
  const [row] = await db.query<{
    key: string;
    sha: string;
    by: string;
    uploaded: boolean;
    bound: boolean;
    deleted: boolean;
    deleted_by: string | null;
  }>(
    `SELECT storage_metadata->'final'->>'key' AS key, storage_metadata->'final'->>'sha256' AS sha,
            created_by_id::text AS by, is_uploaded AS uploaded, bound_at IS NOT NULL AS bound, is_deleted AS deleted,
            deleted_by_id::text AS deleted_by
       FROM file_assets WHERE id = $1`,
    [id]
  );
  if (!row) throw new Error(`no file ${id}`);
  return row;
}

/** Whether each file named name of the work item of issueId is uploaded (confirmed), in no particular order. */
export async function uploadsNamed(db: Database, issueId: string, name: string): Promise<boolean[]> {
  const rows = await db.query<{ uploaded: boolean }>(
    `SELECT is_uploaded AS uploaded FROM file_assets WHERE issue_id = $1 AND attributes->>'name' = $2`,
    [issueId, name]
  );
  return rows.map(({ uploaded }) => uploaded);
}

/**
 * The links of the work item of issueId as nerve keeps them, deleted ones too, in the order they were added: each its
 * address, its title, who added it and who wrote it last, by id, and whether it is deleted.
 */
export async function intakeLinksOf(db: Database, issueId: string) {
  return db.query<{ url: string; title: string | null; adder: string | null; by: string | null; deleted: boolean }>(
    `SELECT url, title, created_by_id::text AS adder, updated_by_id::text AS by, deleted_at IS NOT NULL AS deleted
       FROM issue_links WHERE issue_id = $1 ORDER BY created_at, id`,
    [issueId]
  );
}
