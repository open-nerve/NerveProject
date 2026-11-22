import type { Database } from "../db";

// Database reads of the M7 notification and subscription stories, read-only: the page version and the API version of
// a story assert the same rows (M7 P4 spec 2.11).

/**
 * The rows of the account of accountId among the subscribers of the work item of issueId, the deleted ones too, in the
 * order they were made: whether it is live; whether it is of the work item's workspace and project, and made and last
 * written by its subscriber; and whether its moments are a subscription's, made and last written at one (live), or an
 * unsubscription's, deleted at its last write, after it was made.
 */
export async function subscriptionRows(db: Database, issueId: string, accountId: string) {
  return db.query<{ live: boolean; hers: boolean; stamped: boolean }>(
    `SELECT s.deleted_at IS NULL AS live,
            s.workspace_id = i.workspace_id AND s.project_id = i.project_id
              AND s.created_by_id = s.subscriber_id AND s.updated_by_id = s.subscriber_id AS hers,
            CASE WHEN s.deleted_at IS NULL THEN s.updated_at = s.created_at
                 ELSE s.deleted_at = s.updated_at AND s.updated_at > s.created_at END AS stamped
       FROM issue_subscribers s JOIN issues i ON i.id = s.issue_id
      WHERE s.issue_id = $1 AND s.subscriber_id = $2 ORDER BY s.created_at, s.id`,
    [issueId, accountId]
  );
}

/**
 * The states of the notifications of ids as their rows hold them, in the order of ids: whether each is read, archived
 * and snoozed, and whether its last write is its receiver's.
 */
export async function notificationStates(db: Database, ids: readonly string[]) {
  const rows = await db.query<{ id: string; read: boolean; archived: boolean; snoozed: boolean; by_receiver: boolean }>(
    `SELECT id::text, read_at IS NOT NULL AS read, archived_at IS NOT NULL AS archived,
            snoozed_till IS NOT NULL AS snoozed, updated_by_id IS NOT DISTINCT FROM receiver_id AS by_receiver
       FROM notifications WHERE id = ANY($1::uuid[])`,
    [ids]
  );
  return ids.map((id) => {
    const row = rows.find((r) => r.id === id);
    if (!row) throw new Error(`no notification ${id}`);
    const { read, archived, snoozed, by_receiver } = row;
    return { read, archived, snoozed, by_receiver };
  });
}
