import { readdirSync } from "node:fs";
import path from "node:path";

import { applicationName, runNerve } from "../../fixtures/server";
import { expect, test } from "../../fixtures/test";

/** The migration files bin/nerve embeds, in order. */
const migrationFiles = readdirSync(path.resolve(import.meta.dirname, "../../../server/migrations/sql"))
  .filter((name) => name.endsWith(".sql"))
  .toSorted();

test("S1: an operator starts nerve with the test configuration and it becomes ready", async ({ request, db }) => {
  const healthz = await request.get("/healthz");
  expect(healthz.status()).toBe(200);
  expect(await healthz.json()).toEqual({ status: "ok" });

  const readyz = await request.get("/readyz");
  expect(readyz.status()).toBe(200);
  expect(await readyz.json()).toEqual({ status: "ok" });

  // Since /readyz, nerve keeps a pooled connection to this worker's database.
  const [sessions] = await db.query<{ count: number }>(
    "SELECT count(*)::int AS count FROM pg_stat_activity WHERE datname = current_database() AND application_name = $1",
    [applicationName]
  );
  expect(sessions?.count).toBeGreaterThan(0);

  // Every migration file is applied, and the database's latest version is
  // the last file's.
  expect(migrationFiles.length).toBeGreaterThan(0);
  const { stdout } = await runNerve(["migrate", "status"], db.url);
  const [header, ...rows] = stdout.trimEnd().split("\n");
  expect(header?.split(/\s{2,}/)).toEqual(["VERSION", "STATE", "APPLIED AT", "SOURCE"]);
  expect(rows.map((row) => row.split(/\s+/)).map(([, state, , source]) => [state, source])).toEqual(
    migrationFiles.map((file) => ["applied", file])
  );
  const [latest] = await db.query<{ version: number }>("SELECT max(version_id)::int AS version FROM goose_db_version");
  expect(latest?.version).toBe(Number.parseInt(migrationFiles.at(-1) ?? "", 10));
});
