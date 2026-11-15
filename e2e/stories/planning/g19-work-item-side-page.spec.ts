import type { Locator, Page } from "@playwright/test";

import { watchPage } from "../../fixtures/browser";
import { dayFromToday, recordApi } from "../../fixtures/cycle-pages";
import { modalDialog } from "../../fixtures/dialogs";
import { displayPanel } from "../../fixtures/issue-filters-page";
import { expectListPageQuiet } from "../../fixtures/issue-list-page";
import { modulePath, turnModules } from "../../fixtures/module-pages";
import { createModule } from "../../fixtures/modules";
import {
  bodiesSent,
  cycleListWorld,
  moduleListWorld,
  openListPage,
  shownItems,
} from "../../fixtures/planning-list-pages";
import { openCommands, selectCommand } from "../../fixtures/power-k";
import { answerTo, moveWithinApp } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// G19's page version of the work item's side (M6 P7c spec 2.4, 2.5, 2.7, 2.8, 2.12; R5, R6, R8, R9, R11, R13, R14; M3
// handoff 3), as mem uses it in his browser. The command palette's search at a project's scope names its cycles, each
// with the project's identifier, and goes to the one picked; a work item's Change cycle offers No cycle and the cycles
// it may join, never an ended one, and sends the cycle alone; its Change modules stays open and sends each pick as one
// module added or taken out, worked out in the work item's turn from the modules it then holds. A cycle's page opened
// wide and narrowed to a phone's width gives the narrow row's layouts and display: a layout picked changes the cycle's
// own preferences alone, and the display never groups a cycle's page by cycle, nor a module's by module; the cycle's
// sidebar toggle stays at that width and opens the sidebar again. A module's empty page's "add existing" opens its
// dialog with its search focused. That the narrow row waits for the page's preferences is its unit tests' (the
// preferences here are answered before the page narrows, re-76 F2). The world is made through the API.

/**
 * The groupings the display panel, opened by the header's Display, offers, in order: the options under its Group by,
 * each a button pressed or not.
 */
async function groupings(page: Page): Promise<Locator> {
  const panel = await displayPanel(page);
  const groupBy = panel.getByRole("button", { name: "Group by", exact: true });
  if ((await groupBy.getAttribute("aria-expanded")) !== "true") await groupBy.click();
  return groupBy.locator("../..").locator("button[aria-pressed]");
}

test("G19 (page): on Web's list the palette's search for spr names one cycle, Sprint, after Web's identifier, and goes to its page; on Fix the login's page Change cycle offers No cycle and Sprint, not Ended, and Sprint picked is one change of its cycle alone; Change modules stays open, Alpha picked sends the modules Alpha and, once the page shows it, picked again sends none", async ({
  api,
  browserName,
  signedInPage,
}, testInfo) => {
  const w = await cycleListWorld(api, testInfo);
  await turnModules(api, w.token, w.web, true);
  // Sprint is in progress and Ended ended; Alpha and Beta are Web's modules; Fix the login is in no cycle and no module.
  await w.cycle("Ended", { start_date: "2025-02-01", end_date: "2025-02-14" });
  const sprint = await w.cycle("Sprint", { start_date: dayFromToday(-7), end_date: dayFromToday(7) });
  const alpha = await createModule(api, w.token, w.web.id, { name: "Alpha" });
  await createModule(api, w.token, w.web.id, { name: "Beta" });
  const fix = await w.item("Fix the login");

  const page = await signedInPage(w.mem.tokens);
  const watch = await watchPage(page);
  await page.goto(w.projectPath);
  await expect.poll(() => shownItems(page)).toEqual(["Fix the login"]);

  // The search for spr, at Web's scope: its group of cycles names Sprint alone, after Web's identifier; picked, it goes
  // to Sprint's page, whose empty list says it is a cycle's.
  await openCommands(page);
  await page.getByRole("combobox").fill("spr");
  await expect(page.getByRole("group", { name: "Cycles", exact: true }).getByRole("option")).toHaveText(["WEB Sprint"]);
  await selectCommand(page, /^WEB ?Sprint$/);
  await expect(page).toHaveURL(w.path(sprint.id));
  await expect(page.getByText("No work items to show in this cycle", { exact: true })).toBeVisible();

  // Fix the login's page, once its sidebar shows it in no cycle: Change cycle offers No cycle and the cycle it may
  // join, Sprint, never Ended; Sprint picked is one change of its cycle alone, shown once nerve has it. Bodies are read
  // from now on, the page loaded (bodiesSent).
  await moveWithinApp(page, `/${w.slug}/browse/WEB-${fix.sequence_id}`);
  const cycle = (name: string) => page.getByRole("button", { name: `Cycle: ${name}`, exact: true });
  await expect(page.getByText("Created by", { exact: true })).toBeVisible();
  await expect(cycle("No cycle")).toBeVisible();
  const path = `/api/v0/issues/${fix.id}`;
  const changes = await bodiesSent(page, browserName, "PATCH", path);
  await openCommands(page);
  await selectCommand(page, /^Change cycle/);
  await expect(page.getByRole("option")).toHaveText(["No cycle", "Sprint"]);
  expect((await answerTo(page, "PATCH", path, () => selectCommand(page, /^Sprint$/))).status()).toBe(200);
  await expect(cycle("Sprint")).toBeVisible();
  expect(changes).toEqual([{ cycle_id: sprint.id }]);

  // Change modules: Alpha picked adds Fix the login to it, and the list stays open. A pick adds or takes out by the
  // modules the work item holds as nerve confirmed them, so Alpha is picked again once the page behind the palette
  // (hidden from assistive technology while it is open) shows Fix the login in Alpha: that pick takes it out, the
  // modules sent worked out in its turn.
  const modules = (name: string) =>
    page.getByRole("button", { name: `Modules: ${name}`, exact: true, includeHidden: true });
  await openCommands(page);
  await selectCommand(page, /^Change modules/);
  expect((await answerTo(page, "PATCH", path, () => selectCommand(page, /^Alpha$/))).status()).toBe(200);
  await expect(modules("Alpha")).toBeVisible();
  await expect(page.getByRole("option", { name: "Alpha", exact: true })).toBeVisible();
  expect((await answerTo(page, "PATCH", path, () => selectCommand(page, /^Alpha$/))).status()).toBe(200);
  await expect(modules("No module")).toBeVisible();
  expect(changes).toEqual([{ cycle_id: sprint.id }, { module_ids: [alpha.id] }, { module_ids: [] }]);
  await expectListPageQuiet(page, watch);
});

test("G19 (page): Sprint's page opened wide and narrowed to 390 px gives its row's Layout and Display: Board picked is one change of Sprint's own preferences, none of Web's; the display offers no grouping by cycle; the sidebar's toggle shows and opens the sidebar again; Alpha's page's display offers no grouping by module", async ({
  api,
  signedInPage,
}, testInfo) => {
  const w = await cycleListWorld(api, testInfo);
  await turnModules(api, w.token, w.web, true);
  const sprint = await w.cycle("Sprint", { start_date: dayFromToday(-7), end_date: dayFromToday(7) });
  const alpha = await createModule(api, w.token, w.web.id, { name: "Alpha" });
  await w.item("Plan", { cycle_id: sprint.id, module_ids: [alpha.id] });

  // Sprint's page, opened wide (openListPage closes its sidebar) and narrowed once it lists Plan: the narrow row's
  // layouts and display. The API requests are recorded from now on (recordApi, every engine).
  const { page, watch } = await openListPage(signedInPage, w.mem, w.path(sprint.id));
  await expect.poll(() => shownItems(page)).toEqual(["Plan"]);
  const requests = recordApi(page);
  await page.setViewportSize({ width: 390, height: 844 });
  const layout = page.getByRole("button", { name: "Layout", exact: true });
  await expect(layout).toBeVisible();
  await expect(page.getByRole("button", { name: "Display", exact: true })).toBeVisible();

  // Board picked: one change, of Sprint's own preferences, none of Web's.
  await layout.click();
  const board = await answerTo(page, "PATCH", `/api/v0/me/cycles/${sprint.id}/preferences`, () =>
    page.getByRole("menuitem", { name: "Board", exact: true }).click()
  );
  expect(board.status()).toBe(200);
  expect(requests().requests.filter((request) => request.startsWith("PATCH "))).toEqual([
    `PATCH /api/v0/me/cycles/${sprint.id}/preferences`,
  ]);

  // The display groups Sprint's board by its page's own options: by module, never by cycle.
  await expect(await groupings(page)).toHaveText([
    "States",
    "Priority",
    "Modules",
    "Labels",
    "Assignees",
    "Created by",
  ]);
  await page.keyboard.press("Escape");

  // The sidebar's toggle shows at this width, and opens the sidebar again: its head names Sprint.
  const toggle = page.getByRole("button", { name: "Toggle details", exact: true });
  await expect(toggle).toBeVisible();
  await expect(toggle).toHaveAttribute("aria-pressed", "false");
  await toggle.click();
  await expect(toggle).toHaveAttribute("aria-pressed", "true");
  await expect(page.getByRole("heading", { name: "Sprint", level: 4 })).toBeVisible();

  // Alpha's page, as narrow, once it lists Plan: its display groups by cycle, never by module.
  await moveWithinApp(page, modulePath(w.slug, w.web.id, alpha.id));
  await expect.poll(() => shownItems(page)).toEqual(["Plan"]);
  await expect(await groupings(page)).toHaveText([
    "States",
    "Priority",
    "Cycle",
    "Labels",
    "Assignees",
    "Created by",
    "None",
  ]);
  await page.keyboard.press("Escape");
  await expectListPageQuiet(page, watch);
});

test("G19 (page): Alpha's empty page's Add existing work item opens its dialog with its search focused: ArrowDown, the search never clicked, reaches Draft, which Enter chooses", async ({
  api,
  signedInPage,
}, testInfo) => {
  const w = await moduleListWorld(api, testInfo);
  const alpha = await w.module("Alpha");
  await w.item("Draft");

  // The empty state's "add existing": its dialog, once its search has nerve's answer, lists Draft, its search focused.
  const { page, watch } = await openListPage(signedInPage, w.mem, w.path(alpha.id));
  await expect(page.getByText("No work items to show in this Module", { exact: true })).toBeVisible();
  await answerTo(page, "GET", `/api/v0/workspaces/${w.slug}/issue-search`, () =>
    page.getByRole("button", { name: "Add existing work item", exact: true }).click()
  );
  const dialog = modalDialog(page);
  await expect(dialog).toHaveAccessibleName("Add existing work item");
  await expect(dialog.getByRole("option")).toHaveText(["WEB-1Draft"]);
  await expect(dialog.getByRole("combobox", { name: "Type to search", exact: true })).toBeFocused();

  // The keys alone reach Draft and choose it, which its remove button then names.
  await page.keyboard.press("ArrowDown");
  await page.keyboard.press("Enter");
  await expect(dialog.getByRole("button", { name: "Remove Draft", exact: true })).toBeVisible();
  await dialog.getByRole("button", { name: "Cancel", exact: true }).click();
  await expect(dialog).toHaveCount(0);
  await expectListPageQuiet(page, watch);
});
