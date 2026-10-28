import { changeProject, createProject, createWorkspace, slugFor } from "../../fixtures/api";
import { newAccount } from "../../fixtures/auth";
import { archiveWatermark, blockArchiveProject, completedPeriodicJobs } from "../../fixtures/auto-archive";
import { activityPage, createIssue, issueRead, listIssues, updateIssue } from "../../fixtures/issues";
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
