import { expect } from "@playwright/test";

import type { Database } from "../db";
import { asynchronousDeletionTable, projectTables } from "./catalog";

// The database assertion of a project's deletion, by every table under projects (M3 design 3.6), which the project,
// workspace and purge stories call.

/**
 * P4, W3: the project of projectId is deleted by the account of adminEmail, and with it, at the same moment and by the
 * same account, every row under it that was not deleted before, of each table under projects (projectTables), by its
 * project_id. Each table has such a row, and none is left undeleted; a row deleted before the project, as a label a
 * story deleted, is not counted. M3 rows are immediate; issue/asset/planning-owned rows wait for the committed task.
 * Each table of removedBefore, whose rows the project's archive by the same account removed (M7 P3c spec 2.6, R12),
 * has rows deleted earlier than the project by him instead, and none deleted with it.
 */
export async function expectProjectDeleted(
  db: Database,
  projectId: string,
  adminEmail: string,
  removedBefore: string[] = []
): Promise<void> {
  const [project] = await db.query(
    `SELECT p.deleted_at IS NOT NULL AS deleted, u.email AS by FROM projects p JOIN users u ON u.id = p.updated_by_id WHERE p.id = $1`,
    [projectId]
  );
  expect(project, `the project ${projectId}`).toEqual({ deleted: true, by: adminEmail });
  const tables = await projectTables(db);
  expect(tables, "project deletion includes personal drafts").toContain("draft_issues");
  expect(tables, "the tables of rows removed before").toEqual(expect.arrayContaining(removedBefore));
  // The database compares the moments: a Date holds milliseconds, a timestamptz microseconds.
  const readRows = (names: string[]) =>
    Promise.all(
      names.map(async (name) => {
        const [counts] = await db.query<{ with_it: number; other: number; earlier: number }>(
          `SELECT count(*) FILTER (WHERE t.deleted_at = p.deleted_at AND t.updated_by_id = p.updated_by_id)::int AS with_it,
                count(*) FILTER (WHERE (t.deleted_at IS NULL OR t.deleted_at >= p.deleted_at)
                                   AND (t.deleted_at IS DISTINCT FROM p.deleted_at
                                        OR t.updated_by_id IS DISTINCT FROM p.updated_by_id))::int AS other,
                count(*) FILTER (WHERE t.deleted_at < p.deleted_at AND t.updated_by_id = p.updated_by_id)::int AS earlier
           FROM ${name} t JOIN projects p ON p.id = t.project_id WHERE p.id = $1`,
          [projectId]
        );
        const row = { table: name, deletedWithIt: (counts?.with_it ?? 0) > 0, other: counts?.other };
        return removedBefore.includes(name) ? { ...row, removedBefore: (counts?.earlier ?? 0) > 0 } : row;
      })
    );
  const immediate = tables.filter((name) => !asynchronousDeletionTable(name));
  const deferred = tables.filter(asynchronousDeletionTable);
  expect(await readRows(immediate), `the synchronous rows under ${projectId}`).toEqual(
    projectDeletionCounts(immediate, removedBefore)
  );
  await expect
    .poll(() => readRows(deferred), {
      message: `the asynchronous issue/asset/planning rows under ${projectId}`,
      timeout: 15_000,
    })
    .toEqual(projectDeletionCounts(deferred, removedBefore));
}

function projectDeletionCounts(names: string[], removedBefore: string[]) {
  return names.map((table) =>
    removedBefore.includes(table)
      ? { table, deletedWithIt: false, other: 0, removedBefore: true }
      : { table, deletedWithIt: true, other: 0 }
  );
}
