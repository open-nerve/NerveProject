/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Page } from "@playwright/test";

import { dayFromToday } from "../fixtures/cycle-pages";
import { cycleListWorld, openListPage, shownItems } from "../fixtures/planning-list-pages";
import { openCommands, selectCommand } from "../fixtures/power-k";
import { expect, test } from "../fixtures/test";

// G19's acceptance check of the command palette's rows (M6 P8 spec 2.11; P7c spec 5): the rows are no Tab stops, the
// focus staying in the palette's search throughout, whose own outline is off as Plane's (the open palette is where the
// focus is), so no row's outline is measured; the row the arrows select says so (aria-selected), shows it (its
// background, which no other row has) and is whole within the list's scrolling box, scrolled to as the arrows move down
// the list and back up, in each engine; so do the rows of the menu of the project's cycles. The world is made through
// the API.

/** Where the palette's list stands: the indexes of its rows selected, and of those, whether it shows and is whole. */
interface Selection {
  readonly selected: number[];
  /** Its background is drawn, and differs from the background of a row not selected. */
  readonly highlighted: boolean;
  /** Its box is within the box of the list, which scrolls its rows. */
  readonly whole: boolean;
}

/** The palette's list, as it stands: its rows that may be chosen, in order (a disabled one the arrows skip). */
function selection(page: Page): Promise<Selection> {
  return page.getByRole("listbox").evaluate((list) => {
    const rows = Array.from(list.querySelectorAll('[role="option"]')).filter(
      (row) => row.getAttribute("aria-disabled") !== "true"
    );
    const selected = rows.flatMap((row, index) => (row.getAttribute("aria-selected") === "true" ? [index] : []));
    const row = rows[selected[0] ?? -1];
    const other = rows.find((each) => each.getAttribute("aria-selected") !== "true");
    if (!row || !other) return { selected, highlighted: false, whole: false };
    const background = getComputedStyle(row).backgroundColor;
    const drawn = !/^(transparent|rgba\(.*,\s*0\))$/.test(background);
    const [box, own] = [list.getBoundingClientRect(), row.getBoundingClientRect()];
    // within a pixel, as focusOutline's boxes: the list at its end scrolls by whole pixels, its rows lie at fractions
    return {
      selected,
      highlighted: drawn && background !== getComputedStyle(other).backgroundColor,
      whole:
        own.top >= box.top - 1 &&
        own.bottom <= box.bottom + 1 &&
        own.left >= box.left - 1 &&
        own.right <= box.right + 1,
    };
  });
}

/** The number of rows of the palette's list the arrows may select. */
function rowCount(page: Page): Promise<number> {
  return page
    .getByRole("listbox")
    .getByRole("option")
    .evaluateAll((rows) => rows.filter((row) => row.getAttribute("aria-disabled") !== "true").length);
}

/**
 * Walks the palette's list by the arrows from its first row to its last and back, checking at each row that it alone
 * is selected, shows it and is whole within the list's box; the search keeps the focus throughout. The palette opens
 * with its first row selected; a menu opened from it, with none, until the first arrow selects its first row (cmdk
 * keeps the value of the row chosen, which none of the menu's rows has; Plane's palette the same).
 */
async function walkRows(page: Page, opensSelected: boolean): Promise<void> {
  const search = page.getByRole("combobox");
  const count = await rowCount(page);
  expect(count, "rows enough to scroll").toBeGreaterThan(12);
  if (!opensSelected) {
    await expect.poll(() => selection(page)).toEqual({ selected: [], highlighted: false, whole: false });
    await search.press("ArrowDown");
  }
  const at = async (index: number) => {
    await expect
      .poll(() => selection(page), { message: `the palette's row ${index}` })
      .toEqual({ selected: [index], highlighted: true, whole: true });
    await expect(search).toBeFocused();
  };
  const indexes = Array.from({ length: count }, (_, index) => index);
  // down: each row, then the arrow to the next
  await indexes.reduce(async (previous, index) => {
    await previous;
    await at(index);
    if (index < count - 1) await search.press("ArrowDown");
  }, Promise.resolve());
  // back up: from the row before the last to the first
  await indexes
    .slice(0, -1)
    .toReversed()
    .reduce(async (previous, index) => {
      await previous;
      await search.press("ArrowUp");
      await at(index);
    }, Promise.resolve());
  // and down again by one, past the group's heading the list scrolled to for the first row
  await search.press("ArrowDown");
  await at(1);
}

test("G19 (acceptance): on Sprint's page the palette's rows and the rows of its menu of Web's cycles, each selected by the arrows down the list and back up, say they are selected, show it and are whole within the list's scrolling box, the search keeping the focus", async ({
  api,
  signedInPage,
}, testInfo) => {
  const w = await cycleListWorld(api, testInfo);
  // Web's cycles: Sprint in progress, with Plan, and fourteen drafts, more than the list shows at once.
  const sprint = await w.cycle("Sprint", { start_date: dayFromToday(-3), end_date: dayFromToday(10) });
  await Array.from({ length: 14 }, (_, index) => `Draft ${String(index + 1).padStart(2, "0")}`).reduce(
    async (previous, name) => {
      await previous;
      await w.cycle(name);
    },
    Promise.resolve()
  );
  await w.item("Plan", { cycle_id: sprint.id });

  const { page } = await openListPage(signedInPage, w.mem, w.path(sprint.id));
  await expect.poll(() => shownItems(page)).toEqual(["Plan"]);

  // The palette's own rows.
  await openCommands(page);
  await walkRows(page, true);

  // Its menu of Web's cycles: Sprint and the fourteen drafts.
  await selectCommand(page, /^Open a cycle/);
  await expect(page.getByRole("option", { name: "Draft 01", exact: true })).toBeVisible();
  expect(await rowCount(page)).toBe(15);
  await walkRows(page, false);
  await page.keyboard.press("Escape");
});
