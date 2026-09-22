import path from "node:path";

import { test as base } from "@playwright/test";

import { createApi, type Api } from "./api";
import { createDatabase, templateDatabase, type Database } from "./db";
import { nerveFixtureTimeoutMs, startNerve, type Nerve } from "./server";

export { expect } from "@playwright/test";

interface WorkerFixtures {
  /** The worker's own database, a copy of the migrated template. */
  db: Database;
  /** The worker's own nerve serve, on that database. */
  nerve: Nerve;
}

interface TestFixtures {
  /** The typed API client for the worker's nerve. */
  api: Api;
}

/** Stories import test from here: every worker runs its own nerve on its own database. */
export const test = base.extend<TestFixtures, WorkerFixtures>({
  db: [
    // oxlint-disable-next-line no-empty-pattern -- Playwright reads a fixture's dependencies from this pattern
    async ({}, use, workerInfo) => {
      await use(await createDatabase(`e2e_w${workerInfo.workerIndex}`, templateDatabase));
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
});
