import { createProject, createWorkspace, slugFor } from "../../fixtures/api";
import { expectCycles, type CycleRow } from "../../fixtures/assert/planning";
import { emailFor } from "../../fixtures/auth";
import { expectQuietConsole, watchPage } from "../../fixtures/browser";
import {
  calendarOf,
  cyclePagesWorld,
  cycleSection,
  cyclesPath,
  dismissToasts,
  failedLoads,
  linkedCycles,
  pickDays,
  seenBy,
  shownDay,
  tabKey,
  tabStops,
  toast,
  turnCycles,
} from "../../fixtures/cycle-pages";
import { createCycle } from "../../fixtures/cycles";
import {
  announced,
  closedByEscape,
  closedWithin,
  composeWithEnter,
  composingKey,
  enabledWithin,
  modalDialog,
  sentWithin,
} from "../../fixtures/dialogs";
import { registerOnboarded, sentHeld, sentTo } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// G1's page version (M6 P5a spec 2.5, 2.6, 2.9): a project's cycles created from its cycles list, by its member in his
// browser, through the new cycle dialog, its name composed by an input method too (M4 handoff M6-04). The API version
// is g1-create-cycle.spec.ts; both assert the same rows.

test("G1 (page): Web's cycles are off, so its member's list says so and reads none, Web's features its admin's alone to open; turned on, the empty list offers the first cycle; mem's new cycle dialog, named by its title, has its name focused, Tab goes round it and Escape closes it, the focus back on Add cycle; a name left empty is said under it and sent nowhere, one of spaces nerve refuses there; Backlog, a draft, is sent without days and the dialog waits for nerve; Sprint 1's days ending on Sprint 2's first are refused, the dialog keeping what it holds and saying why, and are taken ending the day before; Web's guest and a workspace admin who is its guest have no Add cycle", async ({
  api,
  baseURL,
  browser,
  browserName,
  db,
  signedInPage,
}, testInfo) => {
  const { admin, mem, gus, wanda, slug, web, token } = await cyclePagesWorld(api, testInfo, false);
  const path = cyclesPath(slug, web.id);
  const listed = `/api/v0/projects/${web.id}/cycles`;

  // Web's cycles are off: mem's list says so and reads none of them; only an admin may open Web's features.
  const page = await signedInPage(mem.tokens);
  const watch = await watchPage(page);
  expect(
    await sentWithin(page, "GET", listed, async () => {
      await page.goto(path);
      await expect(page.getByText("Cycles is not enabled for this project.", { exact: true })).toBeVisible();
    })
  ).toBe(false);
  await expect(page.getByRole("button", { name: "Manage features", exact: true })).toBeDisabled();
  await seenBy(browser, baseURL ?? "", api, admin.email, path, async (other) => {
    await expect(other.getByRole("button", { name: "Manage features", exact: true })).toBeEnabled();
  });

  // Turned on, Web has no cycle yet: mem's list offers the first, by the dialog its header's Add cycle opens too.
  await turnCycles(api, token, web, true);
  await page.reload();
  await expect(page.getByText("Group and timebox your work in Cycles.", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Set your first cycle", exact: true }).click();
  const dialog = modalDialog(page);
  await expect(dialog).toHaveAccessibleName("Create cycle");
  expect(await closedByEscape(page)).toBe(true);

  // Opened from the keyboard, the dialog has the focus on its name; Tab goes round its controls in their order, and
  // Escape closes it, the focus back on Add cycle.
  const add = page.getByRole("button", { name: "Add cycle", exact: true });
  await add.focus();
  await page.keyboard.press("Enter");
  await expect(dialog).toHaveAccessibleName("Create cycle");
  const name = dialog.getByRole("textbox", { name: "Title", exact: true });
  await expect(name).toBeFocused();
  expect(await tabStops(page, dialog, tabKey(browserName))).toEqual([
    "Description",
    "Date range",
    "Cancel",
    "Create cycle",
    "Project: Web",
    "Title",
  ]);
  expect(await closedByEscape(page)).toBe(true);
  await expect(add).toBeFocused();

  // A name left empty is said under it, which has the focus, and nothing is sent; one of spaces nerve refuses (422),
  // said under it again, the dialog open and nothing written.
  await page.keyboard.press("Enter");
  await expect(name).toBeFocused();
  const create = dialog.getByRole("button", { name: "Create cycle", exact: true });
  expect(await sentWithin(page, "POST", listed, () => create.click())).toBe(false);
  await expect(name).toHaveAccessibleDescription("Title is required");
  await expect(name).toBeFocused();
  await expect(name).toHaveAttribute("aria-invalid", "true");
  await name.fill("   ");
  const spaces = await sentTo(page, "POST", listed, () => create.click());
  expect([spaces.answer.status(), spaces.body]).toEqual([422, { name: "   " }]);
  await expect(name).toHaveAccessibleDescription("Too short");
  await expect(name).toBeFocused();
  await expect(name).toHaveAttribute("aria-invalid", "true");
  await expectCycles(db, web.id, []);

  // Backlog, a draft: sent without days. The dialog waits for nerve, Escape not closing it nor Cancel enabled; once
  // nerve has made the cycle it closes and says so, the focus back on Add cycle, and Backlog is listed.
  await name.fill("Backlog");
  const backlogSent = await sentHeld(page, "POST", listed, () => create.click());
  expect(backlogSent.body).toEqual({ name: "Backlog" });
  expect(await closedByEscape(page)).toBe(false);
  expect(await enabledWithin(dialog.getByRole("button", { name: "Cancel", exact: true }))).toBe(false);
  const backlogMade = await backlogSent.release();
  expect(backlogMade.status()).toBe(201);
  const backlog: string = (await backlogMade.json()).id;
  expect(await closedWithin(page)).toBe(true);
  await expect(toast(page, "Cycle created successfully.")).toBeVisible();
  await expect(add).toBeFocused();
  await expect.poll(() => linkedCycles(cycleSection(page, "Upcoming cycle"))).toEqual([backlog]);
  let rows: CycleRow[] = [
    {
      name: "Backlog",
      description: "",
      start: null,
      end: null,
      sort_order: 65535,
      archived: false,
      deleted: false,
      owner: mem.email,
      by: mem.email,
    },
  ];
  await expectCycles(db, web.id, rows);

  // The admin makes Sprint 2, the 15th to the 28th of March 2031, through the API.
  const sprint2 = await createCycle(api, token, web.id, {
    name: "Sprint 2",
    start_date: "2031-03-15",
    end_date: "2031-03-28",
  });
  rows = [
    ...rows,
    {
      name: "Sprint 2",
      description: "",
      start: "2031-03-15",
      end: "2031-03-28",
      sort_order: 55535,
      archived: false,
      deleted: false,
      owner: admin.email,
      by: admin.email,
    },
  ];

  // Sprint 1, described, picked from the 1st to the 15th of March, Sprint 2's first day: nerve refuses the overlap. The
  // dialog stays with what it holds and says why, read out beside it; nothing is written.
  await dismissToasts(page);
  await page.keyboard.press("Enter");
  await expect(name).toBeFocused();
  await name.fill("Sprint 1");
  await dialog.getByPlaceholder("Description", { exact: true }).fill("Sign-in");
  const days = dialog.getByRole("button", { name: "Date range", exact: true });
  await pickDays(page, days, ["2031-03-01", "2031-03-15"]);
  await name.click();
  await expect(calendarOf(page)).toBeHidden();
  const overlap = await sentTo(page, "POST", listed, () => create.click());
  expect([overlap.answer.status(), overlap.body]).toEqual([
    409,
    { name: "Sprint 1", description: "Sign-in", start_date: "2031-03-01", end_date: "2031-03-15" },
  ]);
  const refusal = toast(page, "These dates overlap another cycle of this project.");
  await expect(refusal).toBeVisible();
  expect(await announced(refusal)).toBe(true);
  expect(await closedWithin(page)).toBe(false);
  await expect(name).toHaveValue("Sprint 1");
  await expect(dialog.getByPlaceholder("Description", { exact: true })).toHaveValue("Sign-in");
  await expect(days).toContainText(shownDay("2031-03-01"));
  await expect(days).toContainText(shownDay("2031-03-15"));
  await expectCycles(db, web.id, rows);

  // Its last day picked again, the 14th: nerve makes Sprint 1, and the dialog closes.
  await pickDays(page, days, ["2031-03-14"]);
  await name.click();
  const sprint1Sent = await sentTo(page, "POST", listed, () => create.click());
  expect([sprint1Sent.answer.status(), sprint1Sent.body]).toEqual([
    201,
    { name: "Sprint 1", description: "Sign-in", start_date: "2031-03-01", end_date: "2031-03-14" },
  ]);
  const sprint1: string = (await sprint1Sent.answer.json()).id;
  expect(await closedWithin(page)).toBe(true);
  rows = [
    ...rows,
    {
      name: "Sprint 1",
      description: "Sign-in",
      start: "2031-03-01",
      end: "2031-03-14",
      sort_order: 45535,
      archived: false,
      deleted: false,
      owner: mem.email,
      by: mem.email,
    },
  ];
  await expectCycles(db, web.id, rows);
  // Read again, the list holds them upcoming by their first days, the draft after them.
  await page.reload();
  await expect.poll(() => linkedCycles(cycleSection(page, "Upcoming cycle"))).toEqual([sprint1, sprint2.id, backlog]);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([
    [`422 POST ${listed}`, `409 POST ${listed}`],
    [],
    [],
  ]);
  await expectQuietConsole(page, watch, { errors: failedLoads(browserName, 422, 409) });

  // Web's guest, and a workspace admin who is its guest, whom nerve shows only their own work items there, see the
  // cycles and may make none: no Add cycle, nor the list's search.
  const noAddCycle = async (other: typeof page) => {
    await expect
      .poll(() => linkedCycles(cycleSection(other, "Upcoming cycle")))
      .toEqual([sprint1, sprint2.id, backlog]);
    await expect(other.getByRole("button", { name: "Add cycle", exact: true })).toHaveCount(0);
    await expect(other.getByRole("button", { name: "Search", exact: true })).toHaveCount(0);
  };
  await seenBy(browser, baseURL ?? "", api, gus.email, path, noAddCycle);
  await seenBy(browser, baseURL ?? "", api, wanda.email, path, noAddCycle);
});

test("G1 (page): an Enter that ends an input method's composition in the name of the admin's new cycle is the input method's: nothing is sent and the dialog stays, holding the name composed; the Enter after the composition sends it, and the dialog closes once nerve has made the cycle", async ({
  api,
  browserName,
  db,
  signedInPage,
}, testInfo) => {
  const email = emailFor(testInfo, "admin");
  const tokens = await registerOnboarded(api, email);
  const token = tokens.access_token;
  const slug = slugFor(testInfo);
  await createWorkspace(api, token, { name: "Acme", slug });
  const web = await createProject(api, token, slug, { name: "Web", identifier: "WEB" });
  await turnCycles(api, token, web, true);
  const listed = `/api/v0/projects/${web.id}/cycles`;

  const page = await signedInPage(tokens);
  const watch = await watchPage(page);
  await page.goto(cyclesPath(slug, web.id));
  await page.getByRole("button", { name: "Set your first cycle", exact: true }).click();
  const dialog = modalDialog(page);
  await expect(dialog).toHaveAccessibleName("Create cycle");
  const name = dialog.getByRole("textbox", { name: "Title", exact: true });
  await expect(name).toBeFocused();

  // Chromium composes 迭代 in the name through its own input method, an Enter ending the composition as an input method
  // gives it, and nothing is sent; Playwright drives no input method in Firefox or WebKit, where 迭代 is filled in.
  if (browserName === "chromium") {
    expect(await sentWithin(page, "POST", listed, () => composeWithEnter(page, "迭代"))).toBe(false);
  } else {
    await name.fill("迭代");
  }
  // In each engine, what an Enter that ends a composition gives the page, in Chromium's and Firefox's order and in
  // Safari's (composingKey): nothing is sent, and the dialog stays, holding 迭代.
  expect(await sentWithin(page, "POST", listed, () => composingKey(name, "Enter"))).toBe(false);
  expect(await closedWithin(page)).toBe(false);
  await expect(name).toHaveValue("迭代");
  await expectCycles(db, web.id, []);

  // The Enter after the composition sends the name; once nerve has made the cycle, the dialog closes and says so.
  const made = await sentTo(page, "POST", listed, () => page.keyboard.press("Enter"));
  expect([made.answer.status(), made.body]).toEqual([201, { name: "迭代" }]);
  expect(await closedWithin(page)).toBe(true);
  await expect(toast(page, "Cycle created successfully.")).toBeVisible();
  await expectCycles(db, web.id, [
    {
      name: "迭代",
      description: "",
      start: null,
      end: null,
      sort_order: 65535,
      archived: false,
      deleted: false,
      owner: email,
      by: email,
    },
  ]);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});
