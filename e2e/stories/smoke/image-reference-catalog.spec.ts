import { keysBelow } from "../../fixtures/assert/catalog";
import { clockTest as test, expect } from "../../fixtures/test";

test("image reference catalog keeps image owners outside the issue ownership graph and retains unknown descendants", async ({
  clockWorld,
}) => {
  const db = clockWorld.db;
  const keys = await keysBelow(db, "issues");
  expect(keys.filter(({ child }) => ["users", "workspaces", "projects"].includes(child))).toEqual([]);
  expect(keys).toEqual(
    expect.arrayContaining([
      { child: "file_assets", column: "issue_id", parent: "issues", target: "id" },
      { child: "file_assets", column: "comment_id", parent: "issue_comments", target: "id" },
      { child: "comment_reactions", column: "comment_id", parent: "issue_comments", target: "id" },
    ])
  );
  await db.query("CREATE TABLE image_catalog_descendant(id uuid PRIMARY KEY, issue_id uuid REFERENCES issues(id))");
  try {
    expect(await keysBelow(db, "issues")).toContainEqual({
      child: "image_catalog_descendant",
      column: "issue_id",
      parent: "issues",
      target: "id",
    });
  } finally {
    await db.query("DROP TABLE image_catalog_descendant");
  }
});

for (const action of ["CASCADE", "SET NULL", "RESTRICT", "NO ACTION DEFERRABLE"]) {
  test(`image reference catalog rejects unsafe ${action} before excluding any edge`, async ({ clockWorld }) => {
    const db = clockWorld.db;
    await db.query(`ALTER TABLE users DROP CONSTRAINT users_avatar_asset_fk,
      ADD CONSTRAINT users_avatar_asset_fk FOREIGN KEY(avatar_asset_id) REFERENCES file_assets(id) ON DELETE ${action}`);
    try {
      await expect(keysBelow(db, "issues"), action).rejects.toThrow();
    } finally {
      await db.query(`ALTER TABLE users DROP CONSTRAINT users_avatar_asset_fk,
        ADD CONSTRAINT users_avatar_asset_fk FOREIGN KEY(avatar_asset_id) REFERENCES file_assets(id) ON DELETE NO ACTION`);
    }
  });
}

test("image reference catalog rejects missing, unindexed and unknown references before excluding any edge", async ({
  clockWorld,
}) => {
  const db = clockWorld.db;
  await db.query("ALTER TABLE users DROP CONSTRAINT users_cover_image_asset_fk");
  try {
    await expect(keysBelow(db, "issues"), "missing required reference").rejects.toThrow();
  } finally {
    await db.query(`ALTER TABLE users ADD CONSTRAINT users_cover_image_asset_fk
      FOREIGN KEY(cover_image_asset_id) REFERENCES file_assets(id) ON DELETE NO ACTION`);
  }
  await db.query("DROP INDEX workspaces_logo_asset_id_idx");
  try {
    await expect(keysBelow(db, "issues"), "missing unconditional index").rejects.toThrow();
  } finally {
    await db.query("CREATE INDEX workspaces_logo_asset_id_idx ON workspaces(logo_asset_id)");
  }
  await db.query(`CREATE TABLE image_catalog_probe(id uuid PRIMARY KEY,
    asset_id uuid CONSTRAINT users_avatar_asset_fk REFERENCES file_assets(id))`);
  try {
    await expect(keysBelow(db, "issues"), "same name on unknown table is not allowed").rejects.toThrow();
  } finally {
    await db.query("DROP TABLE image_catalog_probe");
  }
  expect((await keysBelow(db, "issues")).some(({ child }) => child === "file_assets")).toBe(true);
});
