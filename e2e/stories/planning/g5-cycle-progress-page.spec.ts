import { addProjectMembers, inviteAndAccept, projectMembershipOf, projectMemberWrites } from "../../fixtures/api";
import { expectCountedInDatabase } from "../../fixtures/assert/progress";
import { named, newAccount } from "../../fixtures/auth";
import { expectQuietConsole, watchPage } from "../../fixtures/browser";
import {
  cyclePath,
  cycleRow,
  cycleSection,
  cyclesPath,
  openCyclesList,
  sectionToggle,
  seenBy,
} from "../../fixtures/cycle-pages";
import {
  avatarNames,
  burndownDots,
  onboard,
  pageSignIn,
  rowsOf,
  sidebarDetail,
} from "../../fixtures/cycle-progress-pages";
import { createCycle } from "../../fixtures/cycles";
import { createIssue, type IssueCreate } from "../../fixtures/issues";
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

// G5's page version (M6 P5a spec 2.3–2.5, 2.9, R7, R10): a cycle's progress as the card of the cycle in progress and
// a cycle's progress sidebar show it, by its member in his browser and by one whom nerve shows only his own work items.
// The API version is g5-cycle-progress.spec.ts; both assert the same counts in the database. Web keeps the time of a
// zone where it is about noon as the story starts (noonZone), so today's completions fall on Web's today.

test("G5 (page): Sprint, from two days ago to two days ahead, has six work items of every group, one cancelled; the card of the cycle in progress and Sprint's sidebar count 2 of the 5 not cancelled, 40%, each group's, the cancelled one said apart, and a burndown with no point after today; by assignee ada, max, gil, who has left Web and is named a former member, and the work item without one; by label Bug, UI and none; Day, of one day, draws its one point; walt, acme's admin and Web's guest, sees his own two alone, 50%", async ({
  api,
  baseURL,
  browser,
  db,
  signedInPage,
}, testInfo) => {
  // mem's page, then walt's browser
  test.setTimeout(60_000);
  const { day, admin, mem, walt, slug, web, states, bug, ui } = await progressWorld(api, testInfo);
  // gil, Web's member, is assigned Build the page and Drop IE, then leaves Web. The admin, mem and gil have names, which
  // the pages show of Web's members.
  const gil = await newAccount(api, testInfo, "gil");
  await inviteAndAccept(api, admin.token, slug, gil, 15);
  await addProjectMembers(api, admin.token, web.id, [{ member_id: gil.id, role: 15 }]);
  await Promise.all(
    (
      [
        [admin, "ada"],
        [mem, "max"],
        [gil, "gil"],
      ] as const
    ).map(([one, name]) => named(api, one.token, { display_name: name }))
  );
  await onboard(api, walt.token);
  const sprint = await createCycle(api, admin.token, web.id, { name: "Sprint", start_date: day(-2), end_date: day(2) });
  const oneDay = await createCycle(api, admin.token, web.id, { name: "Day", start_date: day(5), end_date: day(5) });

  // In Sprint: Fix the login, to do, max's, Bug; Ship v1, done today, max's and ada's, Bug and UI; Build the page, in
  // progress, gil's, UI; Drop IE, cancelled, gil's; walt's Plan, in the backlog, and Release, done today, ada's, UI.
  // In Day, which then ends, Tidy up, to do.
  const item = (token: string, body: IssueCreate) => createIssue(api, token, web.id, { cycle_id: sprint.id, ...body });
  await item(admin.token, {
    name: "Fix the login",
    state_id: states.todo,
    assignee_ids: [mem.id],
    label_ids: [bug.id],
  });
  await item(admin.token, {
    name: "Ship v1",
    state_id: states.done,
    assignee_ids: [mem.id, admin.id],
    label_ids: [bug.id, ui.id],
  });
  await item(admin.token, {
    name: "Build the page",
    state_id: states.started,
    assignee_ids: [gil.id],
    label_ids: [ui.id],
  });
  await item(admin.token, { name: "Drop IE", state_id: states.cancelled, assignee_ids: [gil.id] });
  await item(walt.token, { name: "Plan", state_id: states.backlog });
  await item(walt.token, { name: "Release", state_id: states.done, assignee_ids: [admin.id], label_ids: [ui.id] });
  await item(admin.token, { name: "Tidy up", state_id: states.todo, cycle_id: oneDay.id });
  await endCycle(api, admin.token, oneDay.id, day(-5), day(-5));
  const membership = await projectMembershipOf(api, admin.token, web.id, gil.id);
  expect(await projectMemberWrites(api, web.id).remove(admin.token, membership), "gil removed from Web").toEqual({
    status: 204,
  });

  // Sprint's progress as nerve answers it and the database counts it: six work items, five not cancelled, two done
  // today; gil still counts by Build the page, though he has left; walt counts his own two alone.
  const everyone: Progress = {
    ...counts({ backlog: 1, unstarted: 1, started: 1, completed: 2, cancelled: 1 }),
    assignees: byAssignee([admin.id, 2, 0], [mem.id, 1, 1], [gil.id, 0, 1], [null, 0, 1]),
    labels: byLabel([bug.id, 1, 1], [ui.id, 2, 1], [null, 0, 1]),
    burndown: burndown(day(-2), day(2), 5, 5, 5, 3),
  };
  const walts: Progress = {
    ...counts({ backlog: 1, completed: 1 }),
    assignees: byAssignee([admin.id, 1, 0], [null, 0, 1]),
    labels: byLabel([ui.id, 1, 0], [null, 0, 1]),
    burndown: burndown(day(-2), day(2), 2, 2, 2, 1),
  };
  expect(
    [await cycleProgress(api, admin.token, sprint.id), await cycleProgress(api, walt.token, sprint.id)],
    "Sprint's progress to the admin and to walt"
  ).toEqual([
    { status: "current", live: everyone, progress_snapshot: null },
    { status: "current", live: walts, progress_snapshot: null },
  ]);
  await expectCountedInDatabase(db, { cycle: sprint.id }, everyone);
  await expectCountedInDatabase(db, { cycle: sprint.id }, walts, walt.id);

  // mem's list: Sprint's row and the card of the cycle in progress count 2 of the 5 not cancelled, 40%, each group's
  // work items and the cancelled one apart; its burndown has five days, the three to today drawn; three pending.
  const page = await signedInPage(await pageSignIn(api, mem.email));
  const watch = await watchPage(page);
  await page.goto(cyclesPath(slug, web.id));
  const card = cycleSection(page, "Active cycle");
  await expect(cycleRow(page, sprint.id, "Sprint").getByRole("progressbar", { name: "Progress" })).toHaveAttribute(
    "aria-valuenow",
    "40"
  );
  const progress = card.getByRole("region", { name: "Progress", exact: true });
  await expect(progress).toContainText("2/5 work items completed");
  await expect(progress.getByRole("progressbar")).toHaveAttribute("aria-valuenow", "40");
  await expect
    .poll(() => rowsOf(progress))
    .toEqual(["Completed 2 Work items", "Started 1 Work item", "Unstarted 1 Work item", "Backlog 1 Work item"]);
  const cancelled = "1 cancelled work item is excluded from this report.";
  await expect(progress.getByText(cancelled, { exact: true })).toBeVisible();
  const chart = card.getByRole("region", { name: "Work item burndown", exact: true });
  await expect(chart.getByText("Pending work items - 3", { exact: true })).toBeVisible();
  await expect.poll(() => burndownDots(chart)).toEqual({ remaining: 3, ideal: 5, finite: true });

  // Its breakdowns, by assignee first: ada's and max's by name, gil's as a former member's, the one without an
  // assignee last; then by label, the one without a label last. None counts Drop IE, cancelled.
  const byAssigneeRows = ["ada 100% of 2", "max 50% of 2", "Former member 0% of 1", "No assignee 0% of 1"];
  const byLabelRows = ["Bug 50% of 2", "UI 67% of 3", "No labels 0% of 1"];
  await expect(card.getByRole("tab", { name: "Assignees", exact: true })).toHaveAttribute("aria-selected", "true");
  await expect.poll(() => rowsOf(card.getByRole("tabpanel"))).toEqual(byAssigneeRows);
  await card.getByRole("tab", { name: "Labels", exact: true }).click();
  await expect.poll(() => rowsOf(card.getByRole("tabpanel"))).toEqual(byLabelRows);

  // Sprint's page, from the card: its sidebar counts the same, its members ada, max and gil, a former member; its
  // burndown the same five days; its breakdowns by assignee, by state of the five not cancelled, and by label.
  await card
    .locator(`a[href="${cyclePath(slug, web.id, sprint.id)}"]`)
    .first()
    .click();
  await expect(page.getByRole("heading", { name: "Sprint", exact: true })).toBeVisible();
  await expect(sidebarDetail(page, "Work items")).toHaveText("2/5");
  await expect.poll(() => avatarNames(sidebarDetail(page, "Members"))).toEqual(["Former member", "ada", "max"]);
  await expect.poll(() => burndownDots(page.locator("body"))).toEqual({ remaining: 3, ideal: 5, finite: true });
  const tab = (name: string) => page.getByRole("tab", { name, exact: true });
  await expect(tab("Assignees")).toHaveAttribute("aria-selected", "true");
  await expect.poll(() => rowsOf(page.getByRole("tabpanel"))).toEqual(byAssigneeRows);
  await tab("States").click();
  await expect
    .poll(() => rowsOf(page.getByRole("tabpanel")))
    .toEqual(["Backlog 20% of 5", "Unstarted 20% of 5", "Started 20% of 5", "Completed 40% of 5"]);
  await expect(page.getByRole("tabpanel").getByText(cancelled, { exact: true })).toBeVisible();
  await tab("Labels").click();
  await expect.poll(() => rowsOf(page.getByRole("tabpanel"))).toEqual(byLabelRows);

  // Day, of one day, ended: its sidebar's burndown draws its one day, on each line.
  await openCyclesList(page, slug, web.id);
  await sectionToggle(page, "Completed cycle").click();
  await page
    .locator(`a[href="${cyclePath(slug, web.id, oneDay.id)}"]`)
    .first()
    .click();
  await expect(page.getByRole("heading", { name: "Day", exact: true })).toBeVisible();
  await expect(sidebarDetail(page, "Work items")).toHaveText("0/1");
  await expect.poll(() => burndownDots(page.locator("body"))).toEqual({ remaining: 1, ideal: 1, finite: true });
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);

  // walt, acme's admin and Web's guest, whom nerve shows his own work items alone, Plan and Release: his card and his
  // sidebar count those, 1 of 2, 50%, his burndown drawn to today; ada's row and the one without an assignee, UI's
  // and the one without a label.
  await seenBy(browser, baseURL ?? "", api, walt.email, cyclesPath(slug, web.id), async (other) => {
    const own = cycleSection(other, "Active cycle");
    await expect(cycleRow(other, sprint.id, "Sprint").getByRole("progressbar", { name: "Progress" })).toHaveAttribute(
      "aria-valuenow",
      "50"
    );
    const ownProgress = own.getByRole("region", { name: "Progress", exact: true });
    await expect(ownProgress).toContainText("1/2 work items completed");
    await expect(ownProgress.getByRole("progressbar")).toHaveAttribute("aria-valuenow", "50");
    await expect(ownProgress.getByText(cancelled, { exact: true })).toHaveCount(0);
    const ownChart = own.getByRole("region", { name: "Work item burndown", exact: true });
    await expect(ownChart.getByText("Pending work items - 1", { exact: true })).toBeVisible();
    await expect.poll(() => burndownDots(ownChart)).toEqual({ remaining: 3, ideal: 5, finite: true });
    await expect.poll(() => rowsOf(own.getByRole("tabpanel"))).toEqual(["ada 100% of 1", "No assignee 0% of 1"]);
    await own.getByRole("tab", { name: "Labels", exact: true }).click();
    await expect.poll(() => rowsOf(own.getByRole("tabpanel"))).toEqual(["UI 100% of 1", "No labels 0% of 1"]);
    await own
      .locator(`a[href="${cyclePath(slug, web.id, sprint.id)}"]`)
      .first()
      .click();
    await expect(sidebarDetail(other, "Work items")).toHaveText("1/2");
  });
});
