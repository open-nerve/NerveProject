import { applicationName, runNerve } from "../../fixtures/server";
import { expect, test } from "../../fixtures/test";

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

  // M0 has no migration files; from M2 on this lists them as applied.
  const { stdout } = await runNerve(["migrate", "status"], db.url);
  expect(stdout).toBe("no migrations\n");
});
