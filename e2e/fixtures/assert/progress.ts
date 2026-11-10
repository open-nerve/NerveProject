import { expect } from "@playwright/test";

import type { Database } from "../db";
import { countsOf, type Progress } from "../planning-progress";

// Database assertions of the stories of a cycle's and a module's progress, its freeze and a cycle's transfer (M6 P4
// spec 2.1, 2.4–2.6): the numbers counted from the memberships, the work items and their states, assignees and
// labels; the frozen column of a cycle and its last write; the runs of the periodic freeze.

/** The cycle or the module whose work items are counted. */
type Target = { cycle: string } | { module: string };

/** A progress without its burndown: what the database counts in SQL (progressInDatabase). */
type Counted = Omit<Progress, "burndown">;

/**
 * The SQL of the distribution by assignee or label of the work items of the CTE work (progressInDatabase), as the API
 * answers it: of those not cancelled, the completed and the pending by each undeleted one, or none, by id, none last.
 */
function distribution(kind: "assignee" | "label"): string {
  const [rows, key] = kind === "assignee" ? ["issue_assignees", "assignee_id"] : ["issue_labels", "label_id"];
  return `SELECT coalesce(json_agg(json_build_object('${key}', d.id, 'total_issues', d.total, 'completed_issues',
                   d.completed, 'pending_issues', d.pending) ORDER BY d.id NULLS LAST), '[]')
            FROM (SELECT r.${key} AS id, count(*) AS total, count(*) FILTER (WHERE w.g = 'completed') AS completed,
                         count(*) FILTER (WHERE w.g IN ('backlog', 'unstarted', 'started')) AS pending
                    FROM work w LEFT JOIN ${rows} r ON r.issue_id = w.id AND r.deleted_at IS NULL
                   WHERE w.g <> 'cancelled' GROUP BY r.${key}) d`;
}

/**
 * The progress of target as the database has it, without its burndown: its work items by an undeleted membership,
 * undeleted and not archived, those creator made alone when given (an OwnOnly caller's); the counts by their states'
 * groups; and, of those not cancelled, the completed and the pending by each undeleted assignee and label, or none,
 * each only with a work item, by id, none last.
 */
async function progressInDatabase(db: Database, target: Target, creator?: string): Promise<Counted> {
  const [table, column, id] =
    "cycle" in target ? ["cycle_issues", "cycle_id", target.cycle] : ["module_issues", "module_id", target.module];
  const [row] = await db.query<Counted>(
    `WITH work AS (
       SELECT i.id, s."group" AS g
         FROM ${table} m JOIN issues i ON i.id = m.issue_id JOIN states s ON s.id = i.state_id
        WHERE m.${column} = $1 AND m.deleted_at IS NULL AND i.deleted_at IS NULL AND i.archived_at IS NULL
          AND ($2::uuid IS NULL OR i.created_by_id = $2::uuid))
     SELECT (SELECT count(*) FROM work)::int AS total_issues,
            (SELECT count(*) FROM work WHERE g = 'backlog')::int AS backlog_issues,
            (SELECT count(*) FROM work WHERE g = 'unstarted')::int AS unstarted_issues,
            (SELECT count(*) FROM work WHERE g = 'started')::int AS started_issues,
            (SELECT count(*) FROM work WHERE g = 'completed')::int AS completed_issues,
            (SELECT count(*) FROM work WHERE g = 'cancelled')::int AS cancelled_issues,
            (${distribution("assignee")}) AS assignees, (${distribution("label")}) AS labels`,
    [id, creator ?? null]
  );
  if (!row) {
    throw new Error(`no progress of ${JSON.stringify(target)}`);
  }
  return row;
}

/**
 * G5–G7, G12: the progress p that the API answers of target is, but its burndown, what the database counts
 * (progressInDatabase), for creator alone when given.
 */
export async function expectCountedInDatabase(db: Database, target: Target, p: Progress, creator?: string) {
  expect(await progressInDatabase(db, target, creator), `the database's count of ${JSON.stringify(target)}`).toEqual({
    ...countsOf(p),
    assignees: p.assignees,
    labels: p.labels,
  });
}

/** The frozen progress of the cycle of id as its column holds it, null while it is not frozen. */
export async function snapshotOf(db: Database, id: string): Promise<unknown> {
  const [row] = await db.query<{ snapshot: unknown }>(
    "SELECT progress_snapshot AS snapshot FROM cycles WHERE id = $1",
    [id]
  );
  if (!row) {
    throw new Error(`no cycle ${id}`);
  }
  return row.snapshot;
}

/**
 * Waits, polling the database, until the cycle of id is frozen, by the periodic freeze (every 2s in the tests,
 * M6 P4 spec 2.5) or a transfer, and answers its column.
 */
export async function frozenSnapshot(db: Database, id: string): Promise<unknown> {
  await expect
    .poll(() => snapshotOf(db, id), { message: `the cycle ${id} frozen`, timeout: 20_000, intervals: [100, 250, 500] })
    .not.toBeNull();
  return snapshotOf(db, id);
}

/**
 * How the cycle of id was last written and frozen: whether its last write is still the one at written, by the
 * account of the address by (a freeze does not write it, M6 P4 spec R6), and whether it was frozen within [from, to],
 * microseconds and all. frozen_at is nerve's business clock: a bound taken from the runner's own clock holds only
 * while the test profile's clock offset is 0 and nerve runs on the runner's host, as it does locally and on CI.
 */
export async function frozenWrite(db: Database, id: string, written: string, from: string, to: string) {
  const [row] = await db.query(
    `SELECT u.email AS by, c.updated_at = $2::timestamptz AS written_then,
            (c.progress_snapshot->>'frozen_at')::timestamptz BETWEEN $3::timestamptz AND $4::timestamptz AS frozen_within
       FROM cycles c JOIN users u ON u.id = c.updated_by_id WHERE c.id = $1`,
    [id, written, from, to]
  );
  return row;
}

/** The database's clock now, to the microsecond: a moment the runs of the periodic freeze are counted after. */
export async function databaseNow(db: Database): Promise<string> {
  const [row] = await db.query<{ now: string }>("SELECT clock_timestamp()::text AS now");
  if (!row) {
    throw new Error("the database answered no clock");
  }
  return row.now;
}

/** Waits, polling the database, until a run of the periodic freeze that started after since has completed. */
export async function freezeRanSince(db: Database, since: string): Promise<void> {
  await expect
    .poll(
      async () =>
        (
          await db.query(
            `SELECT 1 FROM river_job
              WHERE kind = 'planning.freeze_cycles' AND state = 'completed' AND attempted_at > $1::timestamptz`,
            [since]
          )
        ).length,
      { message: `a run of the freeze after ${since}`, timeout: 20_000, intervals: [100, 250, 500] }
    )
    .toBeGreaterThan(0);
}
