import type { Page } from "@playwright/test";

import { createProject, createWorkspace, slugFor } from "../../fixtures/api";
import { expectCycles, type CycleRow } from "../../fixtures/assert/planning";
import { emailFor } from "../../fixtures/auth";
import { expectQuietConsole, watchPage } from "../../fixtures/browser";
import {
  chooseFromMenu,
  cycleMenu,
  cyclePagesWorld,
  cyclePath,
  cycleSection,
  cyclesPath,
  dismissToasts,
  entriesOf,
  failedLoads,
  linkedCycles,
  menuEntries,
  openCyclesList,
  pickDays,
  recordApi,
  rowMenusKept,
  sectionToggle,
  seenBy,
  shownDay,
  shownDays,
  toast,
  turnCycles,
} from "../../fixtures/cycle-pages";
import { createCycle, cycleCalls } from "../../fixtures/cycles";
import {
  announced,
  closedByEscape,
  closedWithin,
  enabledWithin,
  modalDialog,
  shownWithin,
} from "../../fixtures/dialogs";
import { APP, PROJECT, PROJECT_CYCLES, PROJECT_MEMBER, WORKSPACE, valued } from "../../fixtures/mounts";
import { registerOnboarded, sentHeld, sentTo } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// G3's page version (M6 P5a spec 2.5, 2.6, 2.9, R8, R12): a project's cycles changed by their dialog and by a cycle's
// sidebar, and deleted, by its member in his browser, a row's menu opened by its button and by a right-click (M3
// handoff 3); and the page of a cycle nerve does not have, which S2 covered
// until the cycle pages read nerve's API (P5a T9). The API version is g3-update-delete-cycle.spec.ts; both assert the
// same rows.

/** rows with the cycle of name changed as to says. */
function changed(rows: CycleRow[], name: string, to: Partial<CycleRow>): CycleRow[] {
  return rows.map((c) => (c.name === name ? { ...c, ...to } : c));
}

/** A cycle's row as the stories create it: undeleted, not archived, without a description, made by owner. */
function made(name: string, start: string, end: string, sort_order: number, owner: string): CycleRow {
  return { name, description: "", start, end, sort_order, archived: false, deleted: false, owner, by: owner };
}

test("G3 (page): a right-click on Beta's row opens its menu, inert while closed, whose Edit opens Beta's dialog; mem renames Alpha Sprint A and moves its end by its dialog, which sends what changed and waits for nerve; on Sprint A's page its days moved onto Beta's first are refused and shown as nerve has them, and moved again shown once nerve has; Done, completed, offers no change and its days are disabled; its creator mem and the admin may delete Sprint A, nia, another member, may not, nor Web's guest or a workspace admin who is its guest change it; a deletion refused leaves its dialog open, one under way cannot be dismissed, and Sprint A is deleted", async ({
  api,
  baseURL,
  browser,
  browserName,
  db,
  signedInPage,
}, testInfo) => {
  // mem's page, then four other browsers, each signed in anew
  test.setTimeout(60_000);
  const { admin, mem, nia, gus, wanda, slug, web, token } = await cyclePagesWorld(api, testInfo);
  const path = cyclesPath(slug, web.id);
  // Alpha, mem's, and Beta, the admin's, in January and February 2031; Done, the admin's, ended in 2025.
  const alpha = await createCycle(api, mem.tokens.access_token, web.id, {
    name: "Alpha",
    start_date: "2031-01-01",
    end_date: "2031-01-14",
  });
  const beta = await createCycle(api, token, web.id, {
    name: "Beta",
    start_date: "2031-02-01",
    end_date: "2031-02-14",
  });
  const done = await createCycle(api, token, web.id, {
    name: "Done",
    start_date: "2025-01-01",
    end_date: "2025-01-14",
  });
  let rows = [
    made("Alpha", "2031-01-01", "2031-01-14", 65535, mem.email),
    made("Beta", "2031-02-01", "2031-02-14", 55535, admin.email),
    made("Done", "2025-01-01", "2025-01-14", 45535, admin.email),
  ];
  await expectCycles(db, web.id, rows);

  // mem's menus: Alpha, his, offers its change and its deletion; Beta, the admin's, its change alone; Done, completed,
  // neither, only its archive. None may be archived but Done.
  const page = await signedInPage(mem.tokens);
  const watch = await watchPage(page);
  await page.goto(path);
  await expect.poll(() => linkedCycles(cycleSection(page, "Upcoming cycle"))).toEqual([alpha.id, beta.id]);
  await sectionToggle(page, "Completed cycle").click();
  const upcomingEntries = ["Edit", "Open in new tab", "Copy link", "Archive (disabled)"];
  expect([await menuEntries(page, "Alpha"), await menuEntries(page, "Beta"), await menuEntries(page, "Done")]).toEqual([
    [...upcomingEntries, "Delete"],
    upcomingEntries,
    ["Open in new tab", "Copy link", "Archive"],
  ]);

  // Each row's right-click menu stays in the page while closed, inert: none is shown to assistive technology, and none
  // of its entries takes the focus (M3 handoff 3, ruling F-14). A right-click on Beta's row opens its menu, the same
  // entries as its button's, the first focused; Escape closes it, the focus back on the row's link, and it is inert
  // again. Its Edit opens Beta's dialog, named for a change and holding Beta, which Escape closes.
  await expect(page.getByRole("menu")).toHaveCount(0);
  expect(await rowMenusKept(page)).toEqual({ entries: 12, focusable: 0 });
  const betaLink = page.locator(`a[href="${cyclePath(slug, web.id, beta.id)}"]`);
  await betaLink.click({ button: "right" });
  const rowMenu = page.getByRole("menu");
  expect(await entriesOf(rowMenu)).toEqual(upcomingEntries);
  await expect(rowMenu.getByRole("menuitem", { name: "Edit", exact: true })).toBeFocused();
  await page.keyboard.press("Escape");
  await expect(rowMenu).toHaveCount(0);
  await expect(betaLink).toBeFocused();
  expect(await rowMenusKept(page)).toEqual({ entries: 12, focusable: 0 });
  await betaLink.click({ button: "right" });
  await rowMenu.getByRole("menuitem", { name: "Edit", exact: true }).click();
  await expect(rowMenu).toHaveCount(0);
  const dialog = modalDialog(page);
  await expect(dialog).toHaveAccessibleName("Update cycle");
  const name = dialog.getByRole("textbox", { name: "Title", exact: true });
  await expect(name).toHaveValue("Beta");
  expect(await closedByEscape(page)).toBe(true);

  // Alpha's dialog, named for a change, holds Alpha; renamed Sprint A and its end moved to the 20th, it sends those
  // alone, the days as a pair, and waits for nerve; it closes once nerve has changed it, and says so.
  await chooseFromMenu(page, "Alpha", "Edit");
  await expect(dialog).toHaveAccessibleName("Update cycle");
  await expect(name).toHaveValue("Alpha");
  const formDays = dialog.getByRole("button", { name: "Date range", exact: true });
  await expect(formDays).toContainText(shownDay("2031-01-14"));
  await name.fill("Sprint A");
  await pickDays(page, formDays, ["2031-01-20"]);
  await name.click();
  const alphaPath = `/api/v0/cycles/${alpha.id}`;
  const update = dialog.getByRole("button", { name: "Update cycle", exact: true });
  const renamed = await sentHeld(page, "PATCH", alphaPath, () => update.click());
  expect(renamed.body).toEqual({ name: "Sprint A", start_date: "2031-01-01", end_date: "2031-01-20" });
  expect(await closedByEscape(page)).toBe(false);
  expect(await enabledWithin(dialog.getByRole("button", { name: "Cancel", exact: true }))).toBe(false);
  expect((await renamed.release()).status()).toBe(200);
  expect(await closedWithin(page)).toBe(true);
  await expect(toast(page, "Cycle updated successfully.")).toBeVisible();
  await expect(cycleMenu(page, "Sprint A")).toBeVisible();
  rows = changed(rows, "Alpha", { name: "Sprint A", end: "2031-01-20" });
  await expectCycles(db, web.id, rows);

  // The others' menus, each in a browser of his own: the admin may delete Sprint A, mem's, and Beta, his; nia, another
  // member, changes them and deletes neither; Web's guest, and a workspace admin who is its guest, change none.
  const menusOf = (email: string, want: string[][]) =>
    seenBy(browser, baseURL ?? "", api, email, path, async (other) => {
      await expect.poll(() => linkedCycles(cycleSection(other, "Upcoming cycle"))).toEqual([alpha.id, beta.id]);
      expect([await menuEntries(other, "Sprint A"), await menuEntries(other, "Beta")], email).toEqual(want);
    });
  const reads = ["Open in new tab", "Copy link"];
  await menusOf(admin.email, [
    [...upcomingEntries, "Delete"],
    [...upcomingEntries, "Delete"],
  ]);
  await menusOf(nia.email, [upcomingEntries, upcomingEntries]);
  await menusOf(gus.email, [reads, reads]);
  await menusOf(wanda.email, [reads, reads]);

  // On Sprint A's page, its sidebar's days moved onto Beta's first day, the 1st of February: nerve refuses the
  // overlap, said, and the days are nerve's again; nothing is written.
  await dismissToasts(page);
  await page.getByRole("link").filter({ hasText: "Sprint A" }).first().click();
  await expect(page).toHaveURL(cyclePath(slug, web.id, alpha.id));
  await expect(page.getByRole("heading", { name: "Sprint A", exact: true })).toBeVisible();
  const days = page.getByRole("button", { name: "Date range", exact: true });
  await expect(days).toHaveText(shownDays("2031-01-01", "2031-01-20"));
  const overlap = await sentTo(page, "PATCH", alphaPath, () => pickDays(page, days, ["2031-02-01"]));
  expect([overlap.answer.status(), overlap.body]).toEqual([409, { start_date: "2031-01-01", end_date: "2031-02-01" }]);
  await expect(toast(page, "These dates overlap another cycle of this project.")).toBeVisible();
  await expect(days).toHaveText(shownDays("2031-01-01", "2031-01-20"));
  expect(await shownWithin(toast(page, "Cycle updated successfully."))).toBe(false);
  await expectCycles(db, web.id, rows);

  // Moved to end on the 25th, the days picked show while nerve answers and no other may be picked; only once nerve has
  // changed them does the sidebar say so.
  await dismissToasts(page);
  const moved = await sentHeld(page, "PATCH", alphaPath, () => pickDays(page, days, ["2031-01-25"]));
  expect(moved.body).toEqual({ start_date: "2031-01-01", end_date: "2031-01-25" });
  await expect(days).toBeDisabled();
  await expect(days).toHaveText(shownDays("2031-01-01", "2031-01-25"));
  expect(await shownWithin(toast(page, "Cycle updated successfully."))).toBe(false);
  expect((await moved.release()).status()).toBe(200);
  await expect(toast(page, "Cycle updated successfully.")).toBeVisible();
  await expect(days).toBeEnabled();
  await expect(days).toHaveText(shownDays("2031-01-01", "2031-01-25"));
  rows = changed(rows, "Sprint A", { end: "2031-01-25" });
  await expectCycles(db, web.id, rows);

  // Done's page, from the list: its days, ended, may not be changed, and its menu offers no change.
  await openCyclesList(page, slug, web.id);
  await sectionToggle(page, "Completed cycle").click();
  await page.locator(`a[href="${cyclePath(slug, web.id, done.id)}"]`).click();
  await expect(page.getByRole("heading", { name: "Done", exact: true })).toBeVisible();
  await expect(days).toBeDisabled();
  expect(await menuEntries(page, "Done")).toEqual(["Open in new tab", "Copy link", "Archive"]);

  // Sprint A's deletion asked for, Web's cycles turned off meanwhile: nerve refuses it, said beside the dialog, which
  // stays; Cancel closes it. Turned on again, the deletion waits for nerve, Escape not closing its dialog nor Cancel
  // enabled; once nerve has deleted it, the dialog closes, says so, and Sprint A is listed no more.
  await openCyclesList(page, slug, web.id);
  await chooseFromMenu(page, "Sprint A", "Delete");
  await expect(dialog).toHaveAccessibleName("Delete cycle");
  await expect(dialog).toContainText("Sprint A");
  await turnCycles(api, token, web, false);
  const confirm = dialog.getByRole("button", { name: "Delete", exact: true });
  const off = await sentTo(page, "DELETE", alphaPath, () => confirm.click());
  expect(off.answer.status()).toBe(409);
  const refusal = toast(page, "Cycles are turned off in this project. Turn them on to change them.");
  await expect(refusal).toBeVisible();
  expect(await announced(refusal)).toBe(true);
  expect(await closedWithin(page)).toBe(false);
  await expectCycles(db, web.id, rows);
  await dialog.getByRole("button", { name: "Cancel", exact: true }).click();
  expect(await closedWithin(page)).toBe(true);
  await turnCycles(api, token, web, true);
  await chooseFromMenu(page, "Sprint A", "Delete");
  const deletion = await sentHeld(page, "DELETE", alphaPath, () => confirm.click());
  expect(await closedByEscape(page)).toBe(false);
  expect(await enabledWithin(dialog.getByRole("button", { name: "Cancel", exact: true }))).toBe(false);
  expect((await deletion.release()).status()).toBe(204);
  expect(await closedWithin(page)).toBe(true);
  await expect(toast(page, "Cycle deleted successfully.")).toBeVisible();
  await expect.poll(() => linkedCycles(cycleSection(page, "Upcoming cycle"))).toEqual([beta.id]);
  rows = changed(rows, "Sprint A", { deleted: true });
  await expectCycles(db, web.id, rows);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([
    [`409 PATCH ${alphaPath}`, `409 DELETE ${alphaPath}`],
    [],
    [],
  ]);
  await expectQuietConsole(page, watch, { errors: failedLoads(browserName, 409, 409) });
});

test("G3 (page): the address of Gone, a cycle of Web deleted, says that the cycle does not exist and leads to Web's cycles; the page asks nerve's API alone, its reads of the cycle and of its progress not found, and logs nothing else", async ({
  api,
  browserName,
  signedInPage,
}, testInfo) => {
  const tokens = await registerOnboarded(api, emailFor(testInfo, "admin"));
  const token = tokens.access_token;
  const slug = slugFor(testInfo);
  await createWorkspace(api, token, { name: "Acme", slug });
  const web = await createProject(api, token, slug, { name: "Web", identifier: "WEB" });
  await turnCycles(api, token, web, true);
  const gone = await createCycle(api, token, web.id, { name: "Gone" });
  expect(await cycleCalls(api).remove(token, gone.id), "Gone deleted").toEqual({ status: 204 });

  const page: Page = await signedInPage(tokens);
  const watch = await watchPage(page);
  const recorded = recordApi(page);
  const path = cyclePath(slug, web.id, gone.id);
  await page.goto(path);
  await expect(page.getByText("Cycle does not exist", { exact: true })).toBeVisible();
  // What the page loads, exactly: the app's, the workspace's, the project's and its cycles, then the cycle and its
  // progress, both not found; nothing outside /api/v0 (S2's cycle entry, M6 P5a T9).
  const cycle = `/api/v0/cycles/${gone.id}`;
  const expected = [
    ...APP,
    ...WORKSPACE,
    ...PROJECT,
    ...PROJECT_MEMBER,
    ...PROJECT_CYCLES,
    `GET ${cycle}`,
    `GET ${cycle}/progress`,
  ].map((request) => valued(request, { [slug]: "{slug}", [web.id]: "{project}" }));
  await expect.poll(recorded).toEqual({ pending: 0, requests: expected.toSorted() });
  expect(watch.apiFailures.toSorted()).toEqual([`404 GET ${cycle}`, `404 GET ${cycle}/progress`]);
  expect([watch.oldApiRequests, watch.pageErrors, watch.cspViolations]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch, { errors: failedLoads(browserName, 404, 404) });
  // Its button leads to Web's cycles, of which there is none.
  await page.getByRole("button", { name: "View other cycles", exact: true }).click();
  await expect(page).toHaveURL(cyclesPath(slug, web.id));
  await expect(page.getByText("Group and timebox your work in Cycles.", { exact: true })).toBeVisible();
});
