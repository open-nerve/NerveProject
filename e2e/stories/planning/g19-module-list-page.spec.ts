import type { Page, Response } from "@playwright/test";

import { addProjectMembers, createProject, projectMembershipOf, projectMemberWrites } from "../../fixtures/api";
import { expectCountedInDatabase } from "../../fixtures/assert/progress";
import { anotherTabSignsIn, shownNameOf } from "../../fixtures/auth";
import { watchPage } from "../../fixtures/browser";
import { tabKey, tabStops, toast } from "../../fixtures/cycle-pages";
import { modalDialog, shownWithin } from "../../fixtures/dialogs";
import { createIssue, listIssues, updateIssue } from "../../fixtures/issues";
import { expectListPageQuiet } from "../../fixtures/issue-list-page";
import {
  listedOn,
  modulePath,
  modulesPath,
  moduleRow,
  openModulesList,
  turnModules,
} from "../../fixtures/module-pages";
import { createModule, modulePreferencesRead } from "../../fixtures/modules";
import {
  chooseFromRow,
  headerCount,
  moduleListWorld,
  openListPage,
  quickAddButton,
  quietEachTime,
  rowEntries,
  rowOptions,
  shownItems,
} from "../../fixtures/planning-list-pages";
import { memberships } from "../../fixtures/planning-lists";
import { moduleProgress } from "../../fixtures/planning-progress";
import { answerTo, moveWithinApp, sentHeld } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// G19, a module's page's work items (M6 P6b spec 2.3, 2.6–2.8, 2.10, R3, R14, R15; M4 handoff M6-02; v0 design 7.7),
// as mem and Web's guests use them in a browser: a list read belongs to its session and its page, so nerve's late
// answer never comes to the page of another module, another project, a session after it, or a caller who left the
// project meanwhile; a guest whom Web shows his own work items sees those alone, counted as nerve and the database
// count them; the header's controls are reached by the keyboard, its layouts, filters and display once nerve has
// answered the page's preferences; and the sidebar's rows toggle the page's filters, while the modules list's preview
// of the module offers none. The world is made through the API, nerve's answers read back through it and the database.

/** The address of the reads of the work items of the project of projectId, a module's page's with its module. */
const listed = (projectId: string) => `/api/v0/projects/${projectId}/issues`;

/** Opens the page of the module named to by the crumb of the page of the module named from, which switches it. */
async function switchModule(page: Page, from: string, to: string): Promise<void> {
  await page.getByRole("button", { name: from, exact: true }).click();
  await page.getByRole("option").filter({ hasText: to }).click();
}

/** Lets a held answer of a read of a list through; the work item of name, of that answer alone, does not show. */
async function lateAnswerAway(page: Page, late: { release: () => Promise<Response> }, name: string): Promise<void> {
  expect((await late.release()).status()).toBe(200);
  expect(await shownWithin(rowOptions(page, name)), `${name}, of the late answer`).toBe(false);
}

/** Goes to the modules list of Web, which page shows, within the app, and waits for its own read of them. */
async function openWebModules(page: Page, slug: string, webId: string, moduleIds: string[]): Promise<void> {
  await openModulesList(page, slug, webId);
  await expect(page.getByRole("button", { name: "Add Module", exact: true })).toBeVisible();
  await expect.poll(listedOn(page)).toEqual(moduleIds);
}

test("G19 (page): on the module Next's page the header's layouts, filters, display and new work item are reached by Tab, each by its name; the crumb opens Now, whose header offers its layouts, filters and display only once nerve has answered its preferences; Now's list read is held as the crumb opens Next again, as Back and Forward come back to Next, and as Ops's Launch opens: each page lists its own work items, and Now's late answer comes to none", async ({
  api,
  browserName,
  signedInPage,
}, testInfo) => {
  const w = await moduleListWorld(api, testInfo);
  const [now, next] = [await w.module("Now"), await w.module("Next")];
  await w.item("Plan", { module_ids: [now.id] });
  await w.item("Build", { module_ids: [now.id] });
  await w.item("Ship", { module_ids: [next.id] });
  // Ops, Acme's other project, mem's too: its module Launch has Deploy
  const ops = await createProject(api, w.token, w.slug, { name: "Ops", identifier: "OPS" });
  await addProjectMembers(api, w.token, ops.id, [{ member_id: w.mem.id, role: 15 }]);
  await turnModules(api, w.token, ops, true);
  const launch = await createModule(api, w.token, ops.id, { name: "Launch" });
  await createIssue(api, w.token, ops.id, { name: "Deploy", module_ids: [launch.id] });

  const { page, watch } = await openListPage(signedInPage, w.mem, w.path(next.id));
  const quiet = quietEachTime(page, watch);
  await expect.poll(() => shownItems(page)).toEqual(["Ship"]);

  // From the header's crumb of Next, Tab reaches the layouts, the filters, the display and the new work item, in order.
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

  // The crumb opens Now, nerve's answer to its preferences held: its header offers its new work item, and neither its
  // layouts, its filters nor its display, which come with nerve's answer; Now's list read is then sent, its answer held.
  const controls = page.getByRole("button", {
    name: /^(List Layout|Board Layout|Calendar Layout|Table Layout|Add filter|Display)$/,
  });
  await expect(controls).toHaveCount(6);
  const preferences = await sentHeld(page, "GET", `/api/v0/me/modules/${now.id}/preferences`, () =>
    switchModule(page, "Next", "Now")
  );
  await expect(page).toHaveURL(w.path(now.id));
  await expect(page.getByRole("button", { name: "Now", exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Add work item", exact: true })).toBeVisible();
  await expect(controls).toHaveCount(0);
  const switched = await sentHeld(page, "GET", listed(w.web.id), async () => {
    expect((await preferences.release()).status()).toBe(200);
  });
  await expect(controls).toHaveCount(6);

  // The crumb opens Next again: Next's page lists Ship alone, and counts it; Now's answer comes to it late, and is not
  // shown.
  await switchModule(page, "Now", "Next");
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
  await moveWithinApp(page, modulePath(w.slug, ops.id, launch.id));
  await expect.poll(() => shownItems(page)).toEqual(["Deploy"]);
  await lateAnswerAway(page, away, "Build");
  await expect(headerCount(page, "Launch")).toHaveText("1");
  await quiet();
});

test("G19 (page): the module Now's page reads its list again and nerve's answer is held; the admin removes mem from Web, takes Build out of Now and makes Fresh in it; mem's search reads Web's projects and the page says he has no access; he joins Web again and Now's page lists Fresh and Plan, as nerve has them now; the late answer of before comes to nothing, and nothing fails", async ({
  api,
  signedInPage,
}, testInfo) => {
  const w = await moduleListWorld(api, testInfo);
  const now = await w.module("Now");
  await w.item("Plan", { module_ids: [now.id] });
  const build = await w.item("Build", { module_ids: [now.id] });
  // mem's page's clock, flowing as the real one, lets the story pass SWR's 2-second deduplication window at once
  const page = await signedInPage(w.mem.tokens);
  await page.clock.install();
  const watch = await watchPage(page);
  await page.goto(w.path(now.id));
  await expect.poll(() => shownItems(page)).toEqual(["Build", "Plan"]);

  // Web's modules, then Back to Now, counted 2 as nerve answered, its list read again, nerve's answer held.
  await openWebModules(page, w.slug, w.web.id, [now.id]);
  const late = await sentHeld(page, "GET", listed(w.web.id), async () => {
    await page.goBack();
  });
  await expect(page).toHaveURL(w.path(now.id));
  await expect(headerCount(page, "Now")).toHaveText("2");
  const membership = await projectMembershipOf(api, w.token, w.web.id, w.mem.id);
  expect(await projectMemberWrites(api, w.web.id).remove(w.token, membership), "mem removed from Web").toEqual({
    status: 204,
  });
  await updateIssue(api, w.token, build.id, { module_ids: [] });
  await w.item("Fresh", { module_ids: [now.id] });

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

test("G19 (page): the module Now's list read is held as another tab signs wanda in, Acme's admin whom Web shows her own work items alone: the page follows her and lists her own, counted 1 as nerve and the database count it, and mem's late answer comes to nothing; as Acme's admin she may make a work item and takes hers out of Now, its row of Now alone removed; gus, Web's guest, signed in next, sees his own alone, counted 1 as nerve and the database count it, with neither a new work item nor a removal", async ({
  api,
  context,
  db,
  signedInPage,
}, testInfo) => {
  const w = await moduleListWorld(api, testInfo);
  const now = await w.module("Now");
  await w.item("Plan", { module_ids: [now.id] });
  await w.item("Build", { module_ids: [now.id] });
  const gus = await w.guest("gus", "Gus's own", now.id);
  const wanda = await w.guest("wanda", "Wanda's own", now.id, true);
  // Now's work items as nerve counts them to who, whom Web shows his own alone: its progress, which the database counts
  // of his own as the API story G12 does, and its list's total
  const counted = async (who: typeof gus) => {
    const token = who.tokens.access_token;
    const progress = await moduleProgress(api, token, now.id);
    await expectCountedInDatabase(db, { module: now.id }, progress, who.id);
    return [progress.total_issues, (await listIssues(api, token, w.web.id, { module_id: [now.id], limit: 1 })).total];
  };

  const { page, watch } = await openListPage(signedInPage, w.mem, w.path(now.id));
  const quiet = quietEachTime(page, watch);
  await expect.poll(() => shownItems(page)).toEqual(["Build", "Gus's own", "Plan", "Wanda's own"]);
  await expect(headerCount(page, "Now")).toHaveText("4");

  // Web's modules, then Back to Now, its list read again and nerve's answer held; another tab signs wanda in, and the
  // page follows her session.
  await openWebModules(page, w.slug, w.web.id, [now.id]);
  const late = await sentHeld(page, "GET", listed(w.web.id), async () => {
    await page.goBack();
  });
  await expect(page).toHaveURL(w.path(now.id));
  await (await anotherTabSignsIn(context, api, wanda.email)).close();
  await expect.poll(() => shownItems(page)).toEqual(["Wanda's own"]);
  await expect(headerCount(page, "Now")).toHaveText("1");
  expect(await counted(wanda), "Now's work items counted to wanda, by its progress and its list").toEqual([1, 1]);
  await lateAnswerAway(page, late, "Build");
  expect(await shownItems(page)).toEqual(["Wanda's own"]);
  // Acme's admin, she may make a work item of Now and take hers out of it, as on Web's page (P6b spec 2.7, R15): nerve
  // takes both from the workspace's admin (issue.create, issue.update), whose project role is a guest's.
  await expect(page.getByRole("button", { name: "Add work item", exact: true })).toBeVisible();
  await expect(quickAddButton(page)).toBeVisible();
  expect(await rowEntries(page, "Wanda's own")).toEqual([
    "Edit",
    "Make a copy",
    "Open in new tab",
    "Copy link",
    "Remove from module",
    "Delete",
  ]);
  await chooseFromRow(page, "Wanda's own", "Remove from module");
  await expect(toast(page, "Work item removed from the module successfully.")).toBeVisible();
  await expect.poll(() => shownItems(page)).toEqual([]);
  expect(await memberships(db, w.web.id), "Web's rows of modules: Wanda's own's of Now alone removed").toEqual([
    { kind: "module", target: "Now", archived: false, item: "Build", removed: false },
    { kind: "module", target: "Now", archived: false, item: "Gus's own", removed: false },
    { kind: "module", target: "Now", archived: false, item: "Plan", removed: false },
    { kind: "module", target: "Now", archived: false, item: "Wanda's own", removed: true },
  ]);

  // gus, signed in next, sees his own alone: no new work item, in the header or the list, and a row he only opens.
  await (await anotherTabSignsIn(context, api, gus.email)).close();
  await expect.poll(() => shownItems(page)).toEqual(["Gus's own"]);
  await expect(headerCount(page, "Now")).toHaveText("1");
  expect(await counted(gus), "Now's work items counted to gus, by its progress and its list").toEqual([1, 1]);
  await expect(page.getByRole("button", { name: "Add work item", exact: true })).toHaveCount(0);
  await expect(quickAddButton(page)).toHaveCount(0);
  expect(await rowEntries(page, "Gus's own")).toEqual(["Open in new tab", "Copy link"]);
  await quiet();
});

test("G19 (page): on the module Now's page the sidebar's row of mem adds him to the page's filters, said pressed and kept as the page's own preferences, and takes him out again; on Web's modules list, which the page is left for, the preview of Now shows the same row, which chooses nothing", async ({
  api,
  signedInPage,
}, testInfo) => {
  const w = await moduleListWorld(api, testInfo);
  const now = await w.module("Now");
  await w.item("Plan", { module_ids: [now.id], assignee_ids: [w.mem.id] });
  await w.item("Build", { module_ids: [now.id], assignee_ids: [w.admin.id] });
  // mem's own filters of Now's page, as nerve keeps them
  const filters = async () => {
    const read = await modulePreferencesRead(api, w.mem.tokens.access_token, now.id);
    return "preferences" in read ? read.preferences.filters : read;
  };

  // Now's page, its progress sidebar open as it is at first.
  const page = await signedInPage(w.mem.tokens);
  const watch = await watchPage(page);
  await page.goto(w.path(now.id));
  await expect.poll(() => shownItems(page)).toEqual(["Build", "Plan"]);
  await expect(page.getByRole("button", { name: "Toggle details", exact: true })).toHaveAttribute(
    "aria-pressed",
    "true"
  );

  // The sidebar's row of mem, of Now's assignees, adds him to the page's filters, said pressed, and takes him out
  // again; nerve keeps each change as the page's own.
  const name = shownNameOf(w.mem.email);
  const his = page.locator("button[aria-pressed]").filter({ hasText: name });
  await expect(his).toHaveAttribute("aria-pressed", "false");
  await his.click();
  await expect(his).toHaveAttribute("aria-pressed", "true");
  await expect.poll(() => shownItems(page)).toEqual(["Plan"]);
  await expect(headerCount(page, "Now")).toHaveText("1");
  await expect.poll(filters).toEqual({ assignee_id: [w.mem.id] });
  await his.click();
  await expect(his).toHaveAttribute("aria-pressed", "false");
  await expect.poll(() => shownItems(page)).toEqual(["Build", "Plan"]);
  await expect(headerCount(page, "Now")).toHaveText("2");
  await expect.poll(filters).toEqual({});

  // Web's modules list, Now's row's "More details": its preview shows mem's row of Now's assignees, which is plain.
  await openWebModules(page, w.slug, w.web.id, [now.id]);
  await moduleRow(page, now.id, "Now").hover();
  await page.getByRole("button", { name: "More details", exact: true }).click();
  await expect(page).toHaveURL(`${modulesPath(w.slug, w.web.id)}?peekModule=${now.id}`);
  await expect(page.getByRole("tabpanel").locator("li").filter({ hasText: name })).toBeVisible();
  await expect(his).toHaveCount(0);
  await expectListPageQuiet(page, watch);
});
