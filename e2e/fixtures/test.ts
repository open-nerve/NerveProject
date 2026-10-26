import path from "node:path";
import { createHash } from "node:crypto";
import { existsSync } from "node:fs";

import { test as base, type Page } from "@playwright/test";

import { createApi, type Api } from "./api";
import { createClockWorld, type ClockWorld } from "./clock";
import { signInContext, type AuthTokens } from "./auth";
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
  /**
   * Signs the test's browser context in with tokens, at the worker's nerve or at baseURL (M2 design 9.5),
   * and returns the test's page, which has loaded nothing yet: its first page refreshes the session.
   */
  signedInPage: (tokens: AuthTokens, baseURL?: string) => Promise<Page>;
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
  signedInPage: async ({ page, nerve }, use) => {
    await use(async (tokens, baseURL = nerve.baseURL) => {
      await signInContext(page.context(), baseURL, tokens);
      return page;
    });
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

/** Clock stories own their database, process generations and failure artifacts. */
export const clockTest = base.extend<{ clockWorld: ClockWorld }>({
  clockWorld: [
    // oxlint-disable-next-line no-empty-pattern -- Playwright reads fixture dependencies from this pattern
    async ({}, use, testInfo) => {
      const hash = createHash("sha256")
        .update(`${testInfo.testId}:${testInfo.repeatEachIndex}:${testInfo.retry}`)
        .digest("hex")
        .slice(0, 12);
      const databaseName = `e2e_clock_w${testInfo.workerIndex}_r${testInfo.retry}_${hash}`;
      const logFile = (generation: number) => testInfo.outputPath(`clock-${generation}.log`);
      let generation = 0;
      let world: ClockWorld | undefined;
      let failed = false;
      const errors: unknown[] = [];
      try {
        world = await createClockWorld({
          databaseName,
          logFile: (number) => {
            generation = number + 1;
            return logFile(number);
          },
        });
        await use(world);
        failed = testInfo.status !== testInfo.expectedStatus;
      } catch (error) {
        failed = true;
        errors.push(error);
      } finally {
        try {
          if (failed) {
            // Each artifact is independent: a failed dump/attachment must not
            // prevent process logs from being attached or hide the original error.
            const diagnostics = await Promise.allSettled([
              (async () => {
                const db = world?.db ?? openDatabase(databaseName);
                try {
                  const file = testInfo.outputPath("clock-database.sql");
                  await db.dump(file);
                  await testInfo.attach("clock-database", { path: file, contentType: "application/sql" });
                } finally {
                  if (!world) await db.close();
                }
              })(),
              ...Array.from({ length: generation }, async (_, number) => {
                const file = logFile(number);
                if (existsSync(file))
                  await testInfo.attach(`clock-${number}`, { path: file, contentType: "text/plain" });
              }),
            ]);
            for (const result of diagnostics) if (result.status === "rejected") errors.push(result.reason);
          }
        } finally {
          try {
            await world?.close();
          } catch (error) {
            errors.push(error);
          }
        }
      }
      if (errors.length === 1) throw errors[0];
      if (errors.length > 1) throw new AggregateError(errors, "clock fixture and diagnostics failed");
    },
    // Setup may wait for readiness; failure artifacts may need pg_dump before teardown.
    { timeout: nerveFixtureTimeoutMs + 110_000 },
  ],
});
