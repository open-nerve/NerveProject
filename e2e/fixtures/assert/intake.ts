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
