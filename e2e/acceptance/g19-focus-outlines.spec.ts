/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Locator, Page, TestInfo } from "@playwright/test";

import type { Api } from "../fixtures/api";
import { expectOutlinesWhole } from "../fixtures/browser";
import { cyclePath, cycleSection, cyclesPath, dayFromToday, sectionToggle, tabKey } from "../fixtures/cycle-pages";
import { addCondition, displayPanel } from "../fixtures/issue-filters-page";
import { modulePath, modulesPath, moduleRow, turnModules } from "../fixtures/module-pages";
import { createModule, createModuleLink } from "../fixtures/modules";
import { cycleListWorld, openListPage, shownItems } from "../fixtures/planning-list-pages";
import { transitionsEnded } from "../fixtures/settings-pages";
import { expect, test } from "../fixtures/test";

// G19's acceptance check of the focus outlines on the cycle and the module pages (M6 P8 spec 2.5, 2.11; M3 handoff 3;
// P7a–P7c spec 5): each control, reached by the keyboard as a person tabs to it, shows its outline whole, drawn and cut
// by neither the viewport nor a box that hides what overflows it (expectOutlinesWhole), in each engine. The seven
// controls of spec 2.5 — (a) a row's link, (b) a module's link in its sidebar, (c) Add link, (d) the breadcrumbs' link,
// (e) the days' range, (f) the sidebar's tab, (g) a cycles list's section — the shared controls of M3 handoff 3
// (Button, IconButton, getButtonStyling, CustomMenu), P7a's cycle and modules dropdowns, P7b's filter and grouping and
// its two "add existing" entries, P7c's small screen's row and the cycle page's top row at 390 x 844, and a row's single
// day (M4's DateDropdown). The world is made through the API; Web's cycles and modules are on.

/**
 * The world of the checks (cycleListWorld): mem's Web, its modules on; Alpha, its module with days and a link, and Beta;
 * Sprint, its cycle in progress, and Next, ahead; Plan and Ship, in Sprint and Alpha.
 */
async function outlinesWorld(api: Api, testInfo: TestInfo) {
  const w = await cycleListWorld(api, testInfo);
  await turnModules(api, w.token, w.web, true);
  // nerve lists the newest first: Sprint (in progress), Next (upcoming, of no work item); Alpha, Beta (of none).
  const next = await w.cycle("Next", { start_date: dayFromToday(20), end_date: dayFromToday(30) });
  const sprint = await w.cycle("Sprint", { start_date: dayFromToday(-3), end_date: dayFromToday(10) });
  const beta = await createModule(api, w.token, w.web.id, { name: "Beta" });
  const alpha = await createModule(api, w.token, w.web.id, {
    name: "Alpha",
    start_date: "2031-01-01",
    target_date: "2031-01-31",
  });
  await createModuleLink(api, w.token, alpha.id, { url: "http://169.254.169.254/g19/notes", title: "Notes" });
  // Plan has no days, so its row gives each of its days by itself; Ship has both, so its row gives their range.
  await w.item("Plan", { cycle_id: sprint.id, module_ids: [alpha.id] });
  await w.item("Ship", {
    cycle_id: sprint.id,
    module_ids: [alpha.id],
    start_date: dayFromToday(1),
    target_date: dayFromToday(3),
  });
  return { ...w, next, sprint, alpha, beta };
}

/** A button of page, named name exactly. */
function button(page: Page, name: string): Locator {
  return page.getByRole("button", { name, exact: true });
}

/** The page's breadcrumb link of name: in its header, not the sidebar's link of the same name. */
function crumb(page: Page, name: string): Locator {
  return page.getByRole("main").last().getByRole("link", { name, exact: true });
}

/**
 * The work items' controls of a cycle's or a module's page, which page shows with its sidebar closed: the header's
 * filter (getButtonStyling), display (Button's look), new work item (Button), sidebar toggle (IconButton) and menu of
 * the page's item (CustomMenu); the group's new and "add existing" entries; Plan's cycle and modules (P7a) and its single
 * days (M4's DateDropdown); Ship's range of days (e). Then a condition of filter added, its property, its value and its
 * removal (P7b), and the display's grouping of group (P7b).
 */
async function workItemControls(page: Page, key: string, item: string, filter: string, value: string, group: string) {
  const row = (name: string) => page.getByRole("button", { name: new RegExp(`^${name}: .* , Plan$`) });
  await expectOutlinesWhole(
    page,
    key,
    ...["Add filter", "Display", "Add work item", "Toggle details", `${item} options`].map((name) =>
      button(page, name)
    ),
    button(page, "Create work item: All work items"),
    button(page, "Add existing work item: All work items"),
    row("Modules"),
    row("Cycle"),
    page.getByRole("link", { name: /^WEB-1 Plan / }).getByRole("button", { name: "Start date", exact: true }),
    page.getByRole("link", { name: /^WEB-1 Plan / }).getByRole("button", { name: "Due date", exact: true }),
    page.getByRole("link", { name: /^WEB-2 Ship / }).getByRole("button", { name: "Date Range", exact: true })
  );

  // A condition of filter, its value picked: the condition's property, its value and its removal.
  const chip = await addCondition(page, filter);
  await page.getByRole("option", { name: value, exact: true }).click();
  await page.keyboard.press("Escape");
  await expect(chip.getByRole("button", { name: value, exact: true })).toBeVisible();
  await expectOutlinesWhole(
    page,
    key,
    chip.getByRole("button", { name: filter, exact: true }),
    chip.getByRole("button", { name: value, exact: true }),
    chip.getByRole("button", { name: "Remove filter", exact: true })
  );
  await chip.getByRole("button", { name: "Remove filter", exact: true }).click();
  await expect(chip).toHaveCount(0);

  // The display's grouping by group, within its panel.
  const groupBy = (await displayPanel(page)).getByRole("button", { name: "Group by", exact: true });
  if ((await groupBy.getAttribute("aria-expanded")) !== "true") await groupBy.click();
  await expectOutlinesWhole(page, key, groupBy.locator("../..").getByRole("button", { name: group, exact: true }));
  await page.keyboard.press("Escape");
}

/** The sidebar of a cycle's or a module's page, which page opens by its toggle: its days' range (e) and its tab (f). */
async function sidebarControls(page: Page, key: string, ...more: Locator[]) {
  const toggle = button(page, "Toggle details");
  await toggle.click();
  await expect(toggle).toHaveAttribute("aria-pressed", "true");
  await expectOutlinesWhole(page, key, button(page, "Date range"), page.getByRole("tab", { selected: true }), ...more);
}

/**
 * Sets page's window to a phone's, 390 x 844 (small), or back to the test's, and waits for the app's sidebar to have
 * closed or opened for it: the main area moves with a transition, during which a control measured is wherever the
 * transition has it. Closed, the main area starts where its box does; open, after the sidebar.
 */
async function atWidth(page: Page, small: boolean) {
  const main = page.getByRole("main").last();
  await page.setViewportSize(small ? { width: 390, height: 844 } : { width: 1280, height: 720 });
  const sidebarWidth = () =>
    main.evaluate((element) =>
      Math.round(element.getBoundingClientRect().left - (element.parentElement?.getBoundingClientRect().left ?? 0))
    );
  if (small) await expect.poll(sidebarWidth).toBeLessThanOrEqual(1);
  else await expect.poll(sidebarWidth).toBeGreaterThan(0);
  await transitionsEnded(page.locator("body"));
}

/** A page's small screen (390 x 844): its row's layouts and display (P7c), and its header's toggle and menu. */
async function smallScreenControls(page: Page, key: string, item: string) {
  await atWidth(page, true);
  await expect(button(page, "Layout")).toBeVisible();
  await expectOutlinesWhole(
    page,
    key,
    ...["Layout", "Display", "Toggle details", `${item} options`].map((name) => button(page, name))
  );
  await atWidth(page, false);
}

test("G19 (acceptance): Web's cycles list, wide and at 390 x 844: its search, filters and Add cycle, the breadcrumbs' Cycles, the card of Sprint in progress with its section, its menu, its link, its details and its tab, and the sections with Next's link and menu show their focus outlines whole", async ({
  api,
  browserName,
  signedInPage,
}, testInfo) => {
  const w = await outlinesWorld(api, testInfo);
  const key = tabKey(browserName);
  const page = await signedInPage(w.mem.tokens);
  await page.goto(cyclesPath(w.slug, w.web.id));
  const card = cycleSection(page, "Active cycle");
  const sprintLink = card.locator(`a[href="${cyclePath(w.slug, w.web.id, w.sprint.id)}"]`).first();
  await expect(sprintLink).toBeVisible();
  const nextLink = cycleSection(page, "Upcoming cycle").locator(`a[href="${cyclePath(w.slug, w.web.id, w.next.id)}"]`);
  await expect(nextLink.first()).toBeVisible();

  // The header's, and the card's: its section's button, Sprint's link, its details and its tab.
  const cardToggle = card.getByRole("button", { name: "Active cycle", exact: true });
  await expectOutlinesWhole(
    page,
    key,
    ...["Search", "Filters", "Add cycle", "Sprint options"].map((name) => button(page, name)),
    crumb(page, "Cycles"),
    cardToggle,
    sprintLink,
    card.getByRole("button", { name: "More details", exact: true }),
    card.getByRole("tab", { selected: true })
  );
  // The card closed, the sections below it in view: theirs, and Next's link and menu.
  await cardToggle.click();
  await expect(cardToggle).toHaveAttribute("aria-expanded", "false");
  await expectOutlinesWhole(
    page,
    key,
    sectionToggle(page, "Upcoming cycle"),
    sectionToggle(page, "Completed cycle"),
    nextLink.first(),
    button(page, "Next options")
  );

  // At 390 x 844: the breadcrumbs' Cycles, alone once they have rendered for the small screen ("..." before it), a
  // section and Next's link.
  await atWidth(page, true);
  await expect(page.getByRole("main").last().getByText("...", { exact: true }).first()).toBeVisible();
  await expectOutlinesWhole(page, key, crumb(page, "Cycles"), sectionToggle(page, "Upcoming cycle"), nextLink.first());
});

test("G19 (acceptance): Sprint's page, wide and at 390 x 844, and Next's, empty: its breadcrumbs, header, group, rows, filter, grouping, sidebar and small screen's row show their focus outlines whole, and so does the empty page's Add existing work item", async ({
  api,
  browserName,
  signedInPage,
}, testInfo) => {
  const w = await outlinesWorld(api, testInfo);
  const key = tabKey(browserName);
  const { page } = await openListPage(signedInPage, w.mem, w.path(w.sprint.id));
  await expect.poll(() => shownItems(page)).toEqual(["Plan", "Ship"]);

  await expectOutlinesWhole(page, key, crumb(page, "Cycles"), button(page, "Sprint"));
  await workItemControls(page, key, "Sprint", "Module", "Alpha", "Modules");
  await sidebarControls(page, key);
  await smallScreenControls(page, key, "Sprint");

  // Next's page, of no work item: its empty state's Add existing work item.
  await page.getByRole("button", { name: "Sprint", exact: true }).click();
  await page.getByRole("option", { name: "Next", exact: true }).click();
  await expect(page.getByText("No work items to show in this cycle", { exact: true })).toBeVisible();
  await expectOutlinesWhole(page, key, button(page, "Add existing work item"));
});

test("G19 (acceptance): Web's modules list and Alpha's page, wide and at 390 x 844, and Beta's, empty: Alpha's row's link and days, the breadcrumbs' Modules, the page's header, group, rows, filter, grouping, its sidebar's link, Add link, days and tab, and its small screen's row show their focus outlines whole, and so does the empty page's Add existing work item", async ({
  api,
  browserName,
  signedInPage,
}, testInfo) => {
  const w = await outlinesWorld(api, testInfo);
  const key = tabKey(browserName);
  const page = await signedInPage(w.mem.tokens);
  await page.goto(modulesPath(w.slug, w.web.id));
  const alphaRow = moduleRow(page, w.alpha.id, "Alpha");
  const alphaLink = alphaRow.locator(`a[href="${modulePath(w.slug, w.web.id, w.alpha.id)}"]`).first();
  await expect(alphaLink).toBeVisible();
  await expectOutlinesWhole(
    page,
    key,
    alphaLink,
    alphaRow.getByRole("button", { name: "Date range", exact: true }),
    crumb(page, "Modules")
  );

  // Alpha's page, its sidebar closed, then open.
  await alphaLink.click();
  await expect(page.getByRole("heading", { name: "Alpha", exact: true })).toBeVisible();
  const toggle = button(page, "Toggle details");
  await expect(toggle).toHaveAttribute("aria-pressed", "true");
  await toggle.click();
  await expect(toggle).toHaveAttribute("aria-pressed", "false");
  await expect.poll(() => shownItems(page)).toEqual(["Plan", "Ship"]);
  await expectOutlinesWhole(page, key, crumb(page, "Modules"), button(page, "Alpha"));
  await workItemControls(page, key, "Alpha", "Cycle", "Sprint", "Cycle");
  await sidebarControls(page, key, page.getByRole("link", { name: "Notes", exact: true }), button(page, "Add link"));
  await smallScreenControls(page, key, "Alpha");

  // Beta's page, of no work item: its empty state's Add existing work item.
  await page.getByRole("button", { name: "Alpha", exact: true }).click();
  await page.getByRole("option", { name: "Beta", exact: true }).click();
  await expect(page.getByText("No work items to show in this Module", { exact: true })).toBeVisible();
  await expectOutlinesWhole(page, key, button(page, "Add existing work item"));
});
