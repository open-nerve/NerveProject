import { expect } from "@playwright/test";

import type { Database } from "../db";

// The tables a deletion writes, read from the database's catalog (M3 design 3.6; M4 design 3.6), so that a table a
// later phase adds is read too.

/**
 * The tables under parent: each table whose foreign key names parent. Every table with a column named column is one
 * of them, so a table that has the column without the foreign key fails here. Both lists sort their names under "C":
 * a regclass's text would sort under the database's collation and an information_schema name under "C", two orders
 * that differ (issues, issue_subscribers).
 */
async function tablesUnder(db: Database, parent: string, column: string): Promise<string[]> {
  const tables = await db.query<{ name: string }>(
    `SELECT DISTINCT c.conrelid::regclass::text COLLATE "C" AS name FROM pg_constraint c
      WHERE c.contype = 'f' AND c.confrelid = $1::regclass ORDER BY 1`,
    [parent]
  );
  expect(tables.length, `the tables under ${parent}`).toBeGreaterThan(0);
  const withColumn = await db.query<{ name: string }>(
    `SELECT DISTINCT c.table_name::text COLLATE "C" AS name FROM information_schema.columns c
      WHERE c.table_schema = current_schema() AND c.column_name = $1 ORDER BY 1`,
    [column]
  );
  expect(tables, `the tables under ${parent}, as their ${column} columns name them`).toEqual(withColumn);
  return tables.map(({ name }) => name);
}

/** The tables under projects, whose rows a project's deletion writes: P4, W3 (expectProjectDeleted). */
export function projectTables(db: Database): Promise<string[]> {
  return tablesUnder(db, "projects", "project_id");
}

/** The tables under workspaces, whose rows a workspace's deletion writes: W2, W3 (expectWorkspaceDeleted), W11. */
export function workspaceTables(db: Database): Promise<string[]> {
  return tablesUnder(db, "workspaces", "workspace_id");
}
