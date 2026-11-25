import type { Page } from "@playwright/test";

import { expectQuietConsole, watchPage } from "../../fixtures/browser";
import { recordApi } from "../../fixtures/cycle-pages";
import { modalDialog } from "../../fixtures/dialogs";
import { createIssue } from "../../fixtures/issues";
import { bodiesSent, listRow, shownItems, tableOrder, weekdayThisMonth } from "../../fixtures/planning-list-pages";
import { answerTo, moveWithinApp } from "../../fixtures/settings-pages";
import { stateIds } from "../../fixtures/states";
import { expect, test } from "../../fixtures/test";
import {
  expectQueryHolds,
  listReads,
  viewRead,
  viewReads,
  viewWorld,
  viewsCrumb,
  viewsPath,
} from "../../fixtures/view-pages";
import { savedQuery, viewDefaults } from "../../fixtures/views";

// H12's page version, a project's view's page in each layout (M7 P6a spec 2.5, 2.11; R1; re-96 I3): a view saved in the
// list, the board, the calendar and the table opens in its layout, each read once and the project's work items read
// under its words with that layout's own options; the board pages a column by nerve's cursor; a row's change on the
// view's page is that page's; and the ways the layouts add work items work on a view's page: a group's "+" and a
// calendar day's "add existing". The view's page itself is h12-project-view-page.spec.ts's.

/** Waits for the reads page has out to be answered (recordApi's), as a story does before it moves. */
async function settled(page: Page, recorded: () => { pending: number }): Promise<void> {
  await expect.poll(() => recorded().pending, "the reads out answered").toBe(0);
}

/** The display of a view of layout, grouped by groupBy (nerve's words), the rest nerve's defaults. */
function shown(layout: string, groupBy: string | null = null) {
  return { display_filters: { ...viewDefaults.display_filters, layout, group_by: groupBy } };
}

/** The work items whose blocks page lays out, each by its block's id, whether its content shows or waits for the view. */
function blocks(page: Page): Promise<number> {
  return page
    .locator('[id^="issue-"]')
    .evaluateAll((all) => all.filter((block) => /^issue-[0-9a-f-]{36}$/.test(block.id)).length);
}

/** Goes back to Web's views from a view's page by its crumb, once the page's reads are answered. */
async function back(page: Page, slug: string, projectId: string, last: string): Promise<void> {
  const recorded = recordApi(page);
  await viewsCrumb(page, slug, projectId).click();
  await expect(page.getByRole("link", { name: last, exact: true })).toBeVisible();
  await settled(page, recorded);
}

/**
 * Opens the view of name from Web's views list, which page shows, by its row's link, and checks that it read the view
 * once and the project's work items once, by a query that holds words; then goes back to the list.
 */
async function opened(
  page: Page,
  slug: string,
  projectId: string,
  view: { id: string; name: string },
  words: Parameters<typeof expectQueryHolds>[1]
): Promise<void> {
  const recorded = recordApi(page);
  await page.getByRole("link", { name: view.name, exact: true }).click();
  await expect.poll(() => listReads(recorded().requests, projectId).length).toBe(1);
  await settled(page, recorded);
  expect(viewReads(recorded().requests), `the reads of ${view.name}`).toEqual([viewRead(view.id)]);
  const reads = listReads(recorded().requests, projectId);
  expect(reads, `the list reads of ${view.name}`).toHaveLength(1);
  expectQueryHolds(reads[0] ?? new URLSearchParams(), words);
  await back(page, slug, projectId, view.name);
}

test("H12 (page): alice's views of Web in the list, the board, the calendar and the table each open in their layout, the view read once and Web's work items once under its words with the layout's own options, listing Due and Plain; her change of Plain's priority on the list's page is that page's", async ({
  api,
  signedInPage,
}, testInfo) => {
  const w = await viewWorld(api, testInfo);
  const states = await stateIds(api, w.alice.token, w.web.id);
  const day = weekdayThisMonth();
  const due = await createIssue(api, w.alice.token, w.web.id, {
    name: "Due",
    priority: "urgent",
    state_id: states.Todo,
    target_date: day,
  });
  const plain = await createIssue(api, w.alice.token, w.web.id, { name: "Plain", priority: "low" });
  const list = await w.view(w.alice, { name: "As list", filters: { priority: ["urgent", "low"] }, ...shown("list") });
  const board = await w.view(w.alice, { name: "As board", ...shown("kanban", "state_id") });
  const calendar = await w.view(w.alice, { name: "As calendar", ...shown("calendar") });
  const table = await w.view(w.alice, { name: "As table", ...shown("spreadsheet") });
  const { order_by, sub_issue } = savedQuery(table);

  const page = await signedInPage(w.alice.tokens);
  const watch = await watchPage(page);
  await page.goto(viewsPath(w.slug, w.web.id));
  await expect(page.getByRole("link", { name: "As table", exact: true })).toBeVisible();

  // The list: its words, a hundred at once; the board: by state, thirty a column; the calendar: by the days of its
  // month, four a day; the table: a hundred at once.
  await opened(page, w.slug, w.web.id, list, { ...savedQuery(list), limit: "100" });
  await opened(page, w.slug, w.web.id, board, { order_by, sub_issue, group_by: "state_id", limit: "30" });
  await opened(page, w.slug, w.web.id, calendar, { order_by, sub_issue, group_by: "target_date", limit: "4" });
  await opened(page, w.slug, w.web.id, table, { order_by, sub_issue, limit: "100" });

  // Each again, by what it lays out: the board's cards in their states, the calendar's day, the table's rows.
  const layout = async (name: string, check: () => Promise<void>) => {
    await page.getByRole("link", { name, exact: true }).click();
    await check();
    await back(page, w.slug, w.web.id, name);
  };
  await layout("As board", async () => {
    await expect.poll(() => shownItems(page)).toEqual(["Due", "Plain"]);
    await expect(page.getByRole("button", { name: "Todo", exact: true })).toBeVisible();
  });
  await layout("As calendar", async () => {
    await expect(
      page.locator(`button[aria-label="${day}"]`).locator("..").locator(`[id="issue-${due.id}"]`)
    ).toBeVisible();
  });
  await layout("As table", async () => {
    await expect.poll(() => tableOrder(page)).toEqual([plain.id, due.id]);
  });

  // The list's page: Plain's priority made urgent there, one change, shown on its row.
  await page.getByRole("link", { name: "As list", exact: true }).click();
  await expect.poll(() => shownItems(page)).toEqual(["Due", "Plain"]);
  await listRow(page, plain.id).getByRole("button", { name: "Priority: low", exact: true }).click();
  const changed = await answerTo(page, "PATCH", `/api/v0/issues/${plain.id}`, () =>
    page.getByRole("option", { name: "Urgent", exact: true }).click()
  );
  expect(changed.status()).toBe(200);
  await expect(listRow(page, plain.id).getByRole("button", { name: "Priority: urgent", exact: true })).toBeVisible();

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});

test("H12 (page): alice's board of Web by state lists thirty of Backlog's 31 and, by its Load more, the last by nerve's cursor; on her list of Todo by state, Todo's + opens a new work item of Todo, which joins the view's list; on her calendar, a day's Add existing work item searches those of no day and gives Plain that day", async ({
  api,
  browserName,
  signedInPage,
}, testInfo) => {
  const w = await viewWorld(api, testInfo);
  const states = await stateIds(api, w.alice.token, w.web.id);
  const day = weekdayThisMonth();
  // Item 01 to Item 31, one after another, in Backlog; the newest first.
  await Array.from({ length: 31 }, (_, index) => `Item ${String(index + 1).padStart(2, "0")}`).reduce(
    async (previous, name) => {
      await previous;
      await createIssue(api, w.alice.token, w.web.id, { name, state_id: states.Backlog });
    },
    Promise.resolve()
  );
  const plain = await createIssue(api, w.alice.token, w.web.id, { name: "Plain", state_id: states.Todo });
  const board = await w.view(w.alice, { name: "Board", ...shown("kanban", "state_id") });
  const byState = await w.view(w.alice, {
    name: "By state",
    filters: { state_id: [states.Todo ?? ""] },
    ...shown("list", "state_id"),
  });
  const calendar = await w.view(w.alice, { name: "Calendar", ...shown("calendar") });

  const page = await signedInPage(w.alice.tokens);
  const watch = await watchPage(page);
  const recorded = recordApi(page);
  await page.goto(`${viewsPath(w.slug, w.web.id)}/${board.id}`);
  // Backlog's first thirty, then the last by Load more, read by nerve's cursor.
  await expect.poll(() => blocks(page)).toBe(31);
  // (by the keyboard: the column's quick add, as sticky as the button at its foot, lies over it)
  const more = page.getByRole("button", { name: "Load more ↓", exact: true });
  await more.focus();
  await page.keyboard.press("Enter");
  await expect.poll(() => blocks(page)).toBe(32);
  await expect(more).toHaveCount(0);
  const paged = listReads(recorded().requests, w.web.id).filter((query) => query.has("cursor"));
  expect(paged).toHaveLength(1);
  expectQueryHolds(paged[0] ?? new URLSearchParams(), {
    group_by: "state_id",
    group_key: states.Backlog ?? "",
    limit: "30",
  });

  // The list by state: Todo's +, a new work item of Todo, which the view's list then shows.
  await settled(page, recorded);
  await moveWithinApp(page, `${viewsPath(w.slug, w.web.id)}/${byState.id}`);
  await expect.poll(() => shownItems(page)).toEqual(["Plain"]);
  const created = await bodiesSent(page, browserName, "POST", `/api/v0/projects/${w.web.id}/issues`);
  await page.getByRole("button", { name: "Create work item: Todo", exact: true }).click();
  const form = page.getByRole("dialog", { name: "Create new work item", exact: true });
  await expect(form.getByRole("button", { name: "State: Todo", exact: true })).toBeVisible();
  await form.getByRole("textbox", { name: "Title", exact: true }).fill("Made in Todo");
  await form.getByRole("button", { name: "Save", exact: true }).click();
  await expect(form).toHaveCount(0);
  await expect.poll(() => shownItems(page)).toEqual(["Made in Todo", "Plain"]);
  expect(created).toMatchObject([{ name: "Made in Todo", state_id: states.Todo }]);

  // The calendar: a day's Add existing work item searches Web's work items of no day, and Plain chosen takes that day.
  await settled(page, recorded);
  await moveWithinApp(page, `${viewsPath(w.slug, w.web.id)}/${calendar.id}`);
  const cell = page.locator(`button[aria-label="${day}"]`).locator("..");
  await cell.getByRole("button", { name: "Add work item", exact: true }).click();
  const searched = await answerTo(page, "GET", `/api/v0/workspaces/${w.slug}/issue-search`, () =>
    page.getByRole("menuitem", { name: "Add existing work item", exact: true }).click()
  );
  expect(Object.fromEntries(new URL(searched.url()).searchParams)).toEqual({
    project_id: w.web.id,
    target_date: "none",
  });
  const dialog = modalDialog(page);
  await expect(dialog).toHaveAccessibleName("Add existing work item");
  const toPlain = await bodiesSent(page, browserName, "PATCH", `/api/v0/issues/${plain.id}`);
  await dialog.getByRole("option", { name: /Plain$/ }).click();
  await answerTo(page, "PATCH", `/api/v0/issues/${plain.id}`, () =>
    dialog.getByRole("button", { name: "Add selected work items", exact: true }).click()
  );
  await expect(dialog).toHaveCount(0);
  expect(toPlain).toEqual([{ target_date: day }]);
  await expect(cell.locator(`[id="issue-${plain.id}"]`)).toBeVisible();

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});
