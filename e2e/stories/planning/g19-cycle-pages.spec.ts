import type { Page } from "@playwright/test";

import { addProjectMembers, createProject, createWorkspace, slugFor } from "../../fixtures/api";
import { expectCycles, type CycleRow } from "../../fixtures/assert/planning";
import { anotherTabSignsIn } from "../../fixtures/auth";
import { expectQuietConsole, refuseClipboardWrites, watchPage } from "../../fixtures/browser";
import {
  chooseFromMenu,
  cycleMenu,
  cyclePagesWorld,
  cyclePath,
  cycleSection,
  cyclesPath,
  dismissToasts,
  linkedCycles,
  toast,
  turnCycles,
} from "../../fixtures/cycle-pages";
import { sidebarDetail } from "../../fixtures/cycle-progress-pages";
import { createCycle } from "../../fixtures/cycles";
import { closedByEscape, closedWithin, modalDialog, shownWithin } from "../../fixtures/dialogs";
import { createIssue } from "../../fixtures/issues";
import { moveWithinApp, sentHeld } from "../../fixtures/settings-pages";
import { stateIds } from "../../fixtures/states";
import { expect, test } from "../../fixtures/test";

// G19, the cycle pages' scope (M6 P5a spec 2.5, 2.6, 2.9; v0 design 7.7): what a page sent belongs to the session and
// the page it was sent from, so nerve's late answer says nothing on a page that came after; a cycle's page is its
// cycle's; and the pages' controls say what they are and whether they are open. There is no API version: the late
// answers are the browser's.

/** A cycle made by the new cycle dialog, as expectCycles reads it: a draft, without a description, owner's. */
function draft(name: string, sort_order: number, owner: string): CycleRow {
  return {
    name,
    description: "",
    start: null,
    end: null,
    sort_order,
    archived: false,
    deleted: false,
    owner,
    by: owner,
  };
}

/** The new cycle dialog of page's project's list, opened by Add cycle, its name filled with name. */
async function newCycleDialog(page: Page, name: string) {
  await page.getByRole("button", { name: "Add cycle", exact: true }).click();
  const dialog = modalDialog(page);
  await expect(dialog).toHaveAccessibleName("Create cycle");
  await dialog.getByRole("textbox", { name: "Title", exact: true }).fill(name);
  return dialog;
}

test("G19 (page): mem's new cycle Held is made by nerve while its answer waits, and another tab signs nia in: mem's dialog ends with his session; nia's own new cycle dialog stays open with what she typed as mem's answer arrives, which says nothing and goes nowhere on her page; Held is mem's", async ({
  api,
  context,
  db,
  signedInPage,
}, testInfo) => {
  const { mem, nia, slug, web } = await cyclePagesWorld(api, testInfo);
  const path = cyclesPath(slug, web.id);
  const listed = `/api/v0/projects/${web.id}/cycles`;
  const page = await signedInPage(mem.tokens);
  const watch = await watchPage(page);
  await page.goto(path);
  const dialog = await newCycleDialog(page, "Held");
  const create = dialog.getByRole("button", { name: "Create cycle", exact: true });
  const held = await sentHeld(page, "POST", listed, () => create.click());
  let released = false;
  try {
    expect(held.body).toEqual({ name: "Held" });
    // nerve has made Held, its answer held from the page
    const made = () =>
      db.query<{ id: string; name: string }>("SELECT id::text, name FROM cycles WHERE project_id = $1", [web.id]);
    await expect.poll(made).toHaveLength(1);
    const [heldRow] = await made();

    // Another tab keeps a sign-in of nia: mem's page follows, its dialog gone with his session; nia's list has Held.
    await anotherTabSignsIn(context, api, nia.email);
    await expect(dialog).toHaveCount(0);
    await expect.poll(() => linkedCycles(cycleSection(page, "Upcoming cycle"))).toEqual([heldRow?.id]);

    // nia's own new cycle dialog: mem's answer arrives, and leaves it open with what she typed, says nothing, and
    // moves nowhere.
    const name = (await newCycleDialog(page, "Nia's own")).getByRole("textbox", { name: "Title", exact: true });
    expect((await held.release()).status()).toBe(201);
    released = true;
    expect(await shownWithin(toast(page, "Cycle created successfully."))).toBe(false);
    expect(await closedWithin(page)).toBe(false);
    await expect(name).toHaveValue("Nia's own");
    await expect(page).toHaveURL(path);
    expect(await closedByEscape(page)).toBe(true);
  } finally {
    if (!released) await held.release();
  }
  await expectCycles(db, web.id, [draft("Held", 65535, mem.email)]);
  expect(watch.apiRequests.filter((request) => request === `POST ${listed}`)).toHaveLength(1);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});

test("G19 (page): with mem's new cycle dialog open on Web's list and its cycle sent, Back to Ops, a move to Ops and a move to Infra of Beta, another workspace, each end the dialog, nerve's late answer making the cycle and saying nothing on the page then shown; back on Web, no dialog opens, and the three are Web's drafts, mem's", async ({
  api,
  db,
  signedInPage,
}, testInfo) => {
  const { admin, mem, slug, web, token } = await cyclePagesWorld(api, testInfo);
  // Ops, Acme's other project, mem's too; Infra, of mem's workspace Beta; Web has Roadmap, the admin's draft.
  const ops = await createProject(api, token, slug, { name: "Ops", identifier: "OPS" });
  await addProjectMembers(api, token, ops.id, [{ member_id: mem.id, role: 15 }]);
  await turnCycles(api, token, ops, true);
  const beta = slugFor(testInfo, "beta");
  await createWorkspace(api, mem.tokens.access_token, { name: "Beta", slug: beta });
  const infra = await createProject(api, mem.tokens.access_token, beta, { name: "Infra", identifier: "INF" });
  await turnCycles(api, mem.tokens.access_token, infra, true);
  const roadmap = await createCycle(api, token, web.id, { name: "Roadmap" });
  const [webList, opsList, infraList] = [
    cyclesPath(slug, web.id),
    cyclesPath(slug, ops.id),
    cyclesPath(beta, infra.id),
  ];
  const listed = `/api/v0/projects/${web.id}/cycles`;

  const page = await signedInPage(mem.tokens);
  const watch = await watchPage(page);
  await page.goto(opsList);
  const empty = page.getByText("Group and timebox your work in Cycles.", { exact: true });
  await expect(empty).toBeVisible();
  // On Web's list, Roadmap's, no dialog is open: mem's dialog of the name given, its cycle sent and nerve's answer held.
  const heldOnWeb = async (name: string) => {
    await moveWithinApp(page, webList);
    await expect(cycleMenu(page, "Roadmap")).toBeVisible();
    expect(await shownWithin(modalDialog(page)), "a dialog open on Web's list").toBe(false);
    const dialog = await newCycleDialog(page, name);
    const held = await sentHeld(page, "POST", listed, () =>
      dialog.getByRole("button", { name: "Create cycle", exact: true }).click()
    );
    expect(held.body).toEqual({ name });
    return held;
  };
  // The page of path, empty of cycles, shows no dialog; nerve's answer, let through, says nothing there.
  const lateOn = async (path: string, held: Awaited<ReturnType<typeof heldOnWeb>>) => {
    await expect(page).toHaveURL(path);
    await expect(empty).toBeVisible();
    await expect(modalDialog(page)).toHaveCount(0);
    expect((await held.release()).status()).toBe(201);
    expect(await shownWithin(toast(page, "Cycle created successfully."))).toBe(false);
    await expect(page).toHaveURL(path);
  };

  // Back, to Ops.
  const back = await heldOnWeb("Held back");
  await page.goBack();
  await lateOn(opsList, back);
  // A move to Ops, another project.
  const across = await heldOnWeb("Held across");
  await moveWithinApp(page, opsList);
  await lateOn(opsList, across);
  // A move to Infra, of another workspace.
  const away = await heldOnWeb("Held away");
  await moveWithinApp(page, infraList);
  await lateOn(infraList, away);

  // Back on Web, no dialog opens; its drafts are listed by name, the three made by nerve, mem's.
  await moveWithinApp(page, webList);
  await expect(cycleMenu(page, "Roadmap")).toBeVisible();
  expect(await shownWithin(modalDialog(page)), "a dialog open on Web's list").toBe(false);
  const ids = Object.fromEntries(
    (
      await db.query<{ id: string; name: string }>("SELECT id::text, name FROM cycles WHERE project_id = $1", [web.id])
    ).map(({ id, name }) => [name, id])
  );
  await expect
    .poll(() => linkedCycles(cycleSection(page, "Upcoming cycle")))
    .toEqual([ids["Held across"], ids["Held away"], ids["Held back"], roadmap.id]);
  await expectCycles(db, web.id, [
    draft("Roadmap", 65535, admin.email),
    draft("Held back", 55535, mem.email),
    draft("Held across", 45535, mem.email),
    draft("Held away", 35535, mem.email),
  ]);
  expect(watch.apiRequests.filter((request) => request === `POST ${listed}`)).toHaveLength(3);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});

test("G19 (page): Web's list names each cycle's menu and its filters and their sections, each saying whether it is open; Alpha's link copied with the clipboard refused says the copy failed; on Alpha's page, its progress unanswered yet, the header's switcher opens Beta, whose page and progress show, and Alpha's late progress does not take Beta's place", async ({
  api,
  signedInPage,
}, testInfo) => {
  const { mem, slug, web, token } = await cyclePagesWorld(api, testInfo);
  // Alpha and Beta, in January and February 2031: Alpha has two work items, one done; Beta three, one done.
  const alpha = await createCycle(api, token, web.id, {
    name: "Alpha",
    start_date: "2031-01-01",
    end_date: "2031-01-14",
  });
  const beta = await createCycle(api, token, web.id, {
    name: "Beta",
    start_date: "2031-02-01",
    end_date: "2031-02-14",
  });
  const states = await stateIds(api, token, web.id);
  const item = (name: string, state: "Todo" | "Done", cycle: string) =>
    createIssue(api, token, web.id, { name, state_id: states[state] ?? "", cycle_id: cycle });
  await item("Plan", "Done", alpha.id);
  await item("Review", "Todo", alpha.id);
  await item("Build", "Done", beta.id);
  await item("Test", "Todo", beta.id);
  await item("Ship", "Todo", beta.id);

  const page = await signedInPage(mem.tokens);
  const watch = await watchPage(page);
  await page.goto(cyclesPath(slug, web.id));
  await expect.poll(() => linkedCycles(cycleSection(page, "Upcoming cycle"))).toEqual([alpha.id, beta.id]);

  // Alpha's menu, named by it, says whether it is open; so do the filters, and each of their sections.
  const menu = cycleMenu(page, "Alpha");
  await expect(menu).toHaveAttribute("aria-expanded", "false");
  await menu.click();
  await expect(page.getByRole("menu")).toBeVisible();
  await expect(menu).toHaveAttribute("aria-expanded", "true");
  await page.keyboard.press("Escape");
  await expect(menu).toHaveAttribute("aria-expanded", "false");
  const filters = page.getByRole("button", { name: "Filters", exact: true });
  await expect(filters).toHaveAttribute("aria-expanded", "false");
  await filters.click();
  await expect(filters).toHaveAttribute("aria-expanded", "true");
  const section = (name: string) => page.getByRole("button", { name, exact: true });
  await expect(section("Start date")).toHaveAttribute("aria-expanded", "true");
  await expect(section("End date")).toHaveAttribute("aria-expanded", "true");
  await expect(section("Status")).toHaveAttribute("aria-expanded", "true");
  const completed = page.getByRole("button", { name: "Completed", exact: true });
  await expect(completed).toHaveCount(1);
  await section("Status").click();
  await expect(section("Status")).toHaveAttribute("aria-expanded", "false");
  await expect(completed).toHaveCount(0);
  await page.keyboard.press("Escape");
  await expect(filters).toHaveAttribute("aria-expanded", "false");

  // Alpha's link copied, the browser refusing the clipboard: the page says the copy failed, never that it copied.
  await refuseClipboardWrites(page);
  await chooseFromMenu(page, "Alpha", "Copy link");
  await expect(toast(page, "Something went wrong. Please try again.")).toBeVisible();
  expect(await shownWithin(toast(page, "Link copied to clipboard"))).toBe(false);

  // Alpha's page, its progress held: the header's switcher, named by Alpha, lists Web's cycles in nerve's order, Beta,
  // made last, first, and opens Beta, whose page shows Beta and its progress, 1 of 3 done. Alpha's progress, let
  // through, does not take its place.
  await dismissToasts(page);
  const late = await sentHeld(page, "GET", `/api/v0/cycles/${alpha.id}/progress`, () =>
    page
      .locator(`a[href="${cyclePath(slug, web.id, alpha.id)}"]`)
      .first()
      .click()
  );
  await expect(page.getByRole("heading", { name: "Alpha", exact: true })).toBeVisible();
  const workItems = sidebarDetail(page, "Work items");
  await expect(workItems).toHaveText("");
  await page.getByRole("button", { name: "Alpha", exact: true }).click();
  const options = page.getByRole("option");
  await expect(options).toHaveText(["Beta", "Alpha"]);
  await options.filter({ hasText: "Beta" }).click();
  await expect(page).toHaveURL(cyclePath(slug, web.id, beta.id));
  await expect(page.getByRole("heading", { name: "Beta", exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Beta", exact: true })).toBeVisible();
  await expect(workItems).toHaveText("1/3");
  expect((await late.release()).status()).toBe(200);
  expect(await shownWithin(workItems.getByText("1/2", { exact: true }))).toBe(false);
  await expect(workItems).toHaveText("1/3");
  await expect(page.getByRole("heading", { name: "Beta", exact: true })).toBeVisible();
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});
