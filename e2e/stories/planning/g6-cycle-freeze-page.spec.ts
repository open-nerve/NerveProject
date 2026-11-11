import {
  databaseNow,
  expectCountedInDatabase,
  freezeRanSince,
  frozenSnapshot,
  snapshotOf,
} from "../../fixtures/assert/progress";
import { expectQuietConsole, watchPage } from "../../fixtures/browser";
import { cyclePath, cycleRow, cyclesPath, openCyclesList, sectionToggle } from "../../fixtures/cycle-pages";
import { pageSignIn, rowsOf, sidebarDetail, transferEntry } from "../../fixtures/cycle-progress-pages";
import { createCycle } from "../../fixtures/cycles";
import { createIssue, updateIssue } from "../../fixtures/issues";
import {
  burndown,
  byAssignee,
  byLabel,
  counts,
  cycleProgress,
  endCycle,
  progressWorld,
  type Progress,
} from "../../fixtures/planning-progress";
import { expect, test } from "../../fixtures/test";

// G6's page version (M6 P5a spec 2.3, 2.5, 2.9, R7): an ended cycle's progress, frozen by the periodic freeze (every
// 2s in the tests), as its list's row and its sidebar show it to its member in his browser, once its work items have
// changed. The API version is g6-cycle-freeze.spec.ts; both assert the same counts and the same frozen column. Web
// keeps the time of a zone where it is about noon as the story starts (noonZone).

test("G6 (page): Sprint, of four work items, one of each group but the backlog, ends and the periodic freeze writes its progress, 1 of the 3 not cancelled done; mem's list shows its row at 33% with a transfer of the two unfinished, its sidebar 1/3 and each group's share; mem completes Fix the login and takes Plan out, nerve's live progress then 2 of 2 and its frozen column unchanged after another run of the freeze: reloaded, the sidebar and the list's row stay on the frozen numbers, while the row's transfer goes with the live unfinished ones", async ({
  api,
  db,
  signedInPage,
}, testInfo) => {
  // two runs of the freeze to wait for, and three loads of the page
  test.setTimeout(60_000);
  const { day, admin, mem, walt, slug, web, states, bug } = await progressWorld(api, testInfo);
  const sprint = await createCycle(api, admin.token, web.id, { name: "Sprint", start_date: day(-3), end_date: day(3) });

  // Fix the login, to do, mem's, Bug; Write the docs, done, mem's; walt's Plan, in progress; Drop IE, cancelled. mem
  // puts each in Sprint, which the admin then ends four days ago.
  const fix = await createIssue(api, admin.token, web.id, {
    name: "Fix the login",
    state_id: states.todo,
    assignee_ids: [mem.id],
    label_ids: [bug.id],
  });
  const docs = await createIssue(api, admin.token, web.id, {
    name: "Write the docs",
    state_id: states.done,
    assignee_ids: [mem.id],
  });
  const plan = await createIssue(api, walt.token, web.id, { name: "Plan", state_id: states.started });
  const drop = await createIssue(api, admin.token, web.id, { name: "Drop IE", state_id: states.cancelled });
  expect(
    await Promise.all(
      [fix, docs, plan, drop].map(
        async (one) => (await updateIssue(api, mem.token, one.id, { cycle_id: sprint.id })).cycle_id
      )
    ),
    "the work items in Sprint"
  ).toEqual(Array(4).fill(sprint.id));
  await endCycle(api, admin.token, sprint.id, day(-10), day(-4));

  // The periodic freeze writes Sprint's progress, polled in the database: the numbers it has as it ends.
  const frozenNumbers: Progress = {
    ...counts({ unstarted: 1, started: 1, completed: 1, cancelled: 1 }),
    assignees: byAssignee([mem.id, 1, 1], [null, 0, 1]),
    labels: byLabel([bug.id, 0, 1], [null, 1, 1]),
    burndown: burndown(day(-10), day(-4), 3, 3, 3, 3, 3, 3, 3, 3),
  };
  const column = await frozenSnapshot(db, sprint.id);
  expect(column, "Sprint's frozen column").toEqual({ ...frozenNumbers, frozen_at: expect.any(String) });
  await expectCountedInDatabase(db, { cycle: sprint.id }, frozenNumbers);

  // mem's list: Sprint, completed, at 33%, 1 of its 3 not cancelled done, and its two unfinished to transfer. Its
  // page's sidebar: 1/3, and each group's share of the three.
  const page = await signedInPage(await pageSignIn(api, mem.email));
  const watch = await watchPage(page);
  await page.goto(cyclesPath(slug, web.id));
  await sectionToggle(page, "Completed cycle").click();
  const row = cycleRow(page, sprint.id, "Sprint");
  const ring = row.getByRole("progressbar", { name: "Progress" });
  await expect(ring).toHaveAttribute("aria-valuenow", "33");
  await expect(transferEntry(row)).toHaveText("Transfer 2 work items");
  await row.locator(`a[href="${cyclePath(slug, web.id, sprint.id)}"]`).click();
  await expect(page.getByRole("heading", { name: "Sprint", exact: true })).toBeVisible();
  const workItems = sidebarDetail(page, "Work items");
  await expect(workItems).toHaveText("1/3");
  await page.getByRole("tab", { name: "States", exact: true }).click();
  const frozenShares = ["Backlog 0% of 3", "Unstarted 33% of 3", "Started 33% of 3", "Completed 33% of 3"];
  await expect.poll(() => rowsOf(page.getByRole("tabpanel"))).toEqual(frozenShares);

  // mem completes Fix the login and takes Plan out of Sprint: its live progress is 2 of 2 done, its frozen column the
  // same after another run of the freeze, and nerve answers both.
  expect(
    [
      (await updateIssue(api, mem.token, fix.id, { state_id: states.done })).state_id,
      (await updateIssue(api, mem.token, plan.id, { cycle_id: null })).cycle_id,
    ],
    "mem completes Fix the login and takes Plan out"
  ).toEqual([states.done, null]);
  await freezeRanSince(db, await databaseNow(db));
  const live: Progress = {
    ...counts({ completed: 2, cancelled: 1 }),
    assignees: byAssignee([mem.id, 2, 0]),
    labels: byLabel([bug.id, 1, 0], [null, 1, 0]),
    burndown: burndown(day(-10), day(-4), 2, 2, 2, 2, 2, 2, 2, 2),
  };
  await expectCountedInDatabase(db, { cycle: sprint.id }, live);
  expect(await snapshotOf(db, sprint.id), "Sprint's frozen column after the changes").toEqual(column);
  expect(await cycleProgress(api, admin.token, sprint.id), "Sprint's progress after the changes").toEqual({
    status: "completed",
    live,
    progress_snapshot: column,
  });

  // Reloaded, Sprint's sidebar shows the frozen numbers still, not the live 2/2; and so does its row in the list, at
  // 33%, not 100%, though its transfer has gone with the live unfinished ones.
  await page.reload();
  await expect(page.getByRole("heading", { name: "Sprint", exact: true })).toBeVisible();
  await expect(workItems).toHaveText("1/3");
  await expect.poll(() => rowsOf(page.getByRole("tabpanel"))).toEqual(frozenShares);
  await openCyclesList(page, slug, web.id);
  await sectionToggle(page, "Completed cycle").click();
  await expect(ring).toHaveAttribute("aria-valuenow", "33");
  await expect(transferEntry(row)).toHaveCount(0);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});
