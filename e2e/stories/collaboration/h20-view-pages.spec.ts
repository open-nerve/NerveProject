import type { Page, TestInfo } from "@playwright/test";

import { anotherTabSignsIn } from "../../fixtures/auth";
import { expectQuietConsole, watchPage } from "../../fixtures/browser";
import { deferred } from "../../fixtures/deferred";
import { itemMenu, menuEntries, recordApi, tabKey, tabStops, toast } from "../../fixtures/cycle-pages";
import { closedByEscape, focusInDialog, modalDialog, shownWithin } from "../../fixtures/dialogs";
import { expectLanded, focusedName, tabTo } from "../../fixtures/intake-focus";
import { addCondition } from "../../fixtures/issue-filters-page";
import { createIssue } from "../../fixtures/issues";
import { chooseByKeyboard } from "../../fixtures/menus";
import { rowOptions, shownItems } from "../../fixtures/planning-list-pages";
import { answerTo, holdAnswer, moveWithinApp, sentHeld, sentTo } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";
import {
  layoutButtons,
  listedViews,
  pageMain,
  rowButton,
  viewPath,
  viewRead,
  viewWorld,
  viewsPath,
  viewsRead,
} from "../../fixtures/view-pages";
import { viewCalls, viewDefaults, viewOf, viewRow } from "../../fixtures/views";

// H20's page version for a project's views pages (M7 design 2, H20; P6a spec 2.4, 2.6–2.8, 2.11; R2, R9, R11, R26,
// R28; re-96 I2, m6; v0 design 7.7): what a page sent or read belongs to its page and its session. A list of views
// answered after a move to another project's, a view's read answered after the page left it, an Update view and a
// deletion answered after their page went, and an Update view answered after another tab signed another account in,
// each write and say nothing on the page shown then; back on the view, its page reads it anew and lists what nerve
// holds now. The keyboard works the row's menu, the view's dialog (its Tab order, the icon picker's own Escape), the
// header's switcher and its layouts (the wide buttons and the narrow menu: M2 handoff 3's Dropdown half), and Update
// view, after which the focus is on the filters row's Add filter. The command palette's views are
// h20-views-palette-pages.spec.ts's.

/** A toast on page: the page said something. */
const anyToast = (page: Page) => page.getByLabel("Notifications", { exact: true }).getByRole("dialog");

/** Waits for the reads page has out to be answered (recordApi's), as a story does before it moves or changes nerve. */
async function settled(page: Page, recorded: () => { pending: number }, out = 0): Promise<void> {
  await expect.poll(() => recorded().pending, "the reads out answered").toBe(out);
}

/**
 * Opens the view named to from the header of the page of the view named from by the keyboard (its CustomSearchSelect,
 * M6-M7-04): Enter on its button, which has the focus, opens its list, its search focused; to typed, ArrowDown makes
 * its option the active one, and Enter opens it, once the reads out (the project's views, read as the list opens) are
 * answered.
 */
async function switchByKeyboard(
  page: Page,
  recorded: () => { pending: number },
  from: string,
  to: string
): Promise<void> {
  await pageMain(page).getByRole("button", { name: from, exact: true }).focus();
  await page.keyboard.press("Enter");
  const search = page.getByRole("combobox");
  await expect(search).toBeFocused();
  const option = page.getByRole("option", { name: to, exact: true });
  await expect(option).toBeVisible();
  await page.keyboard.type(to);
  await page.keyboard.press("ArrowDown");
  await expect(search).toHaveAttribute("aria-activedescendant", (await option.getAttribute("id")) ?? "");
  await settled(page, recorded);
  await page.keyboard.press("Enter");
}

/** Records where page's focus is (focusedName) as the test's, under step: each engine's, read in its report. */
async function noteFocus(page: Page, testInfo: TestInfo, step: string): Promise<void> {
  testInfo.annotations.push({ type: "focus", description: `${step}: ${await focusedName(page)}` });
}

test("H20 (page): Web's views read, answered after a move to Ops' views, lists nothing there; Hot's read, answered after Cold's switcher opened Hot by the keyboard and Back came to Cold, writes nothing into Cold's page, and Hot opened again reads it anew, as alice changed it meanwhile; Update view of Hot, answered after Back came to Cold, says nothing there, and Forward reads Hot anew too", async ({
  api,
  signedInPage,
}, testInfo) => {
  const w = await viewWorld(api, testInfo);
  await createIssue(api, w.alice.token, w.web.id, { name: "Crash", priority: "urgent" });
  await createIssue(api, w.alice.token, w.web.id, { name: "Typo", priority: "low" });
  const hot = await w.view(w.alice, { name: "Hot", filters: { priority: ["urgent"] } });
  const cold = await w.view(w.alice, { name: "Cold", filters: { priority: ["low"] } });
  const ops = await w.project("Ops", "OPS", true);
  await w.view(w.alice, { name: "Ops board" }, ops.id);
  const webViews = `/api/v0/projects/${w.web.id}/views`;
  const hotPath = `/api/v0/views/${hot.id}`;
  const issues = `/api/v0/projects/${w.web.id}/issues`;

  // Cold's page, then Ops' views: both pages' code loaded before any route is (WebKit's preloads).
  const page = await signedInPage(w.alice.tokens);
  const watch = await watchPage(page);
  const recorded = recordApi(page);
  await page.goto(viewPath(w.slug, w.web.id, cold.id));
  await expect.poll(() => shownItems(page)).toEqual(["Typo"]);
  await settled(page, recorded);
  await moveWithinApp(page, viewsPath(w.slug, ops.id));
  await expect.poll(() => listedViews(page)).toEqual(["Ops board"]);
  await settled(page, recorded);

  // Web's views, their read held, then Ops' again: let through, Ops' list as it was; Back, Web's list.
  const releaseList = await holdAnswer(page, "GET", webViews);
  await moveWithinApp(page, viewsPath(w.slug, w.web.id));
  await expect.poll(() => recorded().requests).toContain(viewsRead(w.web.id));
  await moveWithinApp(page, viewsPath(w.slug, ops.id));
  await expect.poll(() => listedViews(page)).toEqual(["Ops board"]);
  await settled(page, recorded, 1);
  expect((await answerTo(page, "GET", webViews, releaseList)).status()).toBe(200);
  expect(await shownWithin(page.getByRole("link", { name: "Hot", exact: true })), "Web's Hot, on Ops' list").toBe(
    false
  );
  expect(await listedViews(page)).toEqual(["Ops board"]);
  await page.goBack();
  await expect.poll(() => listedViews(page)).toEqual(["Cold", "Hot"]);
  await settled(page, recorded);

  // Cold's page; its switcher, by the keyboard, opens Hot, whose read nerve answers and the page waits for, alice then
  // changing Hot elsewhere; Back to Cold, then Hot's late answer: Cold's page as it was, its list, its name.
  await answerTo(page, "GET", issues, () => page.getByRole("link", { name: "Cold", exact: true }).click());
  await expect.poll(() => shownItems(page)).toEqual(["Typo"]);
  await settled(page, recorded);
  const answered = deferred();
  const releaseHot = await holdAnswer(page, "GET", hotPath, 1, () => answered.resolve());
  await switchByKeyboard(page, recorded, "Cold", "Hot");
  await expect(page).toHaveURL(viewPath(w.slug, w.web.id, hot.id));
  await answered.promise;
  viewOf(await viewCalls(api).update(w.alice.token, hot.id, { filters: {} }), 200);
  // Cold mounted again reads Cold, then its list (the list's rows of before show meanwhile): both answered
  await answerTo(page, "GET", issues, async () => {
    await page.goBack();
  });
  await expect(page).toHaveURL(viewPath(w.slug, w.web.id, cold.id));
  await expect.poll(() => shownItems(page)).toEqual(["Typo"]);
  await settled(page, recorded, 1);
  const late = recordApi(page);
  expect((await answerTo(page, "GET", hotPath, releaseHot)).status()).toBe(200);
  expect(await shownWithin(rowOptions(page, "Crash")), "Hot's Crash, on Cold's page").toBe(false);
  expect(await shownItems(page)).toEqual(["Typo"]);
  await expect(pageMain(page).getByRole("button", { name: "Cold", exact: true })).toBeVisible();
  await settled(page, late);
  expect(late().requests, "what Hot's late answer read").toEqual([]);

  // Hot again, by the switcher: read anew, once, its words nerve's now: none, Crash and Typo listed.
  const again = recordApi(page);
  await switchByKeyboard(page, recorded, "Cold", "Hot");
  await expect.poll(() => shownItems(page)).toEqual(["Crash", "Typo"]);
  await settled(page, again);
  expect(again().requests.filter((request) => request === viewRead(hot.id))).toHaveLength(1);
  await expect(pageMain(page).getByRole("button", { name: "Remove filter", exact: true })).toHaveCount(0);

  // Low, the draft's; Update view, its answer held; Back to Cold. Let through: nothing said on Cold's page. alice then
  // makes Hot urgent elsewhere; Forward: Hot read anew, once, its words nerve's now: urgent, Crash alone.
  await addCondition(page, "Priority");
  await answerTo(page, "GET", issues, () => page.getByRole("option", { name: "Low", exact: true }).click());
  await page.keyboard.press("Escape");
  await expect.poll(() => shownItems(page)).toEqual(["Typo"]);
  await settled(page, recorded);
  const update = await sentHeld(page, "PATCH", hotPath, () => rowButton(page, "Update view").click());
  await answerTo(page, "GET", issues, async () => {
    await page.goBack();
  });
  await expect.poll(() => shownItems(page)).toEqual(["Typo"]);
  await expect(pageMain(page).getByRole("button", { name: "Cold", exact: true })).toBeVisible();
  await settled(page, recorded, 1);
  expect((await update.release()).status()).toBe(200);
  expect(await shownWithin(anyToast(page))).toBe(false);
  await expect(page).toHaveURL(viewPath(w.slug, w.web.id, cold.id));
  expect(update.body).toEqual({
    filters: { priority: ["low"] },
    display_filters: viewDefaults.display_filters,
    display_properties: viewDefaults.display_properties,
  });
  viewOf(await viewCalls(api).update(w.alice.token, hot.id, { filters: { priority: ["urgent"] } }), 200);
  const forward = recordApi(page);
  await page.goForward();
  await expect.poll(() => shownItems(page)).toEqual(["Crash"]);
  await settled(page, forward);
  expect(forward().requests.filter((request) => request === viewRead(hot.id))).toHaveLength(1);
  await expect(pageMain(page).getByRole("button", { name: "Remove filter", exact: true })).toHaveCount(1);
  await expect(rowButton(page, "Update view")).toHaveCount(0);

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});

test("H20 (page): alice's deletion of Old from its page, answered after Back came to Keep, says nothing and goes nowhere; her Update view of Keep, answered after another tab signed bob in, says nothing on his page, Keep as nerve gives it to him", async ({
  api,
  context,
  db,
  signedInPage,
}, testInfo) => {
  const w = await viewWorld(api, testInfo);
  await createIssue(api, w.alice.token, w.web.id, { name: "Crash", priority: "urgent" });
  await createIssue(api, w.alice.token, w.web.id, { name: "Typo", priority: "low" });
  const keep = await w.view(w.alice, { name: "Keep" });
  const old = await w.view(w.alice, { name: "Old" });
  const issues = `/api/v0/projects/${w.web.id}/issues`;

  const page = await signedInPage(w.alice.tokens);
  const watch = await watchPage(page);
  const recorded = recordApi(page);
  await page.goto(viewPath(w.slug, w.web.id, keep.id));
  await expect.poll(() => shownItems(page)).toEqual(["Crash", "Typo"]);

  // Old by the switcher; its header's menu, by the keyboard, deletes it, nerve's answer held; Back to Keep. Let
  // through: nothing said, Keep's page stays.
  await pageMain(page).getByRole("button", { name: "Keep", exact: true }).click();
  await answerTo(page, "GET", issues, () => page.getByRole("option", { name: "Old", exact: true }).click());
  await expect(pageMain(page).getByRole("button", { name: "Old", exact: true })).toBeVisible();
  await settled(page, recorded);
  await itemMenu(page, "Old").focus();
  await chooseByKeyboard(page, itemMenu(page, "Old"), "Delete");
  const confirm = page.getByRole("dialog", { name: "Are you sure you want to delete this view?", exact: true });
  const deleted = await sentHeld(page, "DELETE", `/api/v0/views/${old.id}`, () =>
    confirm.getByRole("button", { name: "Delete", exact: true }).click()
  );
  await answerTo(page, "GET", issues, async () => {
    await page.goBack();
  });
  await expect(pageMain(page).getByRole("button", { name: "Keep", exact: true })).toBeVisible();
  await settled(page, recorded, 1);
  expect((await deleted.release()).status()).toBe(204);
  expect(await shownWithin(anyToast(page))).toBe(false);
  await expect(page).toHaveURL(viewPath(w.slug, w.web.id, keep.id));
  expect(await shownItems(page)).toEqual(["Crash", "Typo"]);
  expect(await viewRow(db, old.id)).toMatchObject({ deleted: true, written_by: w.alice.id });

  // Urgent, the draft's; Update view, its answer held; another tab signs bob in, whose page this becomes: Keep as
  // nerve has it, urgent, Crash alone, no Update view, his menu a member's. Let through: nothing said on his page.
  await addCondition(page, "Priority");
  await answerTo(page, "GET", issues, () => page.getByRole("option", { name: "Urgent", exact: true }).click());
  await page.keyboard.press("Escape");
  await expect.poll(() => shownItems(page)).toEqual(["Crash"]);
  await settled(page, recorded);
  const update = await sentHeld(page, "PATCH", `/api/v0/views/${keep.id}`, () =>
    rowButton(page, "Update view").click()
  );
  await expect(rowButton(page, "Confirming")).toBeDisabled();
  await (await anotherTabSignsIn(context, api, w.bob.email)).close();
  await expect(rowButton(page, "Confirming")).toHaveCount(0);
  await expect.poll(() => shownItems(page)).toEqual(["Crash"]);
  await settled(page, recorded, 1);
  expect(await menuEntries(page, "Keep")).toEqual(["Open in new tab", "Copy link"]);
  expect((await update.release()).status()).toBe(200);
  expect(await shownWithin(anyToast(page))).toBe(false);
  expect(await shownItems(page)).toEqual(["Crash"]);
  await expect(rowButton(page, "Update view")).toHaveCount(0);
  await expect(pageMain(page).getByRole("button", { name: "Remove filter", exact: true })).toHaveCount(1);

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});

test("H20 (page): alice works Web's views by the keyboard: a row's menu opens and closes back onto its button; Add view's dialog goes round its icon, name, description, Cancel and Create View, its icon picker's Escape closing the picker alone, the next the dialog, back onto Add view; on Hot's page the wide header's Table and the narrow row's Calendar change the draft alone, and Update view saves it, the focus then on the filters row's Add filter", async ({
  api,
  browserName,
  signedInPage,
}, testInfo) => {
  const w = await viewWorld(api, testInfo);
  await createIssue(api, w.alice.token, w.web.id, { name: "Crash", priority: "urgent" });
  const hot = await w.view(w.alice, { name: "Hot", filters: { priority: ["urgent"] } });
  const key = tabKey(browserName);

  const page = await signedInPage(w.alice.tokens);
  const watch = await watchPage(page);
  const recorded = recordApi(page);
  await page.goto(viewsPath(w.slug, w.web.id));
  await expect.poll(() => listedViews(page)).toEqual(["Hot"]);
  await settled(page, recorded);

  // Hot's menu: Enter opens it, Escape closes it, back on its button.
  const menu = itemMenu(page, "Hot");
  await menu.focus();
  await page.keyboard.press("Enter");
  await expect(page.getByRole("menu").getByRole("menuitem")).toHaveCount(4);
  await page.keyboard.press("Escape");
  await expect(page.getByRole("menu")).toHaveCount(0);
  await expectLanded(menu);

  // Add view by Enter: its dialog, its name focused; Tab goes from its icon round its name, its description, Cancel
  // and Create View, back to its icon, and nowhere else.
  const addView = pageMain(page).getByRole("button", { name: "Add view", exact: true });
  await addView.focus();
  await page.keyboard.press("Enter");
  const dialog = modalDialog(page);
  await expect(dialog).toHaveAccessibleName("Create View");
  await expect(dialog.getByRole("textbox", { name: "Title", exact: true })).toBeFocused();
  const icon = dialog.getByRole("button", { name: "View icon", exact: true });
  await icon.focus();
  expect(await tabStops(page, dialog, key, 6)).toEqual(["Title", "Description", "Cancel", "Create View", "View icon"]);
  // The icon picker by Enter; Escape closes it alone, the dialog kept, the focus within; the next Escape the dialog,
  // the focus back on Add view.
  await page.keyboard.press("Enter");
  const iconTab = page.getByRole("tab", { name: "Icon", exact: true });
  await expect(iconTab).toBeVisible();
  expect(await closedByEscape(page)).toBe(false);
  await expect(iconTab).toHaveCount(0);
  expect(await focusInDialog(page)).toBe(true);
  await noteFocus(page, testInfo, "the icon picker closed by Escape");
  expect(await closedByEscape(page)).toBe(true);
  await expectLanded(addView);

  // Hot's page: Table by Tab and Enter on the wide header's layouts, then Calendar by the narrow row's menu; the
  // draft's alone, nothing sent, Update view offered.
  await page.getByRole("link", { name: "Hot", exact: true }).click();
  await expect.poll(() => shownItems(page)).toEqual(["Crash"]);
  await settled(page, recorded);
  const layout = (name: string) => pageMain(page).getByRole("button", { name: `${name} Layout`, exact: true });
  await layout("List").focus();
  await tabTo(page, key, layout("Table"), 3);
  await page.keyboard.press("Enter");
  await expect(layout("Table")).toHaveAttribute("aria-pressed", "true");
  await settled(page, recorded);
  const viewport = page.viewportSize();
  await page.setViewportSize({ width: 700, height: 720 });
  const narrow = pageMain(page).getByRole("button", { name: "Layout", exact: true });
  await narrow.focus();
  await chooseByKeyboard(page, narrow, "Calendar");
  if (viewport) await page.setViewportSize(viewport);
  await expect(layout("Calendar")).toHaveAttribute("aria-pressed", "true");
  await expect(layoutButtons(page)).toHaveCount(4);
  await settled(page, recorded);
  expect(recorded().requests.filter((request) => !request.startsWith("GET "))).toEqual(["POST /api/v0/auth/refresh"]);

  // Update view by Enter: saved, said; its button gone, the focus on the filters row's Add filter.
  const button = rowButton(page, "Update view");
  await button.focus();
  const saved = await sentTo(page, "PATCH", `/api/v0/views/${hot.id}`, () => page.keyboard.press("Enter"));
  expect(saved.answer.status()).toBe(200);
  await expect(toast(page, "View updated successfully.")).toBeVisible();
  await expect(button).toHaveCount(0);
  await expectLanded(pageMain(page).getByRole("button", { name: "Add filter", exact: true }));
  await noteFocus(page, testInfo, "Update view saved");
  expect(saved.body).toEqual({
    filters: { priority: ["urgent"] },
    display_filters: { ...viewDefaults.display_filters, layout: "calendar" },
    display_properties: viewDefaults.display_properties,
  });
  expect(await viewCalls(api).read(w.alice.token, hot.id)).toEqual({ status: 200, view: await saved.answer.json() });

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});
