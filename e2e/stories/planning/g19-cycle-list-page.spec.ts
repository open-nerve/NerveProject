import type { Page, Response } from "@playwright/test";

import { addProjectMembers, createProject, projectMembershipOf, projectMemberWrites } from "../../fixtures/api";
import { anotherTabSignsIn, shownNameOf } from "../../fixtures/auth";
import { watchPage } from "../../fixtures/browser";
import { cyclePath, cycleSection, cyclesPath, tabKey, tabStops, toast, turnCycles } from "../../fixtures/cycle-pages";
import { changeCyclePreferences, createCycle, cyclePreferencesRead } from "../../fixtures/cycles";
import { modalDialog, shownWithin } from "../../fixtures/dialogs";
import { createIssue, updateIssue, type Issue } from "../../fixtures/issues";
import { expectListPageQuiet } from "../../fixtures/issue-list-page";
import {
  chooseFromRow,
  cycleListWorld,
  headerCount,
  listQueries,
  openListPage,
  quickAddButton,
  quietEachTime,
  rowEntries,
  rowOptions,
  shownItems,
} from "../../fixtures/planning-list-pages";
import { answerTo, moveWithinApp, sentHeld } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// G19, a cycle's page's work items (M6 P5b spec 2.1, 2.7, 2.8, 2.10; M4 handoff M6-02; v0 design 7.7), as mem and
// Web's guests use them in a browser: a list read belongs to its session and its page, so nerve's late answer never
// comes to the page of another cycle, another project, a session after it, or a caller who left the project meanwhile;
// a guest whom Web shows his own work items sees those alone; the card of the cycle in progress lists its urgent and
// high work items by its own read and takes the cycle's page to a row's work items; and the header's controls are
// reached by the keyboard. The world is made through the API, nerve's answers read back through it.

/** The address of the reads of the work items of the project of projectId, a cycle's page's with its cycle. */
const listed = (projectId: string) => `/api/v0/projects/${projectId}/issues`;

/** Opens the page of the cycle named to by the switcher of the header of the page of the cycle named from. */
async function switchCycle(page: Page, from: string, to: string): Promise<void> {
  await page.getByRole("button", { name: from, exact: true }).click();
  await page.getByRole("option").filter({ hasText: to }).click();
}

/** Lets a held answer of a read of a list through; the work item of name, of that answer alone, does not show. */
async function lateAnswerAway(page: Page, late: { release: () => Promise<Response> }, name: string): Promise<void> {
  expect((await late.release()).status()).toBe(200);
  expect(await shownWithin(rowOptions(page, name)), `${name}, of the late answer`).toBe(false);
}

/** The calendar day, YYYY-MM-DD in UTC, days after today: a cycle from three days ago to three ahead is in progress. */
function dayFromToday(days: number): string {
  return new Date(Date.now() + days * 86_400_000).toISOString().slice(0, 10);
}

test("G19 (page): on Next's page the header's layouts, filters, display and new work item are reached by Tab, each by its name; Now's list read is held as the switcher opens Now and Next again, as Back and Forward come back to Next, and as Ops's Launch opens: each page lists its own work items, and Now's late answer comes to none", async ({
  api,
  browserName,
  signedInPage,
}, testInfo) => {
  const w = await cycleListWorld(api, testInfo);
  const [now, next] = [await w.cycle("Now"), await w.cycle("Next")];
  await w.item("Plan", { cycle_id: now.id });
  await w.item("Build", { cycle_id: now.id });
  await w.item("Ship", { cycle_id: next.id });
  // Ops, Acme's other project, mem's too: its cycle Launch has Deploy
  const ops = await createProject(api, w.token, w.slug, { name: "Ops", identifier: "OPS" });
  await addProjectMembers(api, w.token, ops.id, [{ member_id: w.mem.id, role: 15 }]);
  await turnCycles(api, w.token, ops, true);
  const launch = await createCycle(api, w.token, ops.id, { name: "Launch" });
  await createIssue(api, w.token, ops.id, { name: "Deploy", cycle_id: launch.id });

  const { page, watch } = await openListPage(signedInPage, w.mem, w.path(next.id));
  const quiet = quietEachTime(page, watch);
  await expect.poll(() => shownItems(page)).toEqual(["Ship"]);

  // From the header's switcher, Tab reaches the layouts, the filters, the display and the new work item, in order.
  await page.getByRole("button", { name: "Next", exact: true }).focus();
  expect(await tabStops(page, page.locator("body"), tabKey(browserName), 7)).toEqual([
    "List Layout",
    "Board Layout",
    "Calendar Layout",
    "Table Layout",
    "Add filter",
    "Display",
    "Add work item",
  ]);

  // The switcher opens Now, whose list's answer nerve gives and the page waits for, then Next again: Next's page lists
  // Ship alone, and counts it; Now's answer comes to it late, and is not shown.
  const switched = await sentHeld(page, "GET", listed(w.web.id), () => switchCycle(page, "Next", "Now"));
  await expect(page).toHaveURL(w.path(now.id));
  await switchCycle(page, "Now", "Next");
  await expect(page).toHaveURL(w.path(next.id));
  await expect.poll(() => shownItems(page)).toEqual(["Ship"]);
  await lateAnswerAway(page, switched, "Build");
  await expect(headerCount(page, "Next")).toHaveText("1");

  // Back to Now, counted 2 as nerve answered, its list read again, held; Forward to Next, Ship alone.
  const back = await sentHeld(page, "GET", listed(w.web.id), async () => {
    await page.goBack();
  });
  await expect(page).toHaveURL(w.path(now.id));
  await expect(headerCount(page, "Now")).toHaveText("2");
  await page.goForward();
  await expect(page).toHaveURL(w.path(next.id));
  await expect.poll(() => shownItems(page)).toEqual(["Ship"]);
  await lateAnswerAway(page, back, "Build");
  expect(await shownItems(page), "Next's work items, after Now's late answer").toEqual(["Ship"]);
  await expect(headerCount(page, "Next")).toHaveText("1");

  // Back to Now again, its read held; Ops's Launch, of another project: Deploy alone.
  const away = await sentHeld(page, "GET", listed(w.web.id), async () => {
    await page.goBack();
  });
  await expect(page).toHaveURL(w.path(now.id));
  await moveWithinApp(page, cyclePath(w.slug, ops.id, launch.id));
  await expect.poll(() => shownItems(page)).toEqual(["Deploy"]);
  await lateAnswerAway(page, away, "Build");
  await expect(headerCount(page, "Launch")).toHaveText("1");
  await quiet();
});

test("G19 (page): Now's page reads its list again and nerve's answer is held; the admin removes mem from Web, takes Build out of Now and makes Fresh in it; mem's search reads Web's projects and the page says he has no access; he joins Web again and Now's page lists Fresh and Plan, as nerve has them now; the late answer of before comes to nothing, and nothing fails", async ({
  api,
  signedInPage,
}, testInfo) => {
  const w = await cycleListWorld(api, testInfo);
  const now = await w.cycle("Now");
  await w.item("Plan", { cycle_id: now.id });
  const build = await w.item("Build", { cycle_id: now.id });
  // mem's page's clock, flowing as the real one, lets the story pass SWR's 2-second deduplication window at once
  const page = await signedInPage(w.mem.tokens);
  await page.clock.install();
  const watch = await watchPage(page);
  await page.goto(w.path(now.id));
  await expect.poll(() => shownItems(page)).toEqual(["Build", "Plan"]);

  // Web's cycles, then Back to Now, counted 2 as nerve answered, its list read again, nerve's answer held.
  await moveWithinApp(page, cyclesPath(w.slug, w.web.id));
  await expect(cycleSection(page, "Upcoming cycle")).toBeVisible();
  const late = await sentHeld(page, "GET", listed(w.web.id), async () => {
    await page.goBack();
  });
  await expect(page).toHaveURL(w.path(now.id));
  await expect(headerCount(page, "Now")).toHaveText("2");
  const membership = await projectMembershipOf(api, w.token, w.web.id, w.mem.id);
  expect(await projectMemberWrites(api, w.web.id).remove(w.token, membership), "mem removed from Web").toEqual({
    status: 204,
  });
  await updateIssue(api, w.token, build.id, { cycle_id: null });
  await w.item("Fresh", { cycle_id: now.id });

  // mem's search reads Web's projects: Web, public, is no longer his, and its page says so.
  await page.clock.fastForward(3_000);
  const catalog = await answerTo(page, "GET", `/api/v0/workspaces/${w.slug}/projects`, () =>
    page.keyboard.press("ControlOrMeta+k")
  );
  expect(catalog.status()).toBe(200);
  await expect(modalDialog(page).getByRole("combobox")).toBeFocused();
  await page.keyboard.press("Escape");
  await expect(page.getByText("Seems like you don’t have access to this Project", { exact: true })).toBeVisible();

  // He joins Web again, as a member as before: Now's page reads its list anew, Fresh and Plan, as nerve has it now;
  // the answer of his read before, let through, does not come back to it.
  await page.getByRole("button", { name: "Join project", exact: true }).click();
  await expect.poll(() => shownItems(page)).toEqual(["Fresh", "Plan"]);
  await lateAnswerAway(page, late, "Build");
  expect(await shownItems(page)).toEqual(["Fresh", "Plan"]);
  await expect(headerCount(page, "Now")).toHaveText("2");
  await expectListPageQuiet(page, watch);
});

test("G19 (page): Now's list read is held as another tab signs wanda in, Acme's admin whom Web shows her own work items alone: the page follows her and lists her own, counted 1, and mem's late answer comes to nothing; as Acme's admin she may make a work item and takes hers out of Now; gus, Web's guest, signed in next, sees his own alone, counted 1, with neither a new work item nor a removal", async ({
  api,
  context,
  signedInPage,
}, testInfo) => {
  const w = await cycleListWorld(api, testInfo);
  const now = await w.cycle("Now");
  await w.item("Plan", { cycle_id: now.id });
  await w.item("Build", { cycle_id: now.id });
  const gus = await w.guest("gus", "Gus's own", now.id);
  const wanda = await w.guest("wanda", "Wanda's own", now.id, true);

  const { page, watch } = await openListPage(signedInPage, w.mem, w.path(now.id));
  const quiet = quietEachTime(page, watch);
  await expect.poll(() => shownItems(page)).toEqual(["Build", "Gus's own", "Plan", "Wanda's own"]);
  await expect(headerCount(page, "Now")).toHaveText("4");

  // Now's page loaded again, its list's answer held; another tab signs wanda in, and the page follows her session.
  const late = await sentHeld(page, "GET", listed(w.web.id), async () => {
    await page.reload();
  });
  await (await anotherTabSignsIn(context, api, wanda.email)).close();
  await expect.poll(() => shownItems(page)).toEqual(["Wanda's own"]);
  await expect(headerCount(page, "Now")).toHaveText("1");
  await lateAnswerAway(page, late, "Build");
  expect(await shownItems(page)).toEqual(["Wanda's own"]);
  // Acme's admin, she may make a work item of Now and take hers out of it, as on Web's page (spec 2.6): nerve takes
  // both from the workspace's admin (issue.create, issue.update), whose project role is a guest's.
  await expect(page.getByRole("button", { name: "Add work item", exact: true })).toBeVisible();
  await expect(quickAddButton(page)).toBeVisible();
  expect(await rowEntries(page, "Wanda's own")).toEqual([
    "Edit",
    "Make a copy",
    "Open in new tab",
    "Copy link",
    "Remove from cycle",
    "Delete",
  ]);
  await chooseFromRow(page, "Wanda's own", "Remove from cycle");
  await expect(toast(page, "Work item removed from the cycle successfully.")).toBeVisible();
  await expect.poll(() => shownItems(page)).toEqual([]);

  // gus, signed in next, sees his own alone: no new work item, in the header or the list, and a row he only opens.
  await (await anotherTabSignsIn(context, api, gus.email)).close();
  await expect.poll(() => shownItems(page)).toEqual(["Gus's own"]);
  await expect(headerCount(page, "Now")).toHaveText("1");
  await expect(page.getByRole("button", { name: "Add work item", exact: true })).toHaveCount(0);
  await expect(quickAddButton(page)).toHaveCount(0);
  expect(await rowEntries(page, "Gus's own")).toEqual(["Open in new tab", "Copy link"]);
  await quiet();
});

test("G19 (page): the card of Now, in progress, lists its 31 urgent and high work items by their own read, the newest first, 30 then the last by its cursor; Hot 31's row opens Now's page of the urgent and high ones, mem's own filters replaced, previewing Hot 31; there the sidebar's row of mem adds him to the filters and takes him out again; back on the card, mem's row of the assignees, once nerve keeps its change, opens Now's page of his work items, kept after a reload", async ({
  api,
  signedInPage,
}, testInfo) => {
  const w = await cycleListWorld(api, testInfo);
  const now = await w.cycle("Now", { start_date: dayFromToday(-3), end_date: dayFromToday(3) });
  // Hot 01 to Hot 30, high; Hot 31, urgent, mem's; Calm, low, the admin's; made in that order
  const made = await Array.from({ length: 31 }, (_, n) => `Hot ${String(n + 1).padStart(2, "0")}`).reduce(
    async (before, name, n) => [
      ...(await before),
      await w.item(name, {
        cycle_id: now.id,
        priority: n === 30 ? "urgent" : "high",
        assignee_ids: n === 30 ? [w.mem.id] : [],
      }),
    ],
    Promise.resolve([] as Issue[])
  );
  await w.item("Calm", { cycle_id: now.id, priority: "low", assignee_ids: [w.admin.id] });
  const token = w.mem.tokens.access_token;
  // mem's own filters of Now's page, which the card's read does not take, and its row replaces
  await changeCyclePreferences(api, token, now.id, { filters: { priority: ["low"], assignee_id: [w.admin.id] } });
  const filters = async () => {
    const read = await cyclePreferencesRead(api, token, now.id);
    return "preferences" in read ? read.preferences.filters : read;
  };
  const preferences = `/api/v0/me/cycles/${now.id}/preferences`;

  const page = await signedInPage(w.mem.tokens);
  const watch = await watchPage(page);
  const queries = listQueries(page, w.web.id);
  const list = cyclesPath(w.slug, w.web.id);
  await page.goto(list);
  const card = cycleSection(page, "Active cycle");
  const tab = (name: string) => card.getByRole("tab", { name, exact: true });

  // The tab of the priority work items: its own query, 30 rows, then the last from its cursor, each once.
  await tab("Priority work items").click();
  const rows = card.getByRole("tabpanel").getByRole("listitem");
  await expect(rows).toHaveCount(30);
  await rows.last().scrollIntoViewIfNeeded();
  await expect(rows).toHaveCount(31);
  const names = await rows.evaluateAll((items) => items.map((item) => /Hot \d\d/.exec(item.textContent ?? "")?.[0]));
  expect(names).toEqual(made.map((issue) => issue.name).toReversed());
  const first = { cycle_id: now.id, priority: "urgent,high", order_by: "-created_at", sub_issue: "true", limit: "30" };
  expect(queries.map((query) => Object.fromEntries(query))).toEqual([first, { ...first, cursor: expect.any(String) }]);

  // Hot 31's row: Now's page of the urgent and high work items, mem's own filters replaced, Hot 31 previewed there.
  await rows.filter({ hasText: "Hot 31" }).click();
  await expect(page).toHaveURL(w.path(now.id));
  await expect(headerCount(page, "Now")).toHaveText("31");
  const peek = page.locator("#full-screen-portal");
  await expect(peek.getByRole("textbox", { name: "Work item title", exact: true })).toHaveValue("Hot 31");
  expect(await filters()).toEqual({ priority: ["urgent", "high"] });
  await peek.getByRole("button", { name: "Close the peek view", exact: true }).click();

  // The sidebar's row of mem adds him to the page's filters, said pressed, and takes him out again.
  const details = page.getByRole("button", { name: "Toggle details", exact: true });
  if ((await details.getAttribute("aria-pressed")) !== "true") await details.click();
  const his = page.locator("button[aria-pressed]").filter({ hasText: shownNameOf(w.mem.email) });
  await expect(his).toHaveAttribute("aria-pressed", "false");
  await his.click();
  await expect(his).toHaveAttribute("aria-pressed", "true");
  await expect(headerCount(page, "Now")).toHaveText("1");
  await expect.poll(filters).toEqual({ priority: ["urgent", "high"], assignee_id: [w.mem.id] });
  await his.click();
  await expect(his).toHaveAttribute("aria-pressed", "false");
  await expect(headerCount(page, "Now")).toHaveText("31");
  await expect.poll(filters).toEqual({ priority: ["urgent", "high"] });

  // Back on the card, mem's row of the assignees: its one change replaces the page's filters by him, and while nerve's
  // answer is held the card stays; once nerve answers, Now's page lists his work item alone, still after a reload.
  await page.goBack();
  await expect(page).toHaveURL(list);
  await tab("Assignees").click();
  const mine = card
    .getByRole("tabpanel")
    .getByRole("button")
    .filter({ hasText: shownNameOf(w.mem.email) });
  const assigned = await sentHeld(page, "PATCH", preferences, () => mine.click());
  expect(assigned.body).toEqual({ filters: { assignee_id: [w.mem.id] } });
  expect(await shownWithin(details), "Now's page").toBe(false);
  await expect(page).toHaveURL(list);
  expect((await assigned.release()).status()).toBe(200);
  await expect(page).toHaveURL(w.path(now.id));
  await expect.poll(() => shownItems(page)).toEqual(["Hot 31"]);
  await page.reload();
  await expect.poll(() => shownItems(page)).toEqual(["Hot 31"]);
  await expect(headerCount(page, "Now")).toHaveText("1");
  expect(await filters()).toEqual({ assignee_id: [w.mem.id] });
  expect(watch.apiRequests.filter((request) => request === `PATCH ${preferences}`)).toHaveLength(4);
  await expectListPageQuiet(page, watch);
});
