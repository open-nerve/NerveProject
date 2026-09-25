import path from "node:path";

import { test as base } from "@playwright/test";

import { createApi, type Api } from "./api";
import { createDatabase, openDatabase, templateDatabase, type Database } from "./db";
import { nerveFixtureTimeoutMs, startNerve, type Nerve } from "./server";

export { expect } from "@playwright/test";

interface WorkerFixtures {
  /** The worker's own database, a copy of the migrated template, with a pool for the assertions. */
  db: Database;
  /** The worker's own nerve serve, on that database. */
  nerve: Nerve;
}

interface TestFixtures {
  /** The typed API client for the worker's nerve. */
  api: Api;
  /**
   * Starts another nerve on the worker's database, with extra variables such
   * as NERVE_AUTH__SIGNUP_ENABLED=false; it stops when the test ends. Each
   * start adds the nerve fixture's budget to the test's timeout, so the
   * fixture's own timeouts fire first.
   */
  nerveWith: (env: Record<string, string>) => Promise<Nerve>;
  /** When the test fails, a pg_dump of the worker's database joins its trace, screenshot and nerve log. */
  databaseSnapshot: void;
}

/** Stories import test from here: every worker runs its own nerve on its own database. */
export const test = base.extend<TestFixtures, WorkerFixtures>({
  db: [
    // oxlint-disable-next-line no-empty-pattern -- Playwright reads a fixture's dependencies from this pattern
    async ({}, use, workerInfo) => {
      const name = `e2e_w${workerInfo.workerIndex}`;
      await createDatabase(name, templateDatabase);
      const db = openDatabase(name);
      await use(db);
      await db.close();
    },
    { scope: "worker" },
  ],
  nerve: [
    async ({ db }, use, workerInfo) => {
      const logFile = path.join(workerInfo.project.outputDir, `nerve-w${workerInfo.workerIndex}.log`);
      const nerve = await startNerve(db.url, logFile);
      await use(nerve);
      await nerve.stop();
    },
    // Playwright's default worker-fixture budget (30 s, shared by setup and
    // teardown) is shorter than the fixture's own timeouts; give it room to
    // let those fire and report first.
    { scope: "worker", timeout: nerveFixtureTimeoutMs },
  ],
  // page and request resolve relative URLs against the worker's nerve.
  baseURL: async ({ nerve }, use) => {
    await use(nerve.baseURL);
  },
  api: async ({ nerve }, use) => {
    await use(createApi(nerve.baseURL));
  },
  nerveWith: async ({ db }, use, testInfo) => {
    const started: Nerve[] = [];
    await use(async (env) => {
      testInfo.setTimeout(testInfo.timeout + nerveFixtureTimeoutMs);
      const nerve = await startNerve(db.url, testInfo.outputPath(`nerve-${started.length + 1}.log`), env);
      started.push(nerve);
      return nerve;
    });
    await Promise.all(started.map((nerve) => nerve.stop()));
  },
  databaseSnapshot: [
    async ({ db }, use, testInfo) => {
      await use();
      if (testInfo.status !== testInfo.expectedStatus) {
        const file = testInfo.outputPath("database.sql");
        await db.dump(file);
        await testInfo.attach("database", { path: file, contentType: "application/sql" });
      }
    },
    { auto: true },
  ],
});
