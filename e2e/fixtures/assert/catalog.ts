import { expect } from "@playwright/test";

import type { Database } from "../db";

// The tables a deletion writes and the foreign keys under them, read from the database's catalog (M3 design 3.6; M4
// design 3.6), so that a table or a key a later phase adds is read too.

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

/** A single-column foreign key, including its target column used by the id-based traversal. */
interface ForeignKey {
  child: string;
  column: string;
  parent: string;
  target: string;
}

/**
 * The foreign keys below table, the catalog's graph walked down from it: each key that names table, then each that
 * names a table holding one of those, and so on, in the order the walk reaches them, each table walked once (a key of
 * a table to itself, as issues.parent_id, among them). Each is a key of one column, which names its row's parent; a
 * key of more columns or a target other than id fails here, named, for its rows would be found another way.
 */
export async function keysBelow(db: Database, table: string): Promise<ForeignKey[]> {
  const all = await db.query<ForeignKey & { columns: number; targetColumns: number }>(
    `SELECT c.conrelid::regclass::text AS child, a.attname::text AS column, c.confrelid::regclass::text AS parent,
            b.attname::text AS target, cardinality(c.conkey) AS columns, cardinality(c.confkey) AS "targetColumns"
       FROM pg_constraint c JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = c.conkey[1]
       JOIN pg_attribute b ON b.attrelid = c.confrelid AND b.attnum = c.confkey[1]
      WHERE c.contype = 'f'`
  );
  // tables grows as the loop goes, each table once.
  const tables = [table];
  const keys: ForeignKey[] = [];
  for (let i = 0; i < tables.length; i++) {
    for (const { columns, targetColumns, ...key } of all.filter((k) => k.parent === tables[i])) {
      expect(columns, `the columns of ${key.child}.${key.column}, a key to ${key.parent}`).toBe(1);
      expect(targetColumns, `the target columns of ${key.child}.${key.column}`).toBe(1);
      expect(key.target, `the target of ${key.child}.${key.column}, a key to ${key.parent}`).toBe("id");
      keys.push(key);
      if (!tables.includes(key.child)) {
        tables.push(key.child);
      }
    }
  }
  if (table === "issues") {
    expect(keys, "the exact comment reaction path below issues").toEqual(
      expect.arrayContaining([
        { child: "comment_reactions", column: "comment_id", parent: "issue_comments", target: "id" },
        { child: "issue_comments", column: "issue_id", parent: "issues", target: "id" },
      ])
    );
  }
  return keys;
}

/** Only issue-owned tables wait for background deletion; unknown catalog tables remain immediate. */
export function issueOwnedTable(name: string): boolean {
  return name === "issues" || name === "draft_issues" || name === "comment_reactions" || name.startsWith("issue_");
}
