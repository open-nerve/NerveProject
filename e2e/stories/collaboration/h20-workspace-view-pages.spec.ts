import type { Page } from "@playwright/test";

import { createWorkspace, slugFor } from "../../fixtures/api";
import { anotherTabSignsIn } from "../../fixtures/auth";
import { expectQuietConsole, watchPage } from "../../fixtures/browser";
import { itemMenu, menuEntries, recordApi, tabKey, tabStops, toast } from "../../fixtures/cycle-pages";
import { deferred } from "../../fixtures/deferred";
import { closedByEscape, focusInDialog, modalDialog, shownWithin } from "../../fixtures/dialogs";
import { NARROW, expectLanded } from "../../fixtures/intake-focus";
import { addCondition } from "../../fixtures/issue-filters-page";
import { chooseByKeyboard } from "../../fixtures/menus";
import { rowOptions, shownItems } from "../../fixtures/planning-list-pages";
import { answerTo, holdAnswer, moveWithinApp, sentHeld, sentTo } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";
import { listedViews, pageMain, rowButton, viewRead } from "../../fixtures/view-pages";
import { viewCalls, viewOf, viewRow } from "../../fixtures/views";
import {
  noteFocus,
  orderBy,
  settled,
  switcher,
  workspaceViewContent,
  workspaceViewPath,
  workspaceViewsPath,
  workspaceViewsRead,
  workspaceViewsWorld,
} from "../../fixtures/workspace-view-pages";

// H20's page version for a workspace's views pages (M7 design 2, H20; M7/P6b spec 2.6–2.11, 2.15; R11, R14, R15;
// M3 handoff 1 and 7; v0 design 7.7): what a page sent or read belongs to its page and its session. The workspace's
// views read, answered after a move to another workspace's list; a view's read, answered after the page left it; an
// Update view and a deletion answered after their page went; an Update view answered after another tab signed another
// account in: each writes and says nothing on the page shown then, and back on the view, its page reads it anew. A
// view's page is keyed by its view: Back and Forward leave its draft and its open dialog behind. The keyboard works a
// row's menu, the view's dialog (its Tab order, the icon picker's own Escape), the header's switcher and its Display,
// and Update view, after which the focus is on the filters row's Add filter; a narrow screen keeps the header's
// controls. The project's views pages are h20-view-pages.spec.ts's.

/** The four default views, by their names, as a views list shows them first. */
const DEFAULTS = ["All work items", "Assigned", "Created", "Subscribed"];

/** A toast on page: the page said something. */
const anyToast = (page: Page) => page.getByLabel("Notifications", { exact: true }).getByRole("dialog");

/** The words Update view saves of a view of Acme's own, in nerve's words: its filters and its order as given. */
function saved(filters: Record<string, string[]>, order = "-created_at") {
  return {
    filters,
    display_filters: { ...workspaceViewContent.display_filters, order_by: order },
    display_properties: workspaceViewContent.display_properties,
  };
}

/**
 * Opens the view named to from the header of the page of the view named from by the keyboard (its switcher's
 * CustomSearchSelect): Enter on its button opens its list, its search focused; to typed, ArrowDown makes its option the
 * active one, and Enter opens it, once the reads out (the workspace's own views, read as the list opens) are answered.
 */
async function switchByKeyboard(page: Page, recorded: () => { pending: number }, from: string, to: string) {
  await switcher(page, from).focus();
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

test("H20 (page): Acme's views read, answered after a move to Beta's views, lists nothing there; Hot's read, answered after Cold's switcher opened Hot by the keyboard and Back came to Cold, writes nothing into Cold's page, and Hot opened again reads it anew, once, as alice changed it meanwhile; Back to Cold leaves Hot's draft and its open dialog behind, Cold's page its own, and Forward reads Hot anew, neither back", async ({
  api,
  signedInPage,
}, testInfo) => {
  const w = await workspaceViewsWorld(api, testInfo);
  const hot = await w.view(w.alice, { name: "Hot", filters: { priority: ["urgent"] } });
  const cold = await w.view(w.alice, { name: "Cold", filters: { priority: ["low"] } });
  const beta = slugFor(testInfo, "beta");
  await createWorkspace(api, w.alice.token, { name: "Beta", slug: beta });
  await w.view(w.alice, { name: "Beta board" }, beta);
  const issues = `/api/v0/workspaces/${w.slug}/issues`;
  const hotPath = `/api/v0/views/${hot.id}`;

  // Cold's page, then Beta's views: both pages' code loaded before any route is (WebKit's preloads).
  const page = await signedInPage(w.alice.tokens);
  const watch = await watchPage(page);
  const recorded = recordApi(page);
  await page.goto(workspaceViewPath(w.slug, cold.id));
  await expect.poll(() => shownItems(page)).toEqual(["Typo"]);
  await settled(page, recorded);
  const toBeta = async () => {
    await moveWithinApp(page, workspaceViewsPath(beta));
    await expect(page).toHaveTitle("Beta - Views");
    await expect.poll(() => listedViews(page)).toEqual([...DEFAULTS, "Beta board"]);
  };
  await toBeta();
  await settled(page, recorded);

  // Acme's views, their read held, then Beta's again: let through, Beta's list as it was; Back, Acme's own.
  const releaseList = await holdAnswer(page, "GET", `/api/v0/workspaces/${w.slug}/views`);
  await moveWithinApp(page, workspaceViewsPath(w.slug));
  await expect(page).toHaveTitle("Acme - Views");
  await expect.poll(() => recorded().requests).toContain(workspaceViewsRead(w.slug));
  await toBeta();
  await settled(page, recorded, 1);
  expect((await answerTo(page, "GET", `/api/v0/workspaces/${w.slug}/views`, releaseList)).status()).toBe(200);
  expect(await shownWithin(page.getByRole("link", { name: "Hot", exact: true })), "Acme's Hot, on Beta's list").toBe(
    false
  );
  expect(await listedViews(page)).toEqual([...DEFAULTS, "Beta board"]);
  await page.goBack();
  await expect(page).toHaveTitle("Acme - Views");
  await expect.poll(() => listedViews(page)).toEqual([...DEFAULTS, "Cold", "Hot"]);
  await settled(page, recorded);

  // Cold's page; its switcher, by the keyboard, opens Hot, whose read nerve answers and the page waits for, alice then
  // changing Hot elsewhere; Back to Cold, then Hot's late answer: Cold's page as it was, its list, its name.
  await answerTo(page, "GET", issues, () => page.getByRole("link", { name: "Cold", exact: true }).click());
  await expect.poll(() => shownItems(page)).toEqual(["Typo"]);
  await settled(page, recorded);
  const answered = deferred();
  const releaseHot = await holdAnswer(page, "GET", hotPath, 1, () => answered.resolve());
  await switchByKeyboard(page, recorded, "Cold", "Hot");
  await expect(page).toHaveURL(workspaceViewPath(w.slug, hot.id));
  await answered.promise;
  viewOf(await viewCalls(api).update(w.alice.token, hot.id, { filters: {} }), 200);
  // Cold mounted again reads Cold, then its list: both answered
  await answerTo(page, "GET", issues, async () => {
    await page.goBack();
  });
  await expect(page).toHaveURL(workspaceViewPath(w.slug, cold.id));
  await expect.poll(() => shownItems(page)).toEqual(["Typo"]);
  await settled(page, recorded, 1);
  const late = recordApi(page);
  expect((await answerTo(page, "GET", hotPath, releaseHot)).status()).toBe(200);
  expect(await shownWithin(rowOptions(page, "Crash")), "Hot's Crash, on Cold's page").toBe(false);
  expect(await shownItems(page)).toEqual(["Typo"]);
  await expect(switcher(page, "Cold")).toBeVisible();
  await settled(page, late);
  expect(late().requests, "what Hot's late answer read").toEqual([]);

  // Hot again, by the switcher: read anew, once, its words nerve's now: none, Crash and Typo listed.
  const again = recordApi(page);
  await switchByKeyboard(page, recorded, "Cold", "Hot");
  await expect.poll(() => shownItems(page)).toEqual(["Crash", "Typo"]);
  await settled(page, again);
  expect(again().requests.filter((request) => request === viewRead(hot.id))).toHaveLength(1);
  await expect(pageMain(page).getByRole("button", { name: "Remove filter", exact: true })).toHaveCount(0);

  // Low, Hot's draft's, and Add view's dialog open, a name typed; Back to Cold: no dialog, Cold's own words and list.
  // Forward: Hot read anew, once, nerve's words, no dialog, nothing to update.
  await addCondition(page, "Priority");
  await answerTo(page, "GET", issues, () => page.getByRole("option", { name: "Low", exact: true }).click());
  await page.keyboard.press("Escape");
  await expect.poll(() => shownItems(page)).toEqual(["Typo"]);
  await expect(rowButton(page, "Update view")).toBeVisible();
  await pageMain(page).getByRole("button", { name: "Add view", exact: true }).click();
  await expect(modalDialog(page)).toHaveAccessibleName("Create View");
  await modalDialog(page).getByRole("textbox", { name: "Title", exact: true }).fill("Unsaved");
  await settled(page, recorded);
  await answerTo(page, "GET", issues, async () => {
    await page.goBack();
  });
  await expect(switcher(page, "Cold")).toBeVisible();
  await expect(modalDialog(page)).toHaveCount(0);
  await expect.poll(() => shownItems(page)).toEqual(["Typo"]);
  await expect(pageMain(page).getByRole("button", { name: "Remove filter", exact: true })).toHaveCount(1);
  await expect(rowButton(page, "Update view")).toHaveCount(0);
  await settled(page, recorded);
  const forward = recordApi(page);
  await answerTo(page, "GET", issues, async () => {
    await page.goForward();
  });
  await expect.poll(() => shownItems(page)).toEqual(["Crash", "Typo"]);
  await settled(page, forward);
  expect(forward().requests.filter((request) => request === viewRead(hot.id))).toHaveLength(1);
  await expect(modalDialog(page)).toHaveCount(0);
  await expect(pageMain(page).getByRole("button", { name: "Remove filter", exact: true })).toHaveCount(0);
  await expect(rowButton(page, "Update view")).toHaveCount(0);

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});

test("H20 (page): alice's Update view of Old, answered after Back came to Keep, says nothing there, and Forward reads Old anew, its words nerve's; her deletion of Old from its page's menu by the keyboard, answered after Back came to Keep, says nothing and goes nowhere; her Update view of Keep, answered after another tab signed bob in, says nothing on his page, Keep as nerve gives it to him", async ({
  api,
  context,
  db,
  signedInPage,
}, testInfo) => {
  const w = await workspaceViewsWorld(api, testInfo);
  const keep = await w.view(w.alice, { name: "Keep" });
  const old = await w.view(w.alice, { name: "Old" });
  const issues = `/api/v0/workspaces/${w.slug}/issues`;

  const page = await signedInPage(w.alice.tokens);
  const watch = await watchPage(page);
  const recorded = recordApi(page);
  await page.goto(workspaceViewPath(w.slug, keep.id));
  await expect.poll(() => shownItems(page)).toEqual(["Crash", "Typo"]);
  /** Urgent, the draft's condition: the list read under it, Crash alone. */
  const urgent = async () => {
    await addCondition(page, "Priority");
    await answerTo(page, "GET", issues, () => page.getByRole("option", { name: "Urgent", exact: true }).click());
    await page.keyboard.press("Escape");
    await expect.poll(() => shownItems(page)).toEqual(["Crash"]);
  };
  /** Back to Keep, whose page mounted again reads Keep, then its list: both answered, all as nerve has Keep. */
  const backToKeep = async () => {
    await answerTo(page, "GET", issues, async () => {
      await page.goBack();
    });
    await expect(switcher(page, "Keep")).toBeVisible();
    await expect(page).toHaveURL(workspaceViewPath(w.slug, keep.id));
    await expect.poll(() => shownItems(page)).toEqual(["Crash", "Typo"]);
  };

  // Old by the switcher; Urgent, Update view, its answer held; Back to Keep. Let through: nothing said; Forward reads
  // Old anew, urgent as nerve has it now, nothing to update.
  await switcher(page, "Keep").click();
  await answerTo(page, "GET", issues, () => page.getByRole("option", { name: "Old", exact: true }).click());
  await expect(switcher(page, "Old")).toBeVisible();
  await expect.poll(() => shownItems(page)).toEqual(["Crash", "Typo"]);
  await settled(page, recorded);
  await urgent();
  await settled(page, recorded);
  const update = await sentHeld(page, "PATCH", `/api/v0/views/${old.id}`, () => rowButton(page, "Update view").click());
  await backToKeep();
  await settled(page, recorded, 1);
  expect((await update.release()).status()).toBe(200);
  expect(await shownWithin(anyToast(page))).toBe(false);
  await expect(page).toHaveURL(workspaceViewPath(w.slug, keep.id));
  expect(update.body).toEqual(saved({ priority: ["urgent"] }));
  const forward = recordApi(page);
  await answerTo(page, "GET", issues, async () => {
    await page.goForward();
  });
  await expect.poll(() => shownItems(page)).toEqual(["Crash"]);
  await settled(page, forward);
  expect(forward().requests.filter((request) => request === viewRead(old.id))).toHaveLength(1);
  await expect(pageMain(page).getByRole("button", { name: "Remove filter", exact: true })).toHaveCount(1);
  await expect(rowButton(page, "Update view")).toHaveCount(0);

  // Old's header's menu, by the keyboard, deletes it, nerve's answer held; Back to Keep. Let through: nothing said,
  // Keep's page stays; Old deleted by alice as stored.
  await itemMenu(page, "Old").focus();
  await chooseByKeyboard(page, itemMenu(page, "Old"), "Delete");
  const confirm = page.getByRole("dialog", { name: "Are you sure you want to delete this view?", exact: true });
  const deleted = await sentHeld(page, "DELETE", `/api/v0/views/${old.id}`, () =>
    confirm.getByRole("button", { name: "Delete", exact: true }).click()
  );
  await backToKeep();
  await settled(page, recorded, 1);
  expect((await deleted.release()).status()).toBe(204);
  expect(await shownWithin(anyToast(page))).toBe(false);
  await expect(page).toHaveURL(workspaceViewPath(w.slug, keep.id));
  expect(await shownItems(page)).toEqual(["Crash", "Typo"]);
  expect(await viewRow(db, old.id)).toMatchObject({ deleted: true, written_by: w.alice.id });

  // Urgent, the draft's; Update view, its answer held; another tab signs bob in, whose page this becomes: Keep as nerve
  // has it, urgent, Crash alone, no Update view, his menu a member's. Let through: nothing said on his page.
  await urgent();
  await settled(page, recorded);
  const kept = await sentHeld(page, "PATCH", `/api/v0/views/${keep.id}`, () => rowButton(page, "Update view").click());
  await expect(rowButton(page, "Confirming")).toBeDisabled();
  await (await anotherTabSignsIn(context, api, w.bob.email)).close();
  await expect(rowButton(page, "Confirming")).toHaveCount(0);
  await expect.poll(() => shownItems(page)).toEqual(["Crash"]);
  await settled(page, recorded, 1);
  expect(await menuEntries(page, "Keep")).toEqual(["Open in new tab", "Copy link"]);
  expect((await kept.release()).status()).toBe(200);
  expect(await shownWithin(anyToast(page))).toBe(false);
  expect(await shownItems(page)).toEqual(["Crash"]);
  await expect(rowButton(page, "Update view")).toHaveCount(0);
  await expect(pageMain(page).getByRole("button", { name: "Remove filter", exact: true })).toHaveCount(1);

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});

test("H20 (page): alice works Acme's views by the keyboard: Hot's row menu opens and closes back onto its button; Add view's dialog goes round its icon, name, description, Cancel and Create View, its icon picker's Escape closing the picker alone, the next the dialog, back onto Add view; on Hot's page Display, named so, says whether it is open and Escape gives the focus back to it; the switcher opens Cold by the keyboard; 800 px wide, the header's controls are all there; Cold's order changed, Update view by Enter saves it, the focus then on the filters row's Add filter", async ({
  api,
  browserName,
  signedInPage,
}, testInfo) => {
  const w = await workspaceViewsWorld(api, testInfo);
  await w.view(w.alice, { name: "Hot", filters: { priority: ["urgent"] } });
  const cold = await w.view(w.alice, { name: "Cold", filters: { priority: ["low"] } });
  const issues = `/api/v0/workspaces/${w.slug}/issues`;
  const key = tabKey(browserName);

  const page = await signedInPage(w.alice.tokens);
  const watch = await watchPage(page);
  const recorded = recordApi(page);
  await page.goto(workspaceViewsPath(w.slug));
  await expect.poll(() => listedViews(page)).toEqual([...DEFAULTS, "Cold", "Hot"]);
  await settled(page, recorded);

  // Hot's menu: Enter opens it, Escape closes it, back on its button.
  const menu = itemMenu(page, "Hot");
  await menu.focus();
  await page.keyboard.press("Enter");
  await expect(page.getByRole("menu").getByRole("menuitem")).toHaveCount(4);
  await page.keyboard.press("Escape");
  await expect(page.getByRole("menu")).toHaveCount(0);
  await expectLanded(menu);
  await noteFocus(page, testInfo, "Hot's menu closed by Escape");

  // Add view by Enter: its dialog, its name focused; Tab goes from its icon round its name, its description, Cancel
  // and Create View, back to its icon, and nowhere else. The icon picker by Enter; Escape closes it alone, the dialog
  // kept, the focus within; the next Escape the dialog, the focus back on Add view.
  const addView = pageMain(page).getByRole("button", { name: "Add view", exact: true });
  await addView.focus();
  await page.keyboard.press("Enter");
  const dialog = modalDialog(page);
  await expect(dialog).toHaveAccessibleName("Create View");
  await expect(dialog.getByRole("textbox", { name: "Title", exact: true })).toBeFocused();
  await dialog.getByRole("button", { name: "View icon", exact: true }).focus();
  expect(await tabStops(page, dialog, key, 6)).toEqual(["Title", "Description", "Cancel", "Create View", "View icon"]);
  await page.keyboard.press("Enter");
  const iconTab = page.getByRole("tab", { name: "Icon", exact: true });
  await expect(iconTab).toBeVisible();
  expect(await closedByEscape(page)).toBe(false);
  await expect(iconTab).toHaveCount(0);
  expect(await focusInDialog(page)).toBe(true);
  await noteFocus(page, testInfo, "the icon picker closed by Escape");
  expect(await closedByEscape(page)).toBe(true);
  await expectLanded(addView);
  await noteFocus(page, testInfo, "Create View closed by Escape");

  // Hot's page: Display, named so, closed, then open; Escape closes it, the focus back on it.
  await answerTo(page, "GET", issues, () => page.getByRole("link", { name: "Hot", exact: true }).click());
  await expect.poll(() => shownItems(page)).toEqual(["Crash"]);
  await settled(page, recorded);
  const display = pageMain(page).getByRole("button", { name: "Display", exact: true });
  await expect(display).toHaveAttribute("aria-expanded", "false");
  await display.click();
  await expect(display).toHaveAttribute("aria-expanded", "true");
  await page.keyboard.press("Escape");
  await expect(display).toHaveAttribute("aria-expanded", "false");
  await expectLanded(display);
  await noteFocus(page, testInfo, "Display closed by Escape");

  // The switcher, by the keyboard: Cold.
  await answerTo(page, "GET", issues, () => switchByKeyboard(page, recorded, "Hot", "Cold"));
  await expect(page).toHaveURL(workspaceViewPath(w.slug, cold.id));
  await expect.poll(() => shownItems(page)).toEqual(["Typo"]);
  await settled(page, recorded);

  // 800 px wide: the header's switcher, filters, Display, Add view and the view's menu are all on the screen.
  const viewport = page.viewportSize();
  await page.setViewportSize(NARROW);
  const header = [switcher(page, "Cold"), pageMain(page).getByRole("button", { name: "Filters", exact: true })];
  await Promise.all(
    [...header, display, addView, itemMenu(page, "Cold")].map((control) => expect(control).toBeInViewport())
  );
  await display.click();
  await expect(display).toHaveAttribute("aria-expanded", "true");
  await page.keyboard.press("Escape");
  await expect(display).toHaveAttribute("aria-expanded", "false");
  if (viewport) await page.setViewportSize(viewport);

  // Cold by priority: Update view by Enter saves it, says so; its button gone, the focus on the filters row's Add
  // filter.
  await answerTo(page, "GET", issues, () => orderBy(page, "Priority", "Urgent None"));
  const button = rowButton(page, "Update view");
  await expect(button).toBeVisible();
  await settled(page, recorded);
  await button.focus();
  const sent = await sentTo(page, "PATCH", `/api/v0/views/${cold.id}`, () => page.keyboard.press("Enter"));
  expect(sent.answer.status()).toBe(200);
  await expect(toast(page, "View updated successfully.")).toBeVisible();
  await expect(button).toHaveCount(0);
  await expectLanded(pageMain(page).getByRole("button", { name: "Add filter", exact: true }));
  await noteFocus(page, testInfo, "Update view saved by Enter");
  expect(sent.body).toEqual(saved({ priority: ["low"] }, "-priority"));
  expect(await viewCalls(api).read(w.alice.token, cold.id)).toEqual({ status: 200, view: await sent.answer.json() });

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});
