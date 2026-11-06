/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { createHash, generateKeyPairSync } from "node:crypto";
import { existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import path from "node:path";

import { test as base } from "@playwright/test";

import { createApi, type Api } from "./api";
import { password } from "./auth";
import { createDatabase, openDatabase, type Database } from "./db";
import { nerveFixtureTimeoutMs, runNerve, startNerve, type Nerve } from "./server";

/** One generation of the deployment's nerve serve. */
export interface Deployed {
  readonly nerve: Nerve;
  readonly api: Api;
  readonly log: string;
}

/**
 * nerve deployed as README's deployment section says, on an empty database of its own: production, the base
 * configuration's files (local, shipped limits), a signing key made by Node's crypto, an exclusive storage root,
 * `nerve migrate up`, then `nerve users create` for the first account, since production closes sign-up.
 */
export interface Deployment {
  readonly db: Database;
  /** The first account's address; its password is the fixtures' password. */
  readonly email: string;
  /** Every variable the deployment sets; the fixture adds only the database URL and the listening address. */
  readonly variables: Readonly<Record<string, string>>;
  readonly current: Deployed;
  /** Stops nerve and starts it again on the same address, database, key and storage root. */
  restart(): Promise<Deployed>;
}

/** The record nerve serve logs first: its effective configuration, secrets masked. */
export function loggedConfiguration(log: string): Record<string, unknown> {
  for (const line of readFileSync(log, "utf8").split("\n")) {
    if (!line.startsWith("{")) continue;
    const record = JSON.parse(line) as { msg?: string; config?: Record<string, unknown> };
    if (record.msg === "configuration loaded" && record.config) return record.config;
  }
  throw new Error(`no configuration record in ${log}`);
}

export const deploymentTest = base.extend<{ deployment: Deployment }>({
  deployment: [
    // oxlint-disable-next-line no-empty-pattern -- Playwright reads a fixture's dependencies from this pattern
    async ({}, use, info) => {
      const hash = createHash("sha256")
        .update(`${info.testId}:${info.repeatEachIndex}:${info.retry}`)
        .digest("hex")
        .slice(0, 12);
      const name = `e2e_demo_w${info.workerIndex}_r${info.retry}_${hash}`;
      const url = await createDatabase(name);
      const db = openDatabase(name);
      const dir = info.outputPath("deployment");
      mkdirSync(dir, { recursive: true });
      const keyFile = path.join(dir, "nerve-jwt.pem");
      writeFileSync(keyFile, generateKeyPairSync("ed25519").privateKey.export({ type: "pkcs8", format: "pem" }), {
        mode: 0o600,
      });
      const variables = {
        NERVE_ENV: "prod",
        NERVE_AUTH__JWT__PRIVATE_KEY_FILE: keyFile,
        NERVE_FILES__LOCAL__ROOT: path.join(dir, "files"),
      };
      const deployed = { deployment: true };
      const logs: string[] = [];
      const start = async (address?: string): Promise<Deployed> => {
        const log = path.join(dir, `nerve-${logs.length}.log`);
        logs.push(log);
        const env = address ? { ...variables, NERVE_SERVER__ADDR: address } : variables;
        const nerve = await startNerve(url, log, env, deployed);
        return { nerve, api: createApi(nerve.baseURL), log };
      };
      const email = `owner-${hash}@example.com`;
      let current: Deployed | undefined;
      const errors: unknown[] = [];
      try {
        await runNerve(["migrate", "up"], url, "", variables, deployed);
        await runNerve(["users", "create", "--email", email], url, `${password}\n`, variables, deployed);
        current = await start();
        await use({
          db,
          email,
          variables,
          get current() {
            if (!current) throw new Error("the deployment is stopped");
            return current;
          },
          async restart() {
            if (!current) throw new Error("the deployment is stopped");
            const address = new URL(current.nerve.baseURL).host;
            await current.nerve.stop();
            current = undefined;
            current = await start(address);
            return current;
          },
        });
      } catch (error) {
        errors.push(error);
      } finally {
        try {
          await current?.nerve.stop();
        } catch (error) {
          errors.push(error);
        }
        for (const log of logs)
          if (existsSync(log)) await info.attach(path.basename(log), { path: log, contentType: "text/plain" });
        if (errors.length || info.status !== info.expectedStatus) {
          const file = info.outputPath("deployment.sql");
          await db.dump(file).then(
            () => info.attach("deployment-database", { path: file, contentType: "application/sql" }),
            (error: unknown) => errors.push(error)
          );
        }
        await db.close();
      }
      if (errors.length === 1) throw errors[0];
      if (errors.length > 1) throw new AggregateError(errors, "deployment fixture and its cleanup failed");
    },
    // Setup migrates an empty database; failure artifacts may need pg_dump before teardown.
    { timeout: nerveFixtureTimeoutMs + 60_000 },
  ],
});
