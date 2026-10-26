import { createHash } from "node:crypto";
import { mkdir, readFile, stat } from "node:fs/promises";

import { expect, test as base, type TestInfo } from "@playwright/test";

import { register } from "../../fixtures/auth";
import { createClockWorld } from "../../fixtures/clock";
import { createDatabase, openDatabase, templateDatabase } from "../../fixtures/db";
import { applicationName, startNerve, type Nerve } from "../../fixtures/server";
import { clockTest } from "../../fixtures/test";

function databaseName(info: TestInfo, label: string): string {
  const hash = createHash("sha256")
    .update(`${info.testId}:${info.repeatEachIndex}:${info.retry}`)
    .digest("hex")
    .slice(0, 12);
  return `e2e_${label}_w${info.workerIndex}_r${info.retry}_${hash}`;
}

// oxlint-disable-next-line no-empty-pattern -- Playwright supplies testInfo after its fixture arguments
base("clock world switches processes on its own database and advances business time", async ({}, info) => {
  const controlName = databaseName(info, "control");
  const clockName = databaseName(info, "clock");
  await createDatabase(controlName, templateDatabase);
  const control = openDatabase(controlName);
  let controlNerve: Nerve | undefined;
  let first: Nerve | undefined;
  let world: Awaited<ReturnType<typeof createClockWorld>> | undefined;
  try {
    controlNerve = await startNerve(control.url, info.outputPath("control.log"));
    const beforeSpawn = new Date();
    world = await createClockWorld({
      databaseName: clockName,
      logFile: (generation) => info.outputPath(`clock-${generation}.log`),
    });
    first = world.current.nerve;
    const log = await stat(info.outputPath("clock-0.log"));
    expect(world.current.startedAt.getTime()).toBeGreaterThanOrEqual(beforeSpawn.getTime());
    expect(world.current.startedAt.getTime()).toBeLessThanOrEqual(log.birthtimeMs);
    const firstEmail = `before-${databaseName(info, "user")}@example.com`;
    await register(world.current.api, firstEmail);
    const firstAt = await world.db.query<{ created_at: Date }>("SELECT created_at FROM users WHERE email=$1", [
      firstEmail,
    ]);
    expect(firstAt).toHaveLength(1);
    expect(firstAt[0]!.created_at.getTime()).toBeGreaterThanOrEqual(world.current.startedAt.getTime());
    expect(await control.query("SELECT email FROM users WHERE email=$1", [firstEmail])).toHaveLength(0);
    expect(world.db.name).toBe(clockName);
    expect(world.db.name).not.toBe(control.name);

    const oldURL = first.baseURL;
    const second = await world.advance("24h");
    expect(second.nerve.baseURL).not.toBe(oldURL);
    expect(second.startedAt.getTime()).toBeLessThanOrEqual((await stat(info.outputPath("clock-1.log"))).birthtimeMs);
    await expect
      .poll(async () =>
        fetch(`${oldURL}/readyz`).then(
          (r) => r.status,
          () => 0
        )
      )
      .toBe(0);
    const secondEmail = `after-${databaseName(info, "user")}@example.com`;
    await register(second.api, secondEmail);
    const secondAt = await world.db.query<{ created_at: Date }>("SELECT created_at FROM users WHERE email=$1", [
      secondEmail,
    ]);
    expect(secondAt).toHaveLength(1);
    expect(secondAt[0]!.created_at.getTime() - firstAt[0]!.created_at.getTime()).toBeGreaterThan(23 * 60 * 60 * 1000);
    expect(await control.query("SELECT email FROM users WHERE email=$1", [secondEmail])).toHaveLength(0);
  } finally {
    try {
      await first?.stop(); // also owns the first process if advance forgot to stop it
    } finally {
      try {
        await world?.close();
      } finally {
        try {
          await controlNerve?.stop();
        } finally {
          try {
            if (world) {
              const active = await control.query<{ count: number }>(
                "SELECT count(*)::int AS count FROM pg_stat_activity WHERE datname=$1 AND application_name=$2",
                [clockName, applicationName]
              );
              expect(active[0]?.count).toBe(0);
            }
          } finally {
            await control.close();
          }
        }
      }
    }
  }
});

// oxlint-disable-next-line no-empty-pattern -- Playwright supplies testInfo after its fixture arguments
base("failed initial startup dumps its own database and closes its connection", async ({}, info) => {
  const name = databaseName(info, "startup");
  const logFile = info.outputPath("blocked.log");
  await mkdir(logFile); // openSync cannot use a directory as the child's log
  await expect(createClockWorld({ databaseName: name, logFile: () => logFile })).rejects.toThrow();
  expect(await readFile(`${logFile}.sql`, "utf8")).toContain("goose_db_version");
  const db = openDatabase(name);
  try {
    const rows = await db.query<{ count: number }>(
      "SELECT count(*)::int AS count FROM pg_stat_activity WHERE datname=current_database() AND application_name=$1",
      [applicationName]
    );
    expect(rows[0]?.count).toBe(0);
  } finally {
    await db.close();
  }
});

// oxlint-disable-next-line no-empty-pattern -- Playwright supplies testInfo after its fixture arguments
base("invalid clock offset closes the process and assertion pool after advance fails", async ({}, info) => {
  const name = databaseName(info, "badclock");
  const world = await createClockWorld({
    databaseName: name,
    logFile: (generation) => info.outputPath(`bad-clock-${generation}.log`),
  });
  const first = world.current.nerve;
  try {
    const email = `failure-${databaseName(info, "user")}@example.com`;
    await register(world.current.api, email);
    await expect(world.advance("invalid-offset")).rejects.toThrow();
    expect(await readFile(`${info.outputPath("bad-clock-1.log")}.sql`, "utf8")).toContain(email);
    expect((await stat(info.outputPath("bad-clock-0.log"))).size).toBeGreaterThan(0);
    expect((await stat(info.outputPath("bad-clock-1.log"))).size).toBeGreaterThan(0);
    const sessions = await openDatabase(name);
    try {
      const rows = await sessions.query<{ count: number }>(
        "SELECT count(*)::int AS count FROM pg_stat_activity WHERE datname=current_database() AND application_name=$1",
        [applicationName]
      );
      expect(rows[0]?.count).toBe(0);
    } finally {
      await sessions.close();
    }
    await expect(world.db.query("SELECT 1")).rejects.toThrow();
  } finally {
    try {
      await first.stop();
    } finally {
      await world.close();
    }
  }
});

clockTest("clock fixture owns its database during use", async ({ clockWorld }, info) => {
  expect(clockWorld.db.name).toBe(databaseName(info, "clock"));
  const email = `fixture-${databaseName(info, "user")}@example.com`;
  await register(clockWorld.current.api, email);
  expect(await clockWorld.db.query("SELECT email FROM users WHERE email=$1", [email])).toHaveLength(1);
});
