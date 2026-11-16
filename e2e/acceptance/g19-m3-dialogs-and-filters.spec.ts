/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Locator, Page, TestInfo } from "@playwright/test";

import type { Api } from "../fixtures/api";
import { watchPage } from "../fixtures/browser";
import {
  chooseFromMenu,
  cyclePath,
  cycleSection,
  cyclesPath,
  entriesOf,
  itemMenu,
  linkedCycles,
  menuEntries,
  tabKey,
} from "../fixtures/cycle-pages";
import { listCycles } from "../fixtures/cycles";
import { closedByBackdropClick, closedByEscape, modalDialog, sentWithin } from "../fixtures/dialogs";
import { listedOn, modulePath, modulesPath, openModulesList, turnModules } from "../fixtures/module-pages";
import { createModule, listModules } from "../fixtures/modules";
import { cycleListWorld } from "../fixtures/planning-list-pages";
import { transitionsEnded } from "../fixtures/settings-pages";
import { expect, test } from "../fixtures/test";

// G19's acceptance check of three items of M3's handoff to M6 (handoffs/M3-closeout.md §3) on M6's own pages, which no
// story reached (M6 closeout record §3.1): a cycle's and a module's dialog, new or a change, closes on a click outside
// it as on Escape, sending nothing, the focus back where it was before it opened (ModalCore, M3 closeout spec 2.5;
// :32); the filters' and the display's popovers, the focus within them, close on Escape with the focus back on their
// button, which says they are closed (FiltersDropdown, M3 closeout spec 2.9; :38); and a module row's right-click
// menu does what its entry, clicked, says (:36), as G3 shows for a cycle's row. In each engine; the world is made
// through the API.

/**
 * The world of the checks (cycleListWorld): mem's Web, its cycles and its modules on; Sprint, its cycle, a draft;
 * Alpha and Beta, its modules, Beta described and in progress.
 */
async function m3World(api: Api, testInfo: TestInfo) {
  const w = await cycleListWorld(api, testInfo);
  await turnModules(api, w.token, w.web, true);
  const sprint = await w.cycle("Sprint");
  const alpha = await createModule(api, w.token, w.web.id, { name: "Alpha" });
  const beta = await createModule(api, w.token, w.web.id, {
    name: "Beta",
    description: "Checkout",
    status: "in-progress",
  });
  return { ...w, sprint, alpha, beta };
}

/** A button of page, named name exactly. */
function button(page: Page, name: string): Locator {
  return page.getByRole("button", { name, exact: true });
}

/**
 * Clicks the backdrop of the one dialog page shows, once its enter transitions have ended, and resolves with whether
 * it closed (closedByBackdropClick) and whether page sent a request of method to path meanwhile or within a second of
 * the close (sentWithin): the change the dialog would make.
 */
async function clickedOutside(page: Page, method: string, path: string): Promise<{ closed: boolean; sent: boolean }> {
  let closed = false;
  const sent = await sentWithin(page, method, path, async () => {
    await transitionsEnded(modalDialog(page));
    closed = await closedByBackdropClick(page);
  });
  return { closed, sent };
}

/**
 * Opens the popover of toggle, a FiltersDropdown's button, by a click, moves the focus into its panel (the one toggle
 * controls) by key, and presses Escape: the popover closes, which toggle says, and the focus is back on toggle.
 */
async function expectEscapeBackTo(page: Page, key: string, toggle: Locator): Promise<void> {
  await expect(toggle).toHaveAttribute("aria-expanded", "false");
  await toggle.click();
  await expect(toggle).toHaveAttribute("aria-expanded", "true");
  const panel = page.locator(`[id="${await toggle.getAttribute("aria-controls")}"]`);
  await expect(panel).toBeVisible();
  await transitionsEnded(panel);
  await page.keyboard.press(key);
  await expect
    .poll(() => panel.evaluate((element) => element.contains(element.ownerDocument.activeElement)))
    .toBe(true);
  await page.keyboard.press("Escape");
  await expect(toggle).toHaveAttribute("aria-expanded", "false");
  await expect(toggle).toBeFocused();
}

test("G19 (acceptance): mem's dialogs on Web's cycles list and modules list close on a click outside them and send nothing: a new cycle's and a new module's, opened from Add cycle and Add Module by the keyboard, a name typed, the focus back on that button; Sprint's and Alpha's change, opened from its row's menu and renamed, the focus back on the menu's button; nerve keeps the cycles and the modules as they were", async ({
  api,
  signedInPage,
}, testInfo) => {
  const w = await m3World(api, testInfo);
  const [cyclesBefore, modulesBefore] = [
    await listCycles(api, w.token, w.web.id),
    await listModules(api, w.token, w.web.id),
  ];
  const page = await signedInPage(w.mem.tokens);
  const watch = await watchPage(page);
  const dialog = modalDialog(page);
  const title = dialog.getByRole("textbox", { name: "Title", exact: true });
  // what a click outside a dialog does: it closes, and its change is not sent
  const outside = { closed: true, sent: false };

  // The cycles list: the new cycle dialog, opened by the keyboard from Add cycle, its name typed, closes on a click
  // outside it, sending nothing, the focus back on Add cycle.
  await page.goto(cyclesPath(w.slug, w.web.id));
  await expect.poll(() => linkedCycles(cycleSection(page, "Upcoming cycle"))).toEqual([w.sprint.id]);
  const addCycle = button(page, "Add cycle");
  await addCycle.focus();
  await page.keyboard.press("Enter");
  await expect(dialog).toHaveAccessibleName("Create cycle");
  await expect(title).toBeFocused();
  await title.fill("Held");
  expect(await clickedOutside(page, "POST", `/api/v0/projects/${w.web.id}/cycles`)).toEqual(outside);
  await expect(addCycle).toBeFocused();

  // Sprint's change, from its row's menu, holds Sprint; renamed, it closes on a click outside it, sending nothing, the
  // focus back on the menu's button.
  await chooseFromMenu(page, "Sprint", "Edit");
  await expect(dialog).toHaveAccessibleName("Update cycle");
  await expect(title).toHaveValue("Sprint");
  await title.fill("Renamed");
  expect(await clickedOutside(page, "PATCH", `/api/v0/cycles/${w.sprint.id}`)).toEqual(outside);
  await expect(itemMenu(page, "Sprint")).toBeFocused();

  // The modules list, alike: the new module dialog from Add Module, and Alpha's change from its row's menu.
  await openModulesList(page, w.slug, w.web.id);
  await expect.poll(listedOn(page)).toEqual([w.alpha.id, w.beta.id]);
  const addModule = button(page, "Add Module");
  await addModule.focus();
  await page.keyboard.press("Enter");
  await expect(dialog).toHaveAccessibleName("Create Module");
  await expect(title).toBeFocused();
  await title.fill("Held");
  expect(await clickedOutside(page, "POST", `/api/v0/projects/${w.web.id}/modules`)).toEqual(outside);
  await expect(addModule).toBeFocused();
  await chooseFromMenu(page, "Alpha", "Edit");
  await expect(dialog).toHaveAccessibleName("Update Module");
  await expect(title).toHaveValue("Alpha");
  await title.fill("Renamed");
  expect(await clickedOutside(page, "PATCH", `/api/v0/modules/${w.alpha.id}`)).toEqual(outside);
  await expect(itemMenu(page, "Alpha")).toBeFocused();

  // The page sent no change, its session's refresh aside, and nerve has the cycles and the modules as they were.
  const changes = watch.apiRequests.filter(
    (request) => !request.startsWith("GET ") && request !== "POST /api/v0/auth/refresh"
  );
  expect(changes).toEqual([]);
  expect([await listCycles(api, w.token, w.web.id), await listModules(api, w.token, w.web.id)]).toEqual([
    cyclesBefore,
    modulesBefore,
  ]);
});

test("G19 (acceptance): the filters of Web's cycles list and modules list, and the display of Sprint's page and Alpha's, each opened by a click and the focus moved into it by the keyboard, close on Escape, their button saying so and having the focus back", async ({
  api,
  browserName,
  signedInPage,
}, testInfo) => {
  const w = await m3World(api, testInfo);
  const key = tabKey(browserName);
  const page = await signedInPage(w.mem.tokens);

  // The cycles list's filters, then the display of Sprint's page, reached by its row's link.
  await page.goto(cyclesPath(w.slug, w.web.id));
  await expect.poll(() => linkedCycles(cycleSection(page, "Upcoming cycle"))).toEqual([w.sprint.id]);
  await expectEscapeBackTo(page, key, button(page, "Filters"));
  await page
    .locator(`a[href="${cyclePath(w.slug, w.web.id, w.sprint.id)}"]`)
    .first()
    .click();
  await expect(page.getByText("No work items to show in this cycle", { exact: true })).toBeVisible();
  await expectEscapeBackTo(page, key, button(page, "Display"));

  // The modules list's filters, then the display of Alpha's page, reached by its row's link.
  await openModulesList(page, w.slug, w.web.id);
  await expect.poll(listedOn(page)).toEqual([w.alpha.id, w.beta.id]);
  await expectEscapeBackTo(page, key, button(page, "Filters"));
  await page
    .locator(`a[href="${modulePath(w.slug, w.web.id, w.alpha.id)}"]`)
    .first()
    .click();
  await expect(page.getByText("No work items to show in this Module", { exact: true })).toBeVisible();
  await expectEscapeBackTo(page, key, button(page, "Display"));
});

test("G19 (acceptance): on Web's modules list a right-click on Beta's row opens its menu, the same entries as its button's, the first focused; its Edit, clicked, closes the menu and opens Beta's change dialog holding Beta's name, description and status, which Escape closes", async ({
  api,
  signedInPage,
}, testInfo) => {
  const w = await m3World(api, testInfo);
  const page = await signedInPage(w.mem.tokens);
  await page.goto(modulesPath(w.slug, w.web.id));
  await expect.poll(listedOn(page)).toEqual([w.alpha.id, w.beta.id]);
  // mem, a member, may change Beta and not delete it; Beta, in progress, may not be archived yet
  const entries = ["Edit", "Open in new tab", "Copy link", "Archive (disabled)"];
  expect(await menuEntries(page, "Beta")).toEqual(entries);

  await page.locator(`a[href="${modulePath(w.slug, w.web.id, w.beta.id)}"]`).click({ button: "right" });
  const rowMenu = page.getByRole("menu");
  expect(await entriesOf(rowMenu)).toEqual(entries);
  const edit = rowMenu.getByRole("menuitem", { name: "Edit", exact: true });
  await expect(edit).toBeFocused();
  await edit.click();
  await expect(rowMenu).toHaveCount(0);
  const dialog = modalDialog(page);
  await expect(dialog).toHaveAccessibleName("Update Module");
  await expect(dialog.getByRole("textbox", { name: "Title", exact: true })).toHaveValue("Beta");
  await expect(dialog.getByPlaceholder("Description", { exact: true })).toHaveValue("Checkout");
  await expect(dialog.getByRole("button", { name: "In Progress", exact: true })).toBeVisible();
  expect(await closedByEscape(page)).toBe(true);
});
