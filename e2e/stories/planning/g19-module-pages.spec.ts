import type { Locator, Page } from "@playwright/test";

import { addProjectMembers, createProject, createWorkspace, slugFor } from "../../fixtures/api";
import { expectModules } from "../../fixtures/assert/planning";
import { anotherTabSignsIn } from "../../fixtures/auth";
import { expectQuietConsole, refuseClipboardWrites, watchPage } from "../../fixtures/browser";
import { chooseFromMenu, cycleMenu, dismissToasts, entriesOf, rowMenusKept, toast } from "../../fixtures/cycle-pages";
import { sidebarDetail } from "../../fixtures/cycle-progress-pages";
import type { Database } from "../../fixtures/db";
import { closedByEscape, closedWithin, modalDialog, shownWithin } from "../../fixtures/dialogs";
import { createIssue } from "../../fixtures/issues";
import {
  layoutButton,
  listedOn,
  madeModule,
  modulePagesWorld,
  modulePath,
  modulesPath,
  noModulesYet,
  turnModules,
} from "../../fixtures/module-pages";
import { createModule, createModuleLink, expectModuleLinkCrawled } from "../../fixtures/modules";
import { moveWithinApp, sentHeld } from "../../fixtures/settings-pages";
import { stateIds } from "../../fixtures/states";
import { expect, test } from "../../fixtures/test";

// G19, the module pages' scope (M6 P6a spec 2.6, 2.7, 2.11; v0 design 7.7): what a page sent belongs to the session and
// the page it was sent from, so nerve's late answer says nothing on a page that came after; a module's page is its
// module's; and the pages' controls say what they are and whether they are open. There is no API version: the late
// answers are the browser's.

/** The new module dialog of page's project's list, opened by Add Module, its name filled with name. */
async function newModuleDialog(page: Page, name: string) {
  await page.getByRole("button", { name: "Add Module", exact: true }).click();
  const dialog = modalDialog(page);
  await expect(dialog).toHaveAccessibleName("Create Module");
  await dialog.getByRole("textbox", { name: "Title", exact: true }).fill(name);
  return dialog;
}

/** The ids of the modules of the project of projectId, by their names, as the database has them. */
async function moduleIds(db: Database, projectId: string): Promise<Record<string, string>> {
  const rows = await db.query<{ id: string; name: string }>(
    "SELECT id::text, name FROM modules WHERE project_id = $1",
    [projectId]
  );
  return Object.fromEntries(rows.map(({ id, name }) => [name, id]));
}

test("G19 (page): mem's new module Held is made by nerve while its answer waits, and another tab signs nia in: mem's dialog ends with his session; nia's own new module dialog stays open with what she typed as mem's answer arrives, which says nothing and goes nowhere on her page; Held is mem's", async ({
  api,
  context,
  db,
  signedInPage,
}, testInfo) => {
  const { mem, nia, slug, web } = await modulePagesWorld(api, testInfo);
  const path = modulesPath(slug, web.id);
  const listed = `/api/v0/projects/${web.id}/modules`;
  const page = await signedInPage(mem.tokens);
  const watch = await watchPage(page);
  await page.goto(path);
  await expect(noModulesYet(page)).toBeVisible();
  const dialog = await newModuleDialog(page, "Held");
  const create = dialog.getByRole("button", { name: "Create Module", exact: true });
  const held = await sentHeld(page, "POST", listed, () => create.click());
  let released = false;
  try {
    expect(held.body).toEqual({ name: "Held", status: "backlog" });
    // nerve has made Held, its answer held from the page
    await expect.poll(async () => Object.keys(await moduleIds(db, web.id))).toEqual(["Held"]);
    const { Held: heldId } = await moduleIds(db, web.id);

    // Another tab keeps a sign-in of nia: mem's page follows, its dialog gone with his session; nia's list has Held.
    await anotherTabSignsIn(context, api, nia.email);
    await expect(dialog).toHaveCount(0);
    await expect.poll(listedOn(page)).toEqual([heldId]);

    // nia's own new module dialog: mem's answer arrives, and leaves it open with what she typed, says nothing, and
    // moves nowhere.
    const name = (await newModuleDialog(page, "Nia's own")).getByRole("textbox", { name: "Title", exact: true });
    expect((await held.release()).status()).toBe(201);
    released = true;
    expect(await shownWithin(toast(page, "Module created successfully."))).toBe(false);
    expect(await closedWithin(page)).toBe(false);
    await expect(name).toHaveValue("Nia's own");
    await expect(page).toHaveURL(path);
    expect(await closedByEscape(page)).toBe(true);
  } finally {
    if (!released) await held.release();
  }
  await expectModules(db, web.id, [madeModule("Held", "backlog", mem.email)]);
  expect(watch.apiRequests.filter((request) => request === `POST ${listed}`)).toHaveLength(1);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});

test("G19 (page): with mem's new module dialog open on Web's list and its module sent, Back to Ops, a move to Ops and a move to Infra of Beta, another workspace, each end the dialog, nerve's late answer making the module and saying nothing on the page then shown; back on Web, no dialog opens, and the three are Web's modules, mem's", async ({
  api,
  db,
  signedInPage,
}, testInfo) => {
  const { admin, mem, slug, web, token } = await modulePagesWorld(api, testInfo);
  // Ops, Acme's other project, mem's too; Infra, of mem's workspace Beta; Web has Roadmap, the admin's module.
  const ops = await createProject(api, token, slug, { name: "Ops", identifier: "OPS" });
  await addProjectMembers(api, token, ops.id, [{ member_id: mem.id, role: 15 }]);
  await turnModules(api, token, ops, true);
  const beta = slugFor(testInfo, "beta");
  await createWorkspace(api, mem.tokens.access_token, { name: "Beta", slug: beta });
  const infra = await createProject(api, mem.tokens.access_token, beta, { name: "Infra", identifier: "INF" });
  await turnModules(api, mem.tokens.access_token, infra, true);
  const roadmap = await createModule(api, token, web.id, { name: "Roadmap" });
  const [webList, opsList, infraList] = [
    modulesPath(slug, web.id),
    modulesPath(slug, ops.id),
    modulesPath(beta, infra.id),
  ];
  const listed = `/api/v0/projects/${web.id}/modules`;

  const page = await signedInPage(mem.tokens);
  const watch = await watchPage(page);
  await page.goto(opsList);
  await expect(noModulesYet(page)).toBeVisible();
  // On Web's list, Roadmap's, no dialog is open: mem's dialog of the name given, its module sent and nerve's answer
  // held.
  const heldOnWeb = async (name: string) => {
    await moveWithinApp(page, webList);
    await expect(cycleMenu(page, "Roadmap")).toBeVisible();
    expect(await shownWithin(modalDialog(page)), "a dialog open on Web's list").toBe(false);
    const dialog = await newModuleDialog(page, name);
    const held = await sentHeld(page, "POST", listed, () =>
      dialog.getByRole("button", { name: "Create Module", exact: true }).click()
    );
    expect(held.body).toEqual({ name, status: "backlog" });
    return held;
  };
  // The page of path, of no module, shows no dialog; nerve's answer, let through, says nothing there.
  const lateOn = async (path: string, held: Awaited<ReturnType<typeof heldOnWeb>>) => {
    await expect(page).toHaveURL(path);
    await expect(noModulesYet(page)).toBeVisible();
    await expect(modalDialog(page)).toHaveCount(0);
    expect((await held.release()).status()).toBe(201);
    expect(await shownWithin(toast(page, "Module created successfully."))).toBe(false);
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

  // Back on Web, no dialog opens; its modules are listed by name, the three made by nerve, mem's.
  await moveWithinApp(page, webList);
  await expect(cycleMenu(page, "Roadmap")).toBeVisible();
  expect(await shownWithin(modalDialog(page)), "a dialog open on Web's list").toBe(false);
  const ids = await moduleIds(db, web.id);
  await expect.poll(listedOn(page)).toEqual([ids["Held across"], ids["Held away"], ids["Held back"], roadmap.id]);
  await expectModules(db, web.id, [
    madeModule("Roadmap", "planned", admin.email),
    madeModule("Held back", "backlog", mem.email, { sort_order: 55535 }),
    madeModule("Held across", "backlog", mem.email, { sort_order: 45535 }),
    madeModule("Held away", "backlog", mem.email, { sort_order: 35535 }),
  ]);
  expect(watch.apiRequests.filter((request) => request === `POST ${listed}`)).toHaveLength(3);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});

test("G19 (page): Web's list names each module's menu, its order, its filters and their sections, each saying whether it is open, and its layouts, each saying whether it is chosen, a small screen's layout menu too; each row's right-click menu stays inert while closed; Alpha's link copied with the clipboard refused says the copy failed; on Alpha's page, its progress unanswered yet, the header's switcher opens Beta, whose page and progress show, and Alpha's late progress does not take Beta's place; Beta's link copied says the copy failed too", async ({
  api,
  db,
  signedInPage,
}, testInfo) => {
  const { mem, slug, web, token } = await modulePagesWorld(api, testInfo);
  // Alpha and Beta, in January and February 2031: Alpha has two work items, one done; Beta three, one done, and the
  // admin's Notes link, link-local: its crawl is blocked.
  const alpha = await createModule(api, token, web.id, {
    name: "Alpha",
    start_date: "2031-01-01",
    target_date: "2031-01-31",
  });
  const beta = await createModule(api, token, web.id, {
    name: "Beta",
    start_date: "2031-02-01",
    target_date: "2031-02-28",
  });
  const states = await stateIds(api, token, web.id);
  const item = (name: string, state: "Todo" | "Done", module: string) =>
    createIssue(api, token, web.id, { name, state_id: states[state] ?? "", module_ids: [module] });
  await item("Plan", "Done", alpha.id);
  await item("Review", "Todo", alpha.id);
  await item("Build", "Done", beta.id);
  await item("Test", "Todo", beta.id);
  await item("Ship", "Todo", beta.id);
  const notesURL = "http://169.254.169.254/g19/notes";
  const notes = await createModuleLink(api, token, beta.id, { url: notesURL, title: "Notes" });
  await expectModuleLinkCrawled(db, notes.id, [notesURL]);

  const page = await signedInPage(mem.tokens);
  const watch = await watchPage(page);
  await page.goto(modulesPath(slug, web.id));
  await expect.poll(listedOn(page)).toEqual([alpha.id, beta.id]);

  // Alpha's menu, named by it, says whether it is open; so do the order, named by the order chosen, and the filters,
  // and each of their sections.
  const opens = async (button: Locator) => {
    await expect(button).toHaveAttribute("aria-expanded", "false");
    await button.click();
    await expect(button).toHaveAttribute("aria-expanded", "true");
  };
  const menu = cycleMenu(page, "Alpha");
  await opens(menu);
  await expect(page.getByRole("menu")).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(menu).toHaveAttribute("aria-expanded", "false");
  const order = page.getByRole("button", { name: "Order by: Name", exact: true });
  await opens(order);
  await expect(page.getByRole("menu").getByRole("menuitem", { name: "Progress", exact: true })).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(order).toHaveAttribute("aria-expanded", "false");
  const filters = page.getByRole("button", { name: "Filters", exact: true });
  await opens(filters);
  const section = (name: string) => page.getByRole("button", { name, exact: true });
  await Promise.all(
    ["Status", "Lead", "Members", "Start date", "Due date"].map((name) =>
      expect(section(name)).toHaveAttribute("aria-expanded", "true")
    )
  );
  const paused = page.getByRole("button", { name: "Paused", exact: true });
  await expect(paused).toHaveCount(1);
  await section("Status").click();
  await expect(section("Status")).toHaveAttribute("aria-expanded", "false");
  await expect(paused).toHaveCount(0);
  await page.keyboard.press("Escape");
  await expect(filters).toHaveAttribute("aria-expanded", "false");

  // The layouts, each named, say which is chosen: the rows. A small screen's layout menu says whether it is open and
  // lays the list out as cards, which the header's layouts then say, back on a wide screen; the rows again.
  const [rows, cards] = [layoutButton(page, "List layout"), layoutButton(page, "Gallery layout")];
  await expect(rows).toHaveAttribute("aria-pressed", "true");
  await expect(cards).toHaveAttribute("aria-pressed", "false");
  const wide = page.viewportSize();
  await page.setViewportSize({ width: 700, height: wide?.height ?? 720 });
  const layouts = page.getByRole("button", { name: "Layout", exact: true });
  await opens(layouts);
  await page.getByRole("menu").getByRole("menuitem", { name: "Gallery layout", exact: true }).click();
  await expect(layouts).toHaveAttribute("aria-expanded", "false");
  await expect(page.getByText("1/3 work items", { exact: true })).toBeVisible();
  if (wide) await page.setViewportSize(wide);
  await expect(cards).toHaveAttribute("aria-pressed", "true");
  await expect(rows).toHaveAttribute("aria-pressed", "false");
  await rows.click();
  await expect(rows).toHaveAttribute("aria-pressed", "true");
  await expect(page.getByText("1/3 work items", { exact: true })).toHaveCount(0);

  // Each row's right-click menu stays in the page while closed, inert: none of its entries takes the focus (M3 handoff
  // 3, ruling F-14). A right-click on Beta's row opens its menu, the same entries as its button's, the first focused;
  // Escape closes it, the focus back on the row's link, and it is inert again.
  await expect(page.getByRole("menu")).toHaveCount(0);
  expect(await rowMenusKept(page)).toEqual({ entries: 8, focusable: 0 });
  const betaLink = page.locator(`a[href="${modulePath(slug, web.id, beta.id)}"]`);
  await betaLink.click({ button: "right" });
  const rowMenu = page.getByRole("menu");
  expect(await entriesOf(rowMenu)).toEqual(["Edit", "Open in new tab", "Copy link", "Archive (disabled)"]);
  await expect(rowMenu.getByRole("menuitem", { name: "Edit", exact: true })).toBeFocused();
  await page.keyboard.press("Escape");
  await expect(rowMenu).toHaveCount(0);
  await expect(betaLink).toBeFocused();
  expect(await rowMenusKept(page)).toEqual({ entries: 8, focusable: 0 });

  // Alpha's link copied, the browser refusing the clipboard: the page says the copy failed, never that it copied.
  const failed = toast(page, "Something went wrong. Please try again.");
  const copied = toast(page, "Link copied to clipboard");
  await refuseClipboardWrites(page);
  await chooseFromMenu(page, "Alpha", "Copy link");
  await expect(failed).toBeVisible();
  expect(await shownWithin(copied)).toBe(false);

  // Alpha's page, its progress held: the header's switcher, named by Alpha, lists Web's modules in nerve's order, Beta,
  // made last, first, and opens Beta, whose page shows Beta and its progress, 1 of 3 done. Alpha's progress, let
  // through, does not take its place.
  await dismissToasts(page);
  const late = await sentHeld(page, "GET", `/api/v0/modules/${alpha.id}/progress`, () =>
    page
      .locator(`a[href="${modulePath(slug, web.id, alpha.id)}"]`)
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
  await expect(page).toHaveURL(modulePath(slug, web.id, beta.id));
  await expect(page.getByRole("heading", { name: "Beta", exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Beta", exact: true })).toBeVisible();
  await expect(workItems).toHaveText("1/3");
  expect((await late.release()).status()).toBe(200);
  expect(await shownWithin(workItems.getByText("1/2", { exact: true }))).toBe(false);
  await expect(workItems).toHaveText("1/3");
  await expect(page.getByRole("heading", { name: "Beta", exact: true })).toBeVisible();

  // Beta's Notes link copied, the clipboard still refused: the sidebar says the copy failed, never that it copied.
  await page.getByRole("button", { name: "Copy link: Notes", exact: true }).click();
  await expect(failed).toBeVisible();
  expect(await shownWithin(copied)).toBe(false);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});
