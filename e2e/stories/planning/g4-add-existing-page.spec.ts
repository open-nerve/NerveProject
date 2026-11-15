import type { Locator, Page } from "@playwright/test";

import { byId } from "../../fixtures/assert/issue";
import {
  cycleJoined,
  cycleMoved,
  expectCycleRows,
  expectModuleRows,
  expectPlanningActivity,
  lastWrite,
  moduleJoined,
  wrote,
} from "../../fixtures/assert/issue-planning";
import { dayFromToday, failedLoads, recordApi, toast } from "../../fixtures/cycle-pages";
import { announced, modalDialog, pageBehindHidden } from "../../fixtures/dialogs";
import { ascendingIds } from "../../fixtures/issue-planning";
import { issueArchived, issueRead } from "../../fixtures/issues";
import { expectListPageQuiet, listGroup } from "../../fixtures/issue-list-page";
import { changeModulePreferences, modulePreferencesRead } from "../../fixtures/modules";
import {
  bodiesSent,
  cycleListWorld,
  moduleListWorld,
  openListPage,
  shownItems,
  weekdayThisMonth,
  workItemLayoutButton,
} from "../../fixtures/planning-list-pages";
import { answerTo } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// G4's and G11's page versions of "add existing" (M6 P7b spec 2.9–2.11, 2.14; R2, R5, R6, R11, R20, R21; M3 handoff 1,
// 3): a cycle's and a module's page adding work items of their project as mem does it in his browser. The dialog,
// named, keeps the page behind it inert, reaches its options by the arrow keys, has no link in them and names each
// chosen one's remove button by it; a cycle's page lists the work items not in its cycle, each one in another cycle
// with that cycle beside it, and sends each chosen one's joining alone, two at once at most, the dialog staying once the
// empty state it was opened from has gone; one nerve refuses stays chosen, said why; the cycle's progress is read once
// for them all; a search that finds only work items of its cycle says so. Its calendar's day adds one in one change of
// its day and its cycle. A module's group header adds one to the module alone, the module joined to those the work item
// holds, an archived one kept. The world is made through the API, nerve's answers read back through the database.

/** The names of the remove buttons of the work items the dialog has chosen, in the order they were chosen. */
function removeButtons(dialog: Locator): Promise<(string | null)[]> {
  return dialog
    .getByRole("button", { name: /^Remove / })
    .evaluateAll((buttons) => buttons.map((button) => button.getAttribute("aria-label")));
}

/**
 * The most of the requests to paths that page had out at once since its timings were cleared, by its own timings of
 * them (Resource Timing): each one out from its start to the end of its answer.
 */
function mostAtOnce(page: Page, paths: string[]): Promise<number> {
  return page.evaluate((wanted) => {
    const spans = performance
      .getEntriesByType("resource")
      .filter((entry) => wanted.includes(new URL(entry.name).pathname)) as PerformanceResourceTiming[];
    const outAt = (start: number) => spans.filter((span) => span.startTime <= start && start < span.responseEnd).length;
    return Math.max(0, ...spans.map((span) => outAt(span.startTime)));
  }, paths);
}

test("G4 (page): Sprint's empty page adds existing work items in its dialog, named, the page behind it inert: it searches nerve's most at once and lists X, Y, in Next, so captioned, Z and U, reached by the arrow keys; Z, Y and X chosen and Z archived behind the page, X and Y join Sprint, one change each and two out at once, Y moved out of Next, Z refused, said why and kept chosen, the dialog staying once the empty state has gone and the progress read once; a search of those in Sprint alone says so; the calendar's day adds U in one change of its day and Sprint", async ({
  api,
  browserName,
  db,
  signedInPage,
}, testInfo) => {
  const w = await cycleListWorld(api, testInfo);
  // Sprint is in progress, with no work items; Next is ahead, with Y; X, Z, done, and U are in no cycle.
  const sprint = await w.cycle("Sprint", { start_date: dayFromToday(-7), end_date: dayFromToday(7) });
  const next = await w.cycle("Next", { start_date: "2099-01-01", end_date: "2099-01-14" });
  const x = await w.item("X");
  const y = await w.item("Y", { cycle_id: next.id });
  const intoNext = (await lastWrite(db, y.id)).at;
  const z = await w.item("Z", { state_id: w.states.Done });
  const u = await w.item("U");

  const { page, watch } = await openListPage(signedInPage, w.mem, w.path(sprint.id));
  const requests = recordApi(page);
  const progressReads = () =>
    requests().requests.filter((request) => request === `GET /api/v0/cycles/${sprint.id}/progress`).length;
  const empty = page.getByText("No work items to show in this cycle", { exact: true });
  await expect(empty).toBeVisible();

  // The empty state's "add existing": its dialog, named, the page behind it inert, searches Web for nerve's most at
  // once and lists, the newest first, the work items not in Sprint, Y with its cycle beside it; no link in them.
  const search = `/api/v0/workspaces/${w.slug}/issue-search`;
  const searched = await answerTo(page, "GET", search, () =>
    page.getByRole("button", { name: "Add existing work item", exact: true }).click()
  );
  expect(Object.fromEntries(new URL(searched.url()).searchParams)).toEqual({ project_id: w.web.id, limit: "100" });
  const dialog = modalDialog(page);
  await expect(dialog).toHaveAccessibleName("Add existing work item");
  expect(await pageBehindHidden(page)).toBe(true);
  const options = dialog.getByRole("option");
  await expect(options).toHaveText(["WEB-4U", "WEB-3Z", "WEB-2Y, Cycle: Next", "WEB-1X"]);
  await expect(dialog.getByRole("option", { name: /^WEB-2 ?Y ?, Cycle: ?Next$/ })).toBeVisible();
  await expect(dialog.getByRole("listbox").locator("a")).toHaveCount(0);

  // From the search field, the arrow keys reach Z, Y and X, each chosen by Enter and its remove button named by it.
  const field = dialog.getByRole("combobox", { name: "Type to search", exact: true });
  await field.focus();
  const keys = ["ArrowDown", "ArrowDown", "Enter", "ArrowDown", "Enter", "ArrowDown", "Enter"];
  await keys.reduce(async (previous, key) => {
    await previous;
    await page.keyboard.press(key);
  }, Promise.resolve());
  await expect.poll(() => removeButtons(dialog)).toEqual(["Remove Z", "Remove Y", "Remove X"]);

  // Z archived behind the page, the three are sent: X and Y join Sprint, one change each, Z is refused, two out at
  // once. The dialog says why and keeps Z chosen; it stays once X's or Y's joining has taken the empty state away.
  expect((await issueArchived(api, w.token, z.id)).status, "Z archived").toBe(200);
  const sent = (id: string) => bodiesSent(page, browserName, "PATCH", `/api/v0/issues/${id}`);
  const [toX, toY, toZ] = [await sent(x.id), await sent(y.id), await sent(z.id)];
  const progressBefore = progressReads();
  await page.evaluate(() => performance.clearResourceTimings());
  await dialog.getByRole("button", { name: "Add selected work items", exact: true }).click();
  const refusal = toast(page, "The work item is archived. Restore it to change it.");
  await expect(refusal).toBeVisible();
  expect(await announced(refusal)).toBe(true);
  await expect.poll(() => removeButtons(dialog)).toEqual(["Remove Z"]);
  await expect.poll(() => shownItems(page)).toEqual(["X", "Y"]);
  await expect(empty).toHaveCount(0);
  await expect(dialog).toHaveAccessibleName("Add existing work item");
  expect([toX, toY, toZ]).toEqual([[{ cycle_id: sprint.id }], [{ cycle_id: sprint.id }], [{ cycle_id: sprint.id }]]);
  const changed = [x.id, y.id, z.id].map((id) => `/api/v0/issues/${id}`);
  expect(await mostAtOnce(page, changed), "the most out at once").toBe(2);
  await expect.poll(progressReads, "the progress read once for the three").toBe(progressBefore + 1);

  // Searched for Y, now Sprint's, the dialog says it is in this cycle, not that it found none.
  await answerTo(page, "GET", search, () => field.fill("Y"));
  await expect(dialog.getByText("Every work item found is already in this cycle.", { exact: true })).toBeVisible();
  await dialog.getByRole("button", { name: "Cancel", exact: true }).click();
  await expect(dialog).toHaveCount(0);

  // In the database X and Y are in Sprint, Y moved out of Next, a row of activity; Z is in no cycle.
  const [xJoined, yMoved] = [(await lastWrite(db, x.id)).at, (await lastWrite(db, y.id)).at];
  const mem = w.mem.email;
  await expectCycleRows(db, x.id, [{ target: "Sprint", added: xJoined, removed: null, adder: mem, by: mem }]);
  await expectCycleRows(db, y.id, [
    { target: "Next", added: intoNext, removed: yMoved, adder: w.admin.email, by: mem },
    { target: "Sprint", added: yMoved, removed: null, adder: mem, by: mem },
  ]);
  await expectPlanningActivity(db, y.id, [
    ...wrote(w.admin.email, intoNext, cycleJoined("Next")),
    ...wrote(mem, yMoved, cycleMoved("Next", "Sprint")),
  ]);
  await expectCycleRows(db, z.id, []);

  // The calendar: a day's "add existing" searches the work items of no day, nerve's most at once, and lists U alone,
  // X and Y being Sprint's; U chosen is one change of its day and its cycle, which nerve makes both.
  const calendar = await answerTo(page, "PATCH", `/api/v0/me/cycles/${sprint.id}/preferences`, () =>
    workItemLayoutButton(page, "Calendar").click()
  );
  expect(calendar.status()).toBe(200);
  const date = weekdayThisMonth();
  const day = page.locator(`button[aria-label="${date}"]`).locator("..");
  await day.getByRole("button", { name: "Add work item", exact: true }).click();
  const undated = await answerTo(page, "GET", search, () =>
    page.getByRole("menuitem", { name: "Add existing work item", exact: true }).click()
  );
  expect(Object.fromEntries(new URL(undated.url()).searchParams)).toEqual({
    project_id: w.web.id,
    target_date: "none",
    limit: "100",
  });
  await expect(dialog).toHaveAccessibleName("Add existing work item");
  await expect(options).toHaveText(["WEB-4U"]);
  const toU = await sent(u.id);
  await options.click();
  const added = await answerTo(page, "PATCH", `/api/v0/issues/${u.id}`, () =>
    dialog.getByRole("button", { name: "Add selected work items", exact: true }).click()
  );
  expect(added.status()).toBe(200);
  await expect(dialog).toHaveCount(0);
  expect(toU).toEqual([{ target_date: date, cycle_id: sprint.id }]);
  await expect(day.locator(`[id="issue-${u.id}"]`)).toBeVisible();
  const uJoined = (await lastWrite(db, u.id)).at;
  await expectCycleRows(db, u.id, [{ target: "Sprint", added: uJoined, removed: null, adder: mem, by: mem }]);
  expect(await issueRead(api, w.token, u.id)).toMatchObject({ status: 200, issue: { target_date: date } });
  await expect.poll(progressReads, "the progress read once more for U").toBe(progressBefore + 2);
  await expectListPageQuiet(page, watch, {
    failures: [`409 PATCH /api/v0/issues/${z.id}`],
    errors: failedLoads(browserName, 409),
  });
});

test("G11 (page): Alpha's page grouped by state adds existing work items from its group Todo's header, named by it: M, done, in Beta and in Gamma, archived, chosen joins Alpha in one change of its modules, Alpha added to those it holds, Gamma kept, and stays done", async ({
  api,
  browserName,
  db,
  signedInPage,
}, testInfo) => {
  const w = await moduleListWorld(api, testInfo);
  const [alpha, beta, gamma] = [await w.module("Alpha"), await w.module("Beta"), await w.module("Gamma")];
  // Plan is Alpha's; M, done, joins Beta and Gamma, which the admin then completes and archives.
  await w.item("Plan", { module_ids: [alpha.id] });
  const m = await w.item("M", { state_id: w.states.Done, module_ids: [beta.id, gamma.id] });
  const made = (await lastWrite(db, m.id)).at;
  await w.archive(gamma.id);
  // mem's page of Alpha grouped by state
  const mine = w.mem.tokens.access_token;
  const saved = await modulePreferencesRead(api, mine, alpha.id);
  if (!("preferences" in saved)) throw new Error(`Alpha's preferences: ${JSON.stringify(saved)}`);
  await changeModulePreferences(api, mine, alpha.id, {
    display_filters: { ...saved.preferences.display_filters, group_by: "state_id" },
  });

  const { page, watch } = await openListPage(signedInPage, w.mem, w.path(alpha.id));
  await expect.poll(() => shownItems(page)).toEqual(["Plan"]);

  // Todo's "add existing", named by its group, lists the work items not in Alpha: M, chosen, is one change of its
  // modules, Alpha added to Beta and Gamma, nothing of Todo's; it joins Alpha's Done group.
  const toM = await bodiesSent(page, browserName, "PATCH", `/api/v0/issues/${m.id}`);
  const searched = await answerTo(page, "GET", `/api/v0/workspaces/${w.slug}/issue-search`, () =>
    page.getByRole("button", { name: "Add existing work item: Todo", exact: true }).click()
  );
  expect(Object.fromEntries(new URL(searched.url()).searchParams)).toEqual({ project_id: w.web.id, limit: "100" });
  const dialog = modalDialog(page);
  await expect(dialog).toHaveAccessibleName("Add existing work item");
  await expect(dialog.getByRole("option")).toHaveText(["WEB-2M"]);
  await dialog.getByRole("option").click();
  const joined = await answerTo(page, "PATCH", `/api/v0/issues/${m.id}`, () =>
    dialog.getByRole("button", { name: "Add selected work items", exact: true }).click()
  );
  expect(joined.status()).toBe(200);
  await expect(dialog).toHaveCount(0);
  expect(toM).toEqual([{ module_ids: [...ascendingIds(beta, gamma), alpha.id] }]);
  await expect.poll(() => shownItems(page)).toEqual(["M", "Plan"]);
  await expect(listGroup(page, "Done").locator(`[id^="issue_${m.id}_"]`)).toHaveCount(1);

  // In the database M is in Alpha too, by mem, Gamma kept; a row of activity.
  const at = (await lastWrite(db, m.id)).at;
  const admin = w.admin.email;
  await expectModuleRows(db, m.id, [
    { target: "Beta", added: made, removed: null, adder: admin, by: admin },
    { target: "Gamma", added: made, removed: null, adder: admin, by: admin },
    { target: "Alpha", added: at, removed: null, adder: w.mem.email, by: w.mem.email },
  ]);
  await expectPlanningActivity(db, m.id, [
    ...wrote(admin, made, ...[beta, gamma].toSorted(byId).map((one) => moduleJoined(one.name))),
    ...wrote(w.mem.email, at, moduleJoined("Alpha")),
  ]);
  await expectListPageQuiet(page, watch);
});
