import type { Locator, Page } from "@playwright/test";

import { changeProjectPreferences, readProjectPreferences } from "../../fixtures/api";
import {
  cycleJoined,
  expectCycleRows,
  expectPlanningActivity,
  lastWrite,
  wrote,
} from "../../fixtures/assert/issue-planning";
import { watchPage } from "../../fixtures/browser";
import { dayFromToday, dismissToasts, recordApi, toast } from "../../fixtures/cycle-pages";
import { cycleCalls } from "../../fixtures/cycles";
import { addCondition, displayPanel } from "../../fixtures/issue-filters-page";
import { expectListPageQuiet } from "../../fixtures/issue-list-page";
import { turnModules } from "../../fixtures/module-pages";
import { createModule, moduleCalls } from "../../fixtures/modules";
import {
  bodiesSent,
  cycleListWorld,
  listQueries,
  shownItems,
  tableOrder,
  workItemLayoutButton,
} from "../../fixtures/planning-list-pages";
import { ids, idsDown } from "../../fixtures/planning-lists";
import { dragOnto } from "../../fixtures/project-pages";
import { answerTo } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// G15's page version, the work items' side (M6 P7b spec 2.3–2.8, 2.14; R3, R4, R7, R9, R17): Web's list as mem uses it
// in his browser, by its cycles and its modules. Its filter by cycle offers No cycle and the cycles that are not
// archived, and No cycle lists the work items in none; grouped by cycle its groups follow nerve's order, the archived
// one named once Web's archive is read, None last; grouped by module a work item shows in each of its modules, an
// archived one too; on the board a work item dragged out of None into a cycle is one change, and one dragged into an
// ended cycle is none; the table orders by the cycles' and the modules' names both ways. The world is made through the
// API, nerve's answers read back through the database.

/** The names of the groups of a list, in their order: each group's header button is named by its group. */
function listGroupNames(page: Page): Promise<(string | null)[]> {
  return page
    .locator('[class~="group/list-header"] > button[aria-expanded]')
    .evaluateAll((buttons) => buttons.map((button) => button.getAttribute("aria-label")));
}

/** The names of the work items whose rows within shows, sorted. */
function rowsWithin(within: Locator): Promise<string[]> {
  return within
    .locator('button[aria-label^="Options: "]')
    .evaluateAll((buttons) => buttons.map((button) => button.getAttribute("aria-label")?.slice(9) ?? "").toSorted());
}

test("G15 (page): Web's list grouped by module shows W1 in Alpha and in Gamma, archived, with W2; grouped by cycle its groups are Sprint, Next, Ended and Old, archived, Web's archive read once, then None; its cycle filter offers No cycle, Sprint, Next and Ended, and No cycle lists W3 alone; on the board W3 dragged out of None into Sprint is one change of its cycle and its order, the counts following nerve, and W1 dragged into Ended sends nothing and says why; the table orders by the cycles' names both ways and by the modules' names down, a work item by its first module", async ({
  api,
  browserName,
  db,
  signedInPage,
}, testInfo) => {
  const w = await cycleListWorld(api, testInfo);
  const { token, web } = w;
  await turnModules(api, token, web, true);
  // nerve lists a project's cycles and modules the newest first: Sprint, Next, Ended; Alpha, Beta. Old and Ended are
  // drafts their work items join before they end, Old then archived; Gamma is completed and archived once joined.
  const old = await w.cycle("Old");
  const ended = await w.cycle("Ended");
  await w.cycle("Next", { start_date: "2099-01-01", end_date: "2099-01-14" });
  const sprint = await w.cycle("Sprint", { start_date: dayFromToday(-7), end_date: dayFromToday(7) });
  const module = (name: string) => createModule(api, token, web.id, { name });
  const gamma = await module("Gamma");
  const beta = await module("Beta");
  const alpha = await module("Alpha");
  const w1 = await w.item("W1", { cycle_id: sprint.id, module_ids: [alpha.id, gamma.id] });
  const w2 = await w.item("W2", { cycle_id: old.id, module_ids: [gamma.id] });
  const w3 = await w.item("W3");
  const w4 = await w.item("W4", { cycle_id: ended.id, module_ids: [beta.id] });
  await w.end(old.id, true);
  expect(
    [
      await cycleCalls(api).update(token, ended.id, { start_date: "2025-02-01", end_date: "2025-02-14" }),
      await moduleCalls(api).update(token, gamma.id, { status: "completed" }),
      await moduleCalls(api).archive(token, gamma.id),
    ],
    "Ended ends in February 2025; Gamma completed and archived"
  ).toEqual(Array(3).fill({ status: 200 }));
  // mem's list of Web: grouped by module, in the order of its places (a drag's), its rows showing their cycle and modules
  const mine = w.mem.tokens.access_token;
  const saved = await readProjectPreferences(api, mine, web.id);
  expect(
    (
      await changeProjectPreferences(api, mine, web.id, {
        display_filters: {
          ...saved.display_filters,
          layout: "list",
          group_by: "module_id",
          sub_group_by: null,
          order_by: "sort_order",
          show_empty_groups: true,
        },
        display_properties: { ...saved.display_properties, cycle: true, modules: true },
      })
    ).status,
    "mem's list of Web"
  ).toBe(200);
  const preferences = `/api/v0/me/projects/${web.id}/preferences`;

  const page = await signedInPage(w.mem.tokens);
  const watch = await watchPage(page);
  const requests = recordApi(page);
  const queries = listQueries(page, web.id);
  await page.goto(w.projectPath);

  // Grouped by module: W1 in Alpha and in Gamma, which is named and captioned archived, with W2.
  await expect.poll(() => listGroupNames(page)).toEqual(["Alpha", "Beta", "Gamma, Archived", "None"]);
  const group = (name: string) => page.getByRole("button", { name, exact: true }).locator("xpath=../../..");
  await expect.poll(() => rowsWithin(group("Alpha"))).toEqual(["W1"]);
  await expect.poll(() => rowsWithin(group("Gamma, Archived"))).toEqual(["W1", "W2"]);

  // Grouped by cycle: nerve's order of Web's cycles, Old archived after them, None last; Web's archive read once.
  const groupBy = (await displayPanel(page)).getByRole("button", { name: "Group by", exact: true });
  if ((await groupBy.getAttribute("aria-expanded")) !== "true") await groupBy.click();
  const byCycle = await answerTo(page, "PATCH", preferences, () =>
    groupBy.locator("../..").getByRole("button", { name: "Cycle", exact: true }).click()
  );
  expect(byCycle.status()).toBe(200);
  await page.keyboard.press("Escape");
  await expect.poll(() => listGroupNames(page)).toEqual(["Sprint", "Next", "Ended", "Old, Archived", "None"]);
  expect(
    requests().requests.filter((request) => request === `GET /api/v0/projects/${web.id}/cycles?archived=true`),
    "the reads of Web's archive of cycles"
  ).toHaveLength(1);

  // The cycle filter offers No cycle and Web's cycles but the archived Old; No cycle lists W3 alone, and taken out
  // again, all four.
  const chip = await addCondition(page, "Cycle");
  await expect(page.getByRole("option")).toHaveText(["No cycle", "Sprint", "Next", "Ended"]);
  const none = await answerTo(page, "PATCH", preferences, () =>
    page.getByRole("option", { name: "No cycle", exact: true }).click()
  );
  expect(none.status()).toBe(200);
  await page.keyboard.press("Escape");
  await expect.poll(() => queries.some((query) => query.get("cycle_id") === "none")).toBe(true);
  await expect.poll(() => shownItems(page)).toEqual(["W3"]);
  const all = await answerTo(page, "PATCH", preferences, () =>
    chip.getByRole("button", { name: "Remove filter", exact: true }).click()
  );
  expect(all.status()).toBe(200);
  await expect.poll(() => shownItems(page)).toEqual(["W1", "W2", "W3", "W4"]);

  // The board by cycle: W3 dragged out of None into Sprint is one change, of its cycle and its order; the columns'
  // counts follow nerve's answer, which has the row and the activity of W3's joining Sprint.
  const board = await answerTo(page, "PATCH", preferences, () => workItemLayoutButton(page, "Board").click());
  expect(board.status()).toBe(200);
  const column = (key: string) => page.locator(`[id^="${key}__"]`);
  const card = (id: string) => page.locator(`[id^="issue_${id}_"]`);
  const count = (name: string) => page.getByRole("button", { name, exact: true }).locator("..");
  await expect(column(sprint.id).filter({ has: card(w1.id) })).toHaveCount(1);
  await expect(column("none").filter({ has: card(w3.id) })).toHaveCount(1);
  await expect(count("Sprint")).toHaveText("Sprint1");
  await expect(count("None")).toHaveText("None1");
  const changes = await bodiesSent(page, browserName, "PATCH", `/api/v0/issues/${w3.id}`);
  const moved = await answerTo(page, "PATCH", `/api/v0/issues/${w3.id}`, () =>
    dragOnto(page, card(w3.id), card(w3.id), column(sprint.id))
  );
  expect(moved.status()).toBe(200);
  expect(changes).toEqual([{ cycle_id: sprint.id, sort_order: expect.any(Number) }]);
  await expect(count("Sprint")).toHaveText("Sprint2");
  await expect(count("None")).toHaveText("None0");
  const joined = (await lastWrite(db, w3.id)).at;
  await expectCycleRows(db, w3.id, [
    { target: "Sprint", added: joined, removed: null, adder: w.mem.email, by: w.mem.email },
  ]);
  await expectPlanningActivity(db, w3.id, wrote(w.mem.email, joined, cycleJoined("Sprint")));

  // W1 dragged into Ended, which has ended: nothing is sent, and the page says nerve's sentence.
  await dragOnto(page, card(w1.id), card(w1.id), column(ended.id));
  await expect(
    toast(page, "The cycle is completed: work items cannot join it, and only its order can change.")
  ).toBeVisible();
  expect(
    requests().requests.filter((request) => request === `PATCH /api/v0/issues/${w1.id}`),
    "W1's changes"
  ).toEqual([]);
  await expect(column(sprint.id).filter({ has: card(w1.id) })).toHaveCount(1);
  await dismissToasts(page);

  // The table: by the cycles' names up and down, W1 and W3, both Sprint's now, by their ids the same way; by the
  // modules' names down, W1 by Alpha, the first of its two, Gamma archived named too, and W3, of none, last.
  const table = await answerTo(page, "PATCH", preferences, () => workItemLayoutButton(page, "Table").click());
  expect(table.status()).toBe(200);
  const header = (name: string) => page.getByRole("table").getByRole("columnheader", { name, exact: true });
  const sortBy = async (name: string, item: string, order: string) => {
    const from = queries.length;
    await header(name).getByRole("button").click();
    await page.getByRole("menuitem", { name: item, exact: true }).click();
    await expect.poll(() => queries.slice(from).some((query) => query.get("order_by") === order)).toBe(true);
  };
  await sortBy("Cycle", "A Z", "cycle_name");
  await expect(header("Cycle")).toHaveAttribute("aria-sort", "ascending");
  await expect.poll(() => tableOrder(page)).toEqual([w4.id, w2.id, ...ids(w1, w3)]);
  await sortBy("Cycle", "Z A", "-cycle_name");
  await expect(header("Cycle")).toHaveAttribute("aria-sort", "descending");
  await expect.poll(() => tableOrder(page)).toEqual([...idsDown(w1, w3), w2.id, w4.id]);
  await sortBy("Modules", "Z A", "-module_name");
  await expect(header("Modules")).toHaveAttribute("aria-sort", "descending");
  await expect(header("Cycle")).toHaveAttribute("aria-sort", "none");
  await expect.poll(() => tableOrder(page)).toEqual([w2.id, w4.id, w1.id, w3.id]);
  await expectListPageQuiet(page, watch);
});
