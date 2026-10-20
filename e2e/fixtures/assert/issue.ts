import { expect } from "@playwright/test";

import { shownNameOf } from "../auth";
import type { Database } from "../db";
import { keysBelow } from "./catalog";

// Database assertions of the work-item stories. The page version and the
// API version of a story call the same function (M4 design 2).

/**
 * A row of a work item's activity as the assertions read it: what its identifiers name, an account by its address, a
 * state or a label by its name.
 */
export interface ActivityRow {
  field: string | null;
  verb: string;
  old_value: string | null;
  new_value: string | null;
  /** What the row's new_identifier names: the account assigned, the state or the label set; else null. */
  named: string | null;
  /** What its old_identifier names: the account unassigned, the state left or the label taken off; else null. */
  was: string | null;
  comment: string;
}

/** The order of the ids, which a work item answers its accounts and labels in, and a write adds them in. */
export function byId(a: { id: string }, b: { id: string }): number {
  return a.id < b.id ? -1 : 1;
}

/**
 * An "updated" row of a work item's activity: field changed from old to now, with its comment, Plane's but for the
 * archiving's and the restoring's rows; named and was, what its new and old identifiers name, for a state's, a
 * label's or an account's row.
 */
export function changed(
  field: string,
  old: string | null,
  now: string,
  comment: string,
  named: string | null = null,
  was: string | null = null
): ActivityRow {
  return { field, verb: "updated", old_value: old, new_value: now, named, was, comment };
}

/** The row of a work item's activity that assigns it to the account of email, by the display name nerve gave him. */
export function assigned(email: string): ActivityRow {
  return changed("assignees", "", shownNameOf(email), "added assignee ", email);
}

/** The rows of a work item's activity that archive it and that restore it (M4 design 3.13, rows 14 and 16). */
export const archivedRow = changed("archived_at", null, "manual_archive", "archived the work item");
export const restoredRow = changed("archived_at", "archive", "restore", "restored the work item");

/** The first rows of a work item's activity: its creation, and its assignment to each of accounts, in id order. */
export function createdFor(...accounts: { id: string; email: string }[]): ActivityRow[] {
  const creation: ActivityRow = {
    field: null,
    verb: "created",
    old_value: null,
    new_value: null,
    named: null,
    was: null,
    comment: "created the work item",
  };
  return [creation, ...accounts.toSorted(byId).map(({ email }) => assigned(email))];
}

/** The activity of the work item of issueId, in its order. */
async function activityOf(db: Database, issueId: string): Promise<ActivityRow[]> {
  return db.query<ActivityRow>(
    `SELECT a.field, a.verb, a.old_value, a.new_value, COALESCE(nu.email, ns.name, nl.name) AS named,
            COALESCE(ou.email, os.name, ol.name) AS was, a.comment
       FROM issue_activities a
       LEFT JOIN users nu ON nu.id = a.new_identifier LEFT JOIN states ns ON ns.id = a.new_identifier
       LEFT JOIN labels nl ON nl.id = a.new_identifier
       LEFT JOIN users ou ON ou.id = a.old_identifier LEFT JOIN states os ON os.id = a.old_identifier
       LEFT JOIN labels ol ON ol.id = a.old_identifier
      WHERE a.issue_id = $1 ORDER BY a.created_at, a.id`,
    [issueId]
  );
}

/**
 * The undeleted assignees, labels and subscribers of the work item of issueId, by address or name, each in order, and
 * its activity.
 */
async function rowsOf(db: Database, issueId: string) {
  const named = async (sql: string) => (await db.query<{ v: string }>(sql, [issueId])).map(({ v }) => v);
  return {
    assignees: await named(`SELECT u.email AS v FROM issue_assignees r JOIN users u ON u.id = r.assignee_id
                              WHERE r.issue_id = $1 AND r.deleted_at IS NULL ORDER BY u.email COLLATE "C"`),
    labels: await named(`SELECT l.name AS v FROM issue_labels r JOIN labels l ON l.id = r.label_id
                           WHERE r.issue_id = $1 AND r.deleted_at IS NULL ORDER BY l.name COLLATE "C"`),
    subscribers: await named(`SELECT u.email AS v FROM issue_subscribers r JOIN users u ON u.id = r.subscriber_id
                                WHERE r.issue_id = $1 AND r.deleted_at IS NULL ORDER BY u.email COLLATE "C"`),
    activity: await activityOf(db, issueId),
  };
}

/**
 * The work item's assignees, labels, subscribers and activity, each row with what the stamps' checks compare: its
 * workspace and project, who made it and wrote it last, who acted (the activity's actor, else who made it), and when.
 */
const rowStamps = `
              SELECT workspace_id, project_id, created_by_id, updated_by_id, created_by_id AS actor, created_at, updated_at, deleted_at
                FROM issue_assignees WHERE issue_id = i.id
              UNION ALL SELECT workspace_id, project_id, created_by_id, updated_by_id, created_by_id, created_at, updated_at, deleted_at
                FROM issue_labels WHERE issue_id = i.id
              UNION ALL SELECT workspace_id, project_id, created_by_id, updated_by_id, created_by_id, created_at, updated_at, deleted_at
                FROM issue_subscribers WHERE issue_id = i.id
              UNION ALL SELECT workspace_id, project_id, created_by_id, updated_by_id, actor_id, created_at, updated_at, deleted_at
                FROM issue_activities WHERE issue_id = i.id`;

/**
 * A work item as created: its number, name, state (by name), priority, dates (YYYY-MM-DD, null for none) and sort
 * order; whether it is completed; the address of the account that created it; its assignees and subscribers by
 * address, its labels by name, each in order; and its activity, in its order.
 */
export interface CreatedIssue {
  sequence_id: number;
  name: string;
  state: string;
  priority: string;
  start_date: string | null;
  target_date: string | null;
  sort_order: number;
  completed: boolean;
  by: string;
  assignees: string[];
  labels: string[];
  subscribers: string[];
  activity: ActivityRow[];
}

/**
 * I1: the work item of issueId holds want, as created (M4 design 3.7, 3.13): undeleted, not archived, numbered
 * want.sequence_id, which its project's last_issue_sequence is now. It and every row of its assignees, labels,
 * subscribers and activity are of its project and workspace, made and last written by the account of want.by, the
 * activity's actor too, at one moment.
 */
export async function expectIssueCreated(db: Database, issueId: string, want: CreatedIssue): Promise<void> {
  const [issue] = await db.query(
    `SELECT i.sequence_id, i.name, s.name AS state, i.priority, to_char(i.start_date, 'YYYY-MM-DD') AS start_date,
            to_char(i.target_date, 'YYYY-MM-DD') AS target_date, i.sort_order, i.completed_at IS NOT NULL AS completed,
            c.email AS by, p.last_issue_sequence,
            i.workspace_id = p.workspace_id AND i.updated_by_id = i.created_by_id AND i.updated_at = i.created_at
              AND i.deleted_at IS NULL AND i.archived_at IS NULL AS as_created
       FROM issues i JOIN projects p ON p.id = i.project_id JOIN states s ON s.id = i.state_id
       JOIN users c ON c.id = i.created_by_id
      WHERE i.id = $1`,
    [issueId]
  );
  const { assignees, labels, subscribers, activity, ...row } = want;
  expect(issue, `the work item ${issueId}`).toEqual({
    ...row,
    last_issue_sequence: want.sequence_id,
    as_created: true,
  });
  expect(await rowsOf(db, issueId), `the rows of the work item ${issueId}`).toEqual({
    assignees,
    labels,
    subscribers,
    activity,
  });
  // The database compares the moments: a Date holds milliseconds, a timestamptz microseconds.
  const [stamps] = await db.query<{ rows: number; stamped: number }>(
    `SELECT count(*)::int AS rows,
            count(*) FILTER (WHERE r.workspace_id = i.workspace_id AND r.project_id = i.project_id
                               AND r.created_by_id = i.created_by_id AND r.updated_by_id = i.created_by_id
                               AND r.actor IS NOT DISTINCT FROM i.created_by_id
                               AND r.created_at = i.created_at AND r.updated_at = i.created_at AND r.deleted_at IS NULL)::int
              AS stamped
       FROM issues i, LATERAL (${rowStamps}) r
      WHERE i.id = $1`,
    [issueId]
  );
  expect(stamps, `the rows of the work item ${issueId}, each made with it`).toEqual({
    rows: assignees.length + labels.length + subscribers.length + activity.length,
    stamped: assignees.length + labels.length + subscribers.length + activity.length,
  });
}

/**
 * A work item as changed: its name, state (by name), priority and dates (YYYY-MM-DD, null for none); whether it is
 * completed; the address of the account that changed it last; its undeleted assignees and subscribers by address, its
 * undeleted labels by name, each in order; its activity, in its order; and how many rows of its assignees, labels,
 * subscribers and activity its last change wrote, those it took off among them.
 */
export interface UpdatedIssue {
  name: string;
  state: string;
  priority: string;
  start_date: string | null;
  target_date: string | null;
  completed: boolean;
  by: string;
  assignees: string[];
  labels: string[];
  subscribers: string[];
  activity: ActivityRow[];
  wrote: number;
}

/**
 * I2: the work item of issueId holds want after a change (M4 design 3.7, 3.13): undeleted, last written by the
 * account of want.by at a moment after its creation, completed (completed_at set) exactly when want.completed. Its
 * undeleted assignees, labels and subscribers and its activity are want's. The rows its last change wrote, those
 * last written at the work item's updated_at, want.wrote of them, are of its project and workspace and want.by's at
 * that moment: those it made, made then; those it took off, soft-deleted then; its activity, acted then.
 */
export async function expectIssueUpdated(db: Database, issueId: string, want: UpdatedIssue): Promise<void> {
  const [issue] = await db.query(
    `SELECT i.name, s.name AS state, i.priority, to_char(i.start_date, 'YYYY-MM-DD') AS start_date,
            to_char(i.target_date, 'YYYY-MM-DD') AS target_date, i.completed_at IS NOT NULL AS completed, u.email AS by,
            i.updated_at > i.created_at AND i.deleted_at IS NULL AS as_changed
       FROM issues i JOIN states s ON s.id = i.state_id JOIN users u ON u.id = i.updated_by_id
      WHERE i.id = $1`,
    [issueId]
  );
  const { assignees, labels, subscribers, activity, wrote, ...row } = want;
  expect(issue, `the work item ${issueId}`).toEqual({ ...row, as_changed: true });
  expect(await rowsOf(db, issueId), `the rows of the work item ${issueId}`).toEqual({
    assignees,
    labels,
    subscribers,
    activity,
  });
  const [stamps] = await db.query<{ rows: number; stamped: number }>(
    `SELECT count(*)::int AS rows,
            count(*) FILTER (WHERE r.workspace_id = i.workspace_id AND r.project_id = i.project_id
                               AND r.updated_by_id = i.updated_by_id
                               AND (r.deleted_at = i.updated_at
                                    OR r.deleted_at IS NULL AND r.created_at = i.updated_at AND r.created_by_id = i.updated_by_id
                                       AND r.actor IS NOT DISTINCT FROM i.updated_by_id))::int AS stamped
       FROM issues i, LATERAL (${rowStamps}) r
      WHERE i.id = $1 AND r.updated_at = i.updated_at`,
    [issueId]
  );
  expect(stamps, `the rows the last change of the work item ${issueId} wrote`).toEqual({ rows: wrote, stamped: wrote });
}

/** A work item's archival: whether it is archived, the address of the account that wrote it last, and its activity. */
export interface Archival {
  archived: boolean;
  by: string;
  activity: ActivityRow[];
}

/**
 * I15: the work item of issueId is archived exactly when want.archived (M4 design 3.8): undeleted, written last by the
 * account of want.by, archived at that last write when it is archived. Its activity is want.activity, the last row of
 * it want.by's, made at that last write.
 */
export async function expectIssueArchival(db: Database, issueId: string, want: Archival): Promise<void> {
  const [issue] = await db.query(
    `SELECT i.archived_at IS NOT NULL AS archived, u.email AS by,
            i.deleted_at IS NULL AND (i.archived_at IS NULL OR i.archived_at = i.updated_at) AND a.actor_id = i.updated_by_id
              AND a.created_by_id = i.updated_by_id AND a.created_at = i.updated_at AS stamped
       FROM issues i JOIN users u ON u.id = i.updated_by_id,
            LATERAL (SELECT actor_id, created_by_id, created_at FROM issue_activities WHERE issue_id = i.id
                      ORDER BY created_at DESC, id DESC LIMIT 1) a
      WHERE i.id = $1`,
    [issueId]
  );
  const { activity, ...row } = want;
  expect(issue, `the work item ${issueId}`).toEqual({ ...row, stamped: true });
  expect(await activityOf(db, issueId), `the activity of the work item ${issueId}`).toEqual(activity);
}

/** The key of a work item's sub-items, which come with P3, as their deletion with their parent's (M4 design 3.6, 3.9). */
const subItems = "issues.parent_id";

/**
 * I15 (I17 from P3): the work item of issueId is deleted by the account of email, and with it, at the same moment and
 * by the same account, every row under it that was not deleted before (M4 design 3.6). A row is under it when a
 * foreign key of its table names the work item or a row under it: the catalog's graph of foreign keys is walked down
 * from issues (keysBelow, which fails on a key of more than one column), so that a row two keys below (a reaction to
 * a comment, which names the comment) is under it too. Each key of the graph has such a row, but its sub-items'
 * (subItems), which no work item has before P3; a row deleted before the work item is neither held to its deletion nor
 * checked here (that it keeps its deletion is `TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt`'s).
 */
export async function expectIssueDeleted(db: Database, issueId: string, email: string): Promise<void> {
  const [issue] = await db.query(
    `SELECT i.deleted_at IS NOT NULL AS deleted, u.email AS by FROM issues i JOIN users u ON u.id = i.updated_by_id WHERE i.id = $1`,
    [issueId]
  );
  expect(issue, `the work item ${issueId}`).toEqual({ deleted: true, by: email });
  const keys = await keysBelow(db, "issues");
  expect(keys.length, "the keys below issues").toBeGreaterThan(1);
  // The rows under the work item, by table, and how each key's were deleted: a round reads each key's rows under the
  // rows found before it, and the walk goes on while a round finds a row that is new.
  const under = new Map<string, Set<string>>([["issues", new Set([issueId])]]);
  const counts = new Map<string, { withIt: number; other: number }>();
  const round = async (): Promise<void> => {
    const found = await Promise.all(
      keys.map((key) =>
        db.query<{ id: string; with_it: boolean; other: boolean }>(
          `SELECT t.id, t.deleted_at = i.deleted_at AND t.updated_by_id = i.updated_by_id AS with_it,
                  (t.deleted_at IS NULL OR t.deleted_at >= i.deleted_at)
                    AND (t.deleted_at IS DISTINCT FROM i.deleted_at OR t.updated_by_id IS DISTINCT FROM i.updated_by_id) AS other
             FROM ${key.child} t, issues i WHERE t.${key.column} = ANY($1::uuid[]) AND i.id = $2`,
          [[...(under.get(key.parent) ?? [])], issueId]
        )
      )
    );
    let grew = false;
    keys.forEach((key, i) => {
      const rows = found[i] ?? [];
      counts.set(`${key.child}.${key.column}`, {
        withIt: rows.filter((r) => r.with_it).length,
        other: rows.filter((r) => r.other).length,
      });
      const ids = under.get(key.child) ?? new Set<string>();
      for (const { id } of rows) {
        grew ||= !ids.has(id);
        ids.add(id);
      }
      under.set(key.child, ids);
    });
    if (grew) {
      await round();
    }
  };
  await round();
  const names = keys.map((k) => `${k.child}.${k.column}`);
  expect(
    names.map((key) => ({ key, deletedWithIt: (counts.get(key)?.withIt ?? 0) > 0, other: counts.get(key)?.other })),
    `the rows under the work item ${issueId}`
  ).toEqual(names.map((key) => ({ key, deletedWithIt: key !== subItems, other: 0 })));
}
