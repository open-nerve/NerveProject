import type { Locator, Page } from "@playwright/test";

import { addProjectMembers, inviteAndAccept, projectMembershipOf, projectMemberWrites } from "../../fixtures/api";
import { expectCountedInDatabase } from "../../fixtures/assert/progress";
import { named, newAccount } from "../../fixtures/auth";
import { expectQuietConsole, watchPage } from "../../fixtures/browser";
import { seenBy } from "../../fixtures/cycle-pages";
import { burndownDots, onboard, pageSignIn, rowsOf, sidebarDetail } from "../../fixtures/cycle-progress-pages";
import { createIssue, type IssueCreate } from "../../fixtures/issues";
import { layoutButton, listedOn, modulePath, moduleRow, modulesPath } from "../../fixtures/module-pages";
import { createModule } from "../../fixtures/modules";
import {
  burndown,
  byAssignee,
  byLabel,
  counts,
  moduleProgress,
  progressWorld,
  type Progress,
} from "../../fixtures/planning-progress";
import { expect, test } from "../../fixtures/test";

// G12's page version (M6 P6a spec 2.4–2.6, 2.11, R8): a module's progress as a project's modules list shows it, its
// rows and its cards, and as the module's progress sidebar does, by its member in his browser and by one whom nerve
// shows only his own work items. The API version is g12-module-progress.spec.ts; both assert the same counts in the
// database. Web keeps the time of a zone where it is about noon as the story starts (noonZone), so today's completions
// fall on Web's today.

/** The progress of the module of id, named name, as page's list shows it: its row's ring, or its card's bar. */
function progressOf(page: Page, id: string, name: string): Locator {
  return moduleRow(page, id, name).getByRole("progressbar", { name: "Progress", exact: true });
}

/**
 * Opens the page of the module of id, of the project of projectId, from page's list by its link: once its sidebar,
 * headed by name, shows.
 */
async function openModule(page: Page, slug: string, projectId: string, id: string, name: string): Promise<void> {
  await page
    .locator(`a[href="${modulePath(slug, projectId, id)}"]`)
    .first()
    .click();
  await expect(page.getByRole("heading", { name, exact: true })).toBeVisible();
}

test("G12 (page): Core, from two days ago to two days ahead, has six work items of every group, one cancelled; Web's modules list counts 2 of the 5 not cancelled, 40%, by Core's row's ring, and laid out as cards by its card's count and bar, as Core's sidebar counts them; its burndown draws the three days to today, none after; its tabs, each showing its own breakdown, count by state, the cancelled one said apart, by assignee ada, max, gil, who has left Web and is named a former member, and the work item without one, and by label Bug, UI and none; Loose, without a target day, has no burndown and its breakdowns all the same; walt, acme's admin and Web's guest, sees his own two alone, 1 of 2, 50%", async ({
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
  const core = await createModule(api, admin.token, web.id, {
    name: "Core",
    start_date: day(-2),
    target_date: day(2),
  });
  const loose = await createModule(api, admin.token, web.id, { name: "Loose", start_date: day(-2) });

  // In Core: Fix the login, to do, max's, Bug, also in Loose; Ship v1, done today, max's and ada's, Bug and UI; Build
  // the page, in progress, gil's, UI; Drop IE, cancelled, gil's; walt's Plan, in the backlog, and Release, done today,
  // ada's, UI.
  const item = (token: string, body: IssueCreate) =>
    createIssue(api, token, web.id, { module_ids: [core.id], ...body });
  await item(admin.token, {
    name: "Fix the login",
    state_id: states.todo,
    assignee_ids: [mem.id],
    label_ids: [bug.id],
    module_ids: [core.id, loose.id],
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
  const membership = await projectMembershipOf(api, admin.token, web.id, gil.id);
  expect(await projectMemberWrites(api, web.id).remove(admin.token, membership), "gil removed from Web").toEqual({
    status: 204,
  });

  // Core's and Loose's progress as nerve answers it and the database counts it: Core's six work items, five not
  // cancelled, two done today; gil still counts by Build the page, though he has left; walt counts his own two alone.
  // Loose has Fix the login alone, and no target day: no burndown.
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
  const looseNumbers: Progress = {
    ...counts({ unstarted: 1 }),
    assignees: byAssignee([mem.id, 0, 1]),
    labels: byLabel([bug.id, 0, 1]),
    burndown: null,
  };
  expect(
    [
      await moduleProgress(api, admin.token, core.id),
      await moduleProgress(api, walt.token, core.id),
      await moduleProgress(api, admin.token, loose.id),
    ],
    "Core's progress to the admin and to walt, and Loose's"
  ).toEqual([everyone, walts, looseNumbers]);
  await expectCountedInDatabase(db, { module: core.id }, everyone);
  await expectCountedInDatabase(db, { module: core.id }, walts, walt.id);
  await expectCountedInDatabase(db, { module: loose.id }, looseNumbers);

  // mem's list, its modules by name: Core's ring counts 2 of the 5 not cancelled, 40%; Loose's, none of 1 done, 0%.
  const page = await signedInPage(await pageSignIn(api, mem.email));
  const watch = await watchPage(page);
  const list = modulesPath(slug, web.id);
  await page.goto(list);
  await expect.poll(listedOn(page)).toEqual([core.id, loose.id]);
  await expect(progressOf(page, core.id, "Core")).toHaveAttribute("aria-valuenow", "40");
  await expect(progressOf(page, loose.id, "Loose")).toHaveAttribute("aria-valuenow", "0");

  // Laid out as cards: Core's card says 2 of 5 work items done, its bar 40%, the same reading; Loose's 0 of 1.
  const cards = layoutButton(page, "Gallery layout");
  await cards.click();
  await expect(cards).toHaveAttribute("aria-pressed", "true");
  await expect(moduleRow(page, core.id, "Core")).toContainText("2/5 work items");
  await expect(progressOf(page, core.id, "Core")).toHaveAttribute("aria-valuenow", "40");
  await expect(moduleRow(page, loose.id, "Loose")).toContainText("0/1 work item");
  await expect(progressOf(page, loose.id, "Loose")).toHaveAttribute("aria-valuenow", "0");

  // Core's page, from its card: its sidebar counts the same, 2/5 and 40%; its burndown has five days, the three to
  // today drawn.
  await openModule(page, slug, web.id, core.id, "Core");
  await expect(sidebarDetail(page, "Work items")).toHaveText("2/5");
  await expect(page.getByRole("button", { name: "Progress 40%", exact: true })).toBeVisible();
  await expect.poll(() => burndownDots(page.locator("body"))).toEqual({ remaining: 3, ideal: 5, finite: true });

  // Its tabs, by state, assignee and label, each showing its own breakdown, by assignee first: ada's and max's by
  // name, gil's as a former member's, the one without an assignee last; by state, each group's share of the five not
  // cancelled, the cancelled one said apart; by label, the one without a label last. None counts Drop IE, cancelled.
  const tab = (name: string) => page.getByRole("tab", { name, exact: true });
  const panel = page.getByRole("tabpanel");
  await expect(page.getByRole("tab")).toHaveText(["States", "Assignees", "Labels"]);
  await expect(tab("Assignees")).toHaveAttribute("aria-selected", "true");
  await expect
    .poll(() => rowsOf(panel))
    .toEqual(["ada 100% of 2", "max 50% of 2", "Former member 0% of 1", "No assignee 0% of 1"]);
  await tab("States").click();
  await expect(tab("States")).toHaveAttribute("aria-selected", "true");
  await expect
    .poll(() => rowsOf(panel))
    .toEqual(["Backlog 20% of 5", "Unstarted 20% of 5", "Started 20% of 5", "Completed 40% of 5"]);
  await expect(panel.getByText("1 cancelled work item is excluded from this report.", { exact: true })).toBeVisible();
  await tab("Labels").click();
  await expect(tab("Labels")).toHaveAttribute("aria-selected", "true");
  await expect.poll(() => rowsOf(panel)).toEqual(["Bug 50% of 2", "UI 67% of 3", "No labels 0% of 1"]);

  // Back on the list, still of cards: Loose's page has no burndown, no data yet, and its breakdowns all the same, by
  // assignee first, its own tab.
  await page.goBack();
  await expect(page).toHaveURL(list);
  await expect(moduleRow(page, loose.id, "Loose")).toContainText("0/1 work item");
  await openModule(page, slug, web.id, loose.id, "Loose");
  await expect(sidebarDetail(page, "Work items")).toHaveText("0/1");
  await expect(page.getByRole("button", { name: "Progress", exact: true })).toBeVisible();
  await expect(page.getByText("No Data yet", { exact: true })).toBeVisible();
  await expect(page.locator("g.recharts-area")).toHaveCount(0);
  await expect(tab("Assignees")).toHaveAttribute("aria-selected", "true");
  await expect.poll(() => rowsOf(panel)).toEqual(["max 0% of 1"]);
  await tab("Labels").click();
  await expect.poll(() => rowsOf(panel)).toEqual(["Bug 0% of 1"]);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);

  // walt, acme's admin and Web's guest, whom nerve shows his own work items alone, Plan and Release: his ring, his
  // card and his sidebar count those, 1 of 2, 50%, his burndown drawn to today; ada's row and the one without an
  // assignee, UI's and the one without a label.
  await seenBy(browser, baseURL ?? "", api, walt.email, list, async (other) => {
    await expect.poll(listedOn(other)).toEqual([core.id, loose.id]);
    await expect(progressOf(other, core.id, "Core")).toHaveAttribute("aria-valuenow", "50");
    await layoutButton(other, "Gallery layout").click();
    await expect(moduleRow(other, core.id, "Core")).toContainText("1/2 work items");
    await expect(progressOf(other, core.id, "Core")).toHaveAttribute("aria-valuenow", "50");
    await openModule(other, slug, web.id, core.id, "Core");
    await expect(sidebarDetail(other, "Work items")).toHaveText("1/2");
    await expect(other.getByRole("button", { name: "Progress 50%", exact: true })).toBeVisible();
    await expect.poll(() => burndownDots(other.locator("body"))).toEqual({ remaining: 3, ideal: 5, finite: true });
    const own = other.getByRole("tabpanel");
    await expect.poll(() => rowsOf(own)).toEqual(["ada 100% of 1", "No assignee 0% of 1"]);
    await other.getByRole("tab", { name: "Labels", exact: true }).click();
    await expect.poll(() => rowsOf(own)).toEqual(["UI 100% of 1", "No labels 0% of 1"]);
  });
});
