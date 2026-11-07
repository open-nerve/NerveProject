import { expect } from "@playwright/test";

import type { Database } from "../db";

// Database assertions of the planning stories (M6 design 2). The page version and the API version of a story call
// the same function.

/** A cycle of a project as expectCycles reads it. */
export interface CycleRow {
  name: string;
  description: string;
  /** Its days, as YYYY-MM-DD; both null for a draft. */
  start: string | null;
  end: string | null;
  sort_order: number;
  archived: boolean;
  deleted: boolean;
  /** The address of the account that owns it: its creator. */
  owner: string;
  /** The address of the account that wrote it last: created, changed, archived, unarchived or deleted it. */
  by: string;
}

/**
 * G1–G3, G8, G18: the cycles of the project of projectId, deleted ones too, are exactly want, by name. Each is a row
 * of the project's workspace, made by its owner; an archived one was archived, and a deleted one deleted, at a moment
 * of its writes, a deleted one at its last (M6 P1 spec 2.4).
 */
export async function expectCycles(db: Database, projectId: string, want: CycleRow[]): Promise<void> {
  expect(
    await db.query(
      `SELECT c.name, c.description, c.start_date::text AS start, c.end_date::text AS end, c.sort_order,
              c.archived_at IS NOT NULL AS archived, c.deleted_at IS NOT NULL AS deleted, o.email AS owner,
              b.email AS by,
              c.workspace_id = p.workspace_id AND c.created_by_id = c.owned_by_id AS in_its_project,
              coalesce(c.archived_at BETWEEN c.created_at AND c.updated_at, true)
                AND coalesce(c.deleted_at = c.updated_at, true) AS written_at_its_moments
         FROM cycles c
         JOIN projects p ON p.id = c.project_id
         JOIN users o ON o.id = c.owned_by_id
         JOIN users b ON b.id = c.updated_by_id
        WHERE c.project_id = $1 ORDER BY c.name COLLATE "C"`,
      [projectId]
    ),
    `the cycles of ${projectId}`
  ).toEqual(
    want
      .toSorted((a, b) => (a.name < b.name ? -1 : a.name > b.name ? 1 : 0))
      .map(({ name, description, start, end, sort_order, archived, deleted, owner, by }) => ({
        name,
        description,
        start,
        end,
        sort_order,
        archived,
        deleted,
        owner,
        by,
        in_its_project: true,
        written_at_its_moments: true,
      }))
  );
}

/** The moment of the last write of each cycle of the project of projectId, by name, deleted ones too. */
export async function cycleMoments(db: Database, projectId: string): Promise<Record<string, string>> {
  const rows = await db.query<{ name: string; at: string }>(
    `SELECT name, updated_at::text AS at FROM cycles WHERE project_id = $1`,
    [projectId]
  );
  return Object.fromEntries(rows.map((r) => [r.name, r.at]));
}
