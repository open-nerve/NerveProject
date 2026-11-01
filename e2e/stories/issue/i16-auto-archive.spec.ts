import { changeProject, createProject, createWorkspace, slugFor } from "../../fixtures/api";
import { login, newAccount, shownNameOf } from "../../fixtures/auth";
import { archiveWatermark, blockArchiveProject, completedPeriodicJobs } from "../../fixtures/auto-archive";
import { activityPage, createIssue, issueRead, listIssues, updateIssue } from "../../fixtures/issues";
import { expectListPageQuiet, issueListProject, openIssueList } from "../../fixtures/issue-list-page";
import { answerTo } from "../../fixtures/settings-pages";
import { listStates } from "../../fixtures/states";
import { clockTest as test, expect } from "../../fixtures/test";

test("I16 (API): a fresh auto-archive attempt archives only old terminal issues with one system activity", async ({
  clockWorld: world,
}, info) => {
  test.setTimeout(60_000);
  const db = world.db;
  const before = world.current.api;
  const admin = await newAccount(before, info, "archive-admin"); // no-expiry PAT, also after +744h
  const token = admin.token;
  const slug = slugFor(info);
  await createWorkspace(before, token, { name: "Auto archive", slug });
  const projects = await Promise.all(
    ["EARLY", "LATE"].map(async (identifier) => {
      const project = await createProject(before, token, slug, { name: identifier, identifier });
      expect((await changeProject(before, token, project.id, { archive_in: 1 })).status).toBe(200);
      return project;
    })
  );
  // Use the database's actual UUID order, the same order as ArchiveProjects.
  const ordered = await db.query<{ id: string }>(
    "SELECT id::text FROM projects WHERE id = ANY($1::uuid[]) ORDER BY id",
    [projects.map((p) => p.id)]
  );
  const early = ordered[0]!.id;
  const late = ordered[1]!.id;
  const states = await listStates(before, token, early);
  const state = (group: string) => {
    const found = states.find((s) => s.group === group);
    if (!found) throw new Error(`missing ${group} state`);
    return found.id;
  };
  const targets = await Promise.all(
    ["completed", "cancelled"].map((group) =>
      createIssue(before, token, early, { name: `old ${group}`, state_id: state(group) })
    )
  );
  const controls = await Promise.all(
    ["backlog", "unstarted", "started"].map((group) =>
      createIssue(before, token, early, { name: `old ${group}`, state_id: state(group) })
    )
  );
  const toComplete = await createIssue(before, token, early, {
    name: "old item completed recently",
    state_id: state("started"),
  });
  const twoMonths = await createProject(before, token, slug, { name: "Two months", identifier: "TWO" });
  expect((await changeProject(before, token, twoMonths.id, { archive_in: 2 })).status).toBe(200);
  const twoDone = (await listStates(before, token, twoMonths.id)).find((s) => s.group === "completed")!;
  const twoOld = await createIssue(before, token, twoMonths.id, {
    name: "31 days is not two months",
    state_id: twoDone.id,
  });

  // A historical completed task is a separate negative control from the crossing attempt below.
  let historical: string | undefined;
  await expect
    .poll(
      async () => {
        const rows = await db.query<{ id: string }>(
          "SELECT id::text FROM river_job WHERE kind='issue.auto_archive' AND state='completed' ORDER BY id LIMIT 1"
        );
        historical = rows[0]?.id;
        return rows.length;
      },
      { timeout: 10_000 }
    )
    .toBe(1);
  await world.current.nerve.stop(); // do not let the old process encounter our blocker during shutdown
  const blocker = await blockArchiveProject(db, late);
  let watermark: string;
  let crossing: string;
  let recentStarted: string;
  try {
    await world.advance("744h");
    crossing = await blocker.waitForAttempt();
    // The worker has scanned early, and is genuinely running, blocked on late.
    // All recent writes are made now, on the already-scanned, unlocked project.
    recentStarted = await archiveWatermark(db);
    const recent = await createIssue(world.current.api, token, early, {
      name: "recently created completed",
      state_id: state("completed"),
    });
    controls.push(recent);
    controls.push(await updateIssue(world.current.api, token, toComplete.id, { state_id: state("completed") }));
    watermark = await archiveWatermark(db); // the final successful recent write precedes this DB timestamp
    expect(
      await completedPeriodicJobs(db, "issue.auto_archive", watermark, world.current.startedAt, historical),
      "historical completed must not satisfy freshness"
    ).toEqual([]);
  } finally {
    await blocker.release();
  }

  await expect
    .poll(
      async () => (await db.query("SELECT id FROM river_job WHERE id=$1 AND state='completed'", [crossing])).length,
      { timeout: 10_000, intervals: [20, 50, 100] }
    )
    .toBe(1);
  const crossingRows = await db.query(
    `SELECT id::text, attempted_at::text, finalized_at::text,
            attempted_at < $2::timestamptz AS began_before_recent,
            finalized_at > $3::timestamptz AS ended_after_watermark,
            finalized_at > $4::timestamptz AS ended_after_spawn
     FROM river_job WHERE id=$1`,
    [crossing, recentStarted, watermark, world.current.startedAt.toISOString()]
  );
  expect(crossingRows).toEqual([
    expect.objectContaining({ began_before_recent: true, ended_after_watermark: true, ended_after_spawn: true }),
  ]);
  console.log(
    "I16 crossing attempt",
    JSON.stringify({ recentStarted, watermark, startedAt: world.current.startedAt, rows: crossingRows })
  );
  // Merely finalized-after-watermark would accept this real, too-early attempt.
  expect(
    await completedPeriodicJobs(db, "issue.auto_archive", watermark, world.current.startedAt, crossing),
    "the crossing attempt must not satisfy freshness"
  ).toEqual([]);
  await expect
    .poll(
      async () => (await completedPeriodicJobs(db, "issue.auto_archive", watermark, world.current.startedAt)).length,
      { timeout: 15_000 }
    )
    .toBeGreaterThan(0);
  const fresh = await completedPeriodicJobs(db, "issue.auto_archive", watermark, world.current.startedAt);
  console.log("I16 fresh attempts", JSON.stringify({ watermark, fresh }));
  await info.attach("auto-archive-attempts", {
    body: JSON.stringify(
      { watermark, startedAt: world.current.startedAt, historical, crossing: crossingRows, fresh },
      null,
      2
    ),
    contentType: "application/json",
  });

  const api = world.current.api;
  const archived = await listIssues(api, token, early, { archived: "true" });
  expect(archived.data?.map((row) => row.id).toSorted()).toEqual(targets.map((row) => row.id).toSorted());
  const active = await listIssues(api, token, early);
  expect(active.data?.map((row) => row.id).toSorted()).toEqual(controls.map((row) => row.id).toSorted());
  expect((await listIssues(api, token, twoMonths.id)).data?.map((row) => row.id)).toEqual([twoOld.id]);
  expect((await listIssues(api, token, twoMonths.id, { archived: "true" })).data).toEqual([]);
  await Promise.all(
    targets.map(async (target) => {
      expect(await issueRead(api, token, target.id)).toMatchObject({
        status: 200,
        issue: { archived_at: expect.any(String), updated_by_id: null },
      });
      const activities = (await activityPage(api, token, target.id, undefined, 100)).data.filter(
        (row) => row.field === "archived_at"
      );
      expect(activities).toEqual([
        expect.objectContaining({ actor_id: null, new_value: "archive", old_value: null, verb: "updated" }),
      ]);
      expect(
        await db.query(
          `SELECT i.updated_by_id, a.actor_id, a.created_by_id, a.updated_by_id AS activity_updated_by,
              i.archived_at = i.updated_at AND a.created_at = i.archived_at AND a.updated_at = i.archived_at AS same_stamp
       FROM issues i JOIN issue_activities a ON a.issue_id=i.id WHERE i.id=$1 AND a.field='archived_at'`,
          [target.id]
        )
      ).toEqual([
        { updated_by_id: null, actor_id: null, created_by_id: null, activity_updated_by: null, same_stamp: true },
      ]);
    })
  );
  await Promise.all(
    [...controls, twoOld].map(async (control) => {
      expect(await issueRead(api, token, control.id)).toMatchObject({
        status: 200,
        issue: { archived_at: null, updated_by_id: admin.id },
      });
      expect(
        await db.query("SELECT id FROM issue_activities WHERE issue_id=$1 AND field='archived_at'", [control.id])
      ).toEqual([]);
    })
  );
});

test("I16 (page): fresh clock process archives terminal issues and shows the system actor", async ({
  clockWorld: world,
  browser,
}, info) => {
  test.setTimeout(60_000);
  const before = world.current;
  const seed = await issueListProject(before.api, info);
  expect((await changeProject(before.api, seed.member.token, seed.project.id, { archive_in: 1 })).status).toBe(200);
  const completed = seed.states.Done;
  if (!completed) throw new Error("missing completed state");
  const old = await createIssue(before.api, seed.member.token, seed.project.id, {
    name: "Old completed",
    state_id: completed,
  });
  const active = await createIssue(before.api, seed.member.token, seed.project.id, { name: "Old active" });
  const changed = await createIssue(before.api, seed.member.token, seed.project.id, { name: "Completed recently" });
  const firstContext = await browser.newContext({ baseURL: before.nerve.baseURL });
  try {
    const first = await firstContext.newPage();
    const { watch } = await openIssueList(first, seed);
    await expect(first.locator(`#issue-${old.id}`)).toBeVisible();
    await expectListPageQuiet(first, watch);
  } finally {
    await firstContext.close();
  }
  const advanced = await world.advance("744h");
  expect(before.nerve.exitCode).toBe(0);
  expect(advanced.nerve).not.toBe(before.nerve);
  // Fresh real login after advancing: the old access/refresh tokens cannot stand in for a new session.
  const tokens = await login(advanced.api, seed.member.email);
  const recent = await createIssue(advanced.api, tokens.access_token, seed.project.id, {
    name: "New completed",
    state_id: completed,
  });
  await updateIssue(advanced.api, tokens.access_token, changed.id, { state_id: completed });
  const watermark = await archiveWatermark(world.db);
  await expect
    .poll(
      async () => (await completedPeriodicJobs(world.db, "issue.auto_archive", watermark, advanced.startedAt)).length,
      { timeout: 15_000 }
    )
    .toBeGreaterThan(0);
  const fresh = await completedPeriodicJobs(world.db, "issue.auto_archive", watermark, advanced.startedAt);
  const freshness = await world.db.query<{ after_spawn: boolean; after_writes: boolean }>(
    "SELECT attempted_at > $2::timestamptz AS after_spawn, attempted_at > $3::timestamptz AS after_writes FROM river_job WHERE id = ANY($1::bigint[])",
    [fresh.map((job) => job.id), advanced.startedAt.toISOString(), watermark]
  );
  expect(freshness.length).toBeGreaterThan(0);
  for (const row of freshness) expect(row).toEqual({ after_spawn: true, after_writes: true });
  expect(
    (await listIssues(advanced.api, tokens.access_token, seed.project.id, { archived: "true" })).data?.map(
      (row) => row.id
    )
  ).toEqual([old.id]);
  expect(
    (await listIssues(advanced.api, tokens.access_token, seed.project.id)).data?.map((row) => row.id).toSorted()
  ).toEqual([active.id, changed.id, recent.id].toSorted());
  const context = await browser.newContext({ baseURL: advanced.nerve.baseURL });
  try {
    const page = await context.newPage();
    const archives = `/${seed.slug}/projects/${seed.project.id}/archives/issues`;
    const { watch } = await openIssueList(page, { ...seed, path: archives });
    await expect(page.locator(`#issue-${old.id}`)).toBeVisible();
    await expect(page.locator('[id^="issue_"]')).toHaveCount(1);
    await page.locator(`#issue-${old.id}`).click();
    await page.getByRole("link", { name: "Open work item in full screen", exact: true }).click();
    await expect(page).toHaveURL(`/${seed.slug}/browse/WEB-${old.sequence_id}`);
    await expect(page.getByText(old.name, { exact: true })).toBeVisible();
    await expect(page.locator('textarea, [contenteditable="true"]')).toHaveCount(0);
    const archived = page.getByText("archived the work item.", { exact: true }).locator("..");
    await expect(archived.getByText("Nerve", { exact: true })).toBeVisible();
    await expect(archived.getByRole("link")).toHaveCount(0);
    await expect(archived.getByText("Deactivated user", { exact: true })).toHaveCount(0);
    await info.attach("system-archive-timeline", { body: await page.screenshot(), contentType: "image/png" });
    await page.getByRole("button", { name: `Options: ${old.name}`, exact: true }).click();
    expect(
      (
        await answerTo(page, "POST", `/api/v0/issues/${old.id}/unarchive`, () =>
          page.getByRole("menuitem", { name: "Restore", exact: true }).click()
        )
      ).status()
    ).toBe(200);
    await expect(page).toHaveURL(`/${seed.slug}/browse/WEB-${old.sequence_id}`);
    const restored = page.getByText("restored the work item.", { exact: true }).locator("..");
    await expect(restored.getByRole("link", { name: shownNameOf(seed.member.email), exact: true })).toBeVisible();
    await expect(restored.getByText("Nerve", { exact: true })).toHaveCount(0);
    const rows = await world.db.query<{ actor_id: string | null; new_value: string }>(
      "SELECT actor_id::text, new_value FROM issue_activities WHERE issue_id=$1 AND field='archived_at' ORDER BY created_at, id",
      [old.id]
    );
    expect(rows).toEqual([
      { actor_id: null, new_value: "archive" },
      { actor_id: seed.member.id, new_value: "restore" },
    ]);
    await info.attach("page-auto-archive", {
      body: JSON.stringify(
        {
          oldURL: before.nerve.baseURL,
          currentURL: advanced.nerve.baseURL,
          startedAt: advanced.startedAt,
          watermark,
          fresh,
          freshness,
          rows,
        },
        null,
        2
      ),
      contentType: "application/json",
    });
    await info.attach("restored-timeline", { body: await page.screenshot(), contentType: "image/png" });
    await expectListPageQuiet(page, watch);
  } finally {
    await context.close();
  }
});
