import { keysBelow } from "../../fixtures/assert/catalog";
import { createDatabase, openDatabase, templateDatabase } from "../../fixtures/db";
import { expect, test } from "../../fixtures/test";

// These databases are empty migrated copies: altering a real catalog must not
// change a worker's story data or depend on the number of actors in that data.
for (const endpoint of [
  {
    name: "comment reaction",
    table: "comment_reactions",
    column: "comment_id",
    parent: "issue_comments",
    other: "actor_id",
  },
  {
    name: "comment issue",
    table: "issue_comments",
    column: "issue_id",
    parent: "issues",
    other: "created_by_id",
  },
]) {
  test(`I17 (catalog): the ${endpoint.name} foreign key must target its parent's id`, async ({
    db: workerDatabase,
  }) => {
    const name = `${workerDatabase.name}_i17_${endpoint.table}`;
    await createDatabase(name, templateDatabase);
    const db = openDatabase(name);
    try {
      await db.query(`ALTER TABLE ${endpoint.parent} ADD UNIQUE (${endpoint.other})`);
      await db.query(`ALTER TABLE ${endpoint.table} DROP CONSTRAINT ${endpoint.table}_${endpoint.column}_fkey`);
      await db.query(
        `ALTER TABLE ${endpoint.table} ADD FOREIGN KEY (${endpoint.column}) REFERENCES ${endpoint.parent} (${endpoint.other})`
      );
      await expect(keysBelow(db, "issues")).rejects.toThrow(`the target of ${endpoint.table}.${endpoint.column}`);
    } finally {
      await db.close();
    }
  });
}
