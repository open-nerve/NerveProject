import type { Page } from "@playwright/test";

import { expectModules, type ModuleRow } from "../../fixtures/assert/planning";
import { shownNameOf } from "../../fixtures/auth";
import { expectQuietConsole, watchPage, type PageWatch } from "../../fixtures/browser";
import {
  calendarOf,
  chooseFromMenu,
  dismissToasts,
  failedLoads,
  pickDays,
  seenBy,
  shownDay,
  shownDays,
  tabKey,
  tabStops,
  toast,
} from "../../fixtures/cycle-pages";
import { avatarNames, sidebarDetail } from "../../fixtures/cycle-progress-pages";
import {
  announced,
  closedByEscape,
  closedWithin,
  enabledWithin,
  modalDialog,
  sentWithin,
  shownWithin,
} from "../../fixtures/dialogs";
import {
  changedModule,
  expectMembersOffered,
  linkedModules,
  madeModule,
  modulePagesWorld,
  modulePath,
  moduleRow,
  modulesPath,
  noModulesYet,
  turnModules,
} from "../../fixtures/module-pages";
import { createModule } from "../../fixtures/modules";
import { sentHeld, sentTo } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// G9's page version (M6 P6a spec 2.6, 2.7, R5, R6, R11, R13): a project's modules created from its modules list, by its
// member in his browser, through the new module dialog; changed by their dialog and by a module's sidebar. The API
// version is g9-create-update-module.spec.ts; both assert the same rows.

/** A change of a module's members as it was sent, its members sorted: nerve takes them as a set (R6). */
function sortedMembers(body: unknown): unknown {
  const { member_ids, ...rest } = body as { member_ids: string[] };
  return { ...rest, member_ids: member_ids.toSorted() };
}

test("G9 (page): Web's modules are off, so its member's list says so and reads none, Web's features its admin's alone to open; turned on, the empty list offers the first module, disabled for Web's guest; mem's new module dialog, named by its title, has its name focused, Tab goes round it and Escape closes it, the focus back on Add Module; a name left empty is said under it and sent nowhere, one of spaces nerve refuses there; Docs, of a name alone, is sent with its status and no days, and the dialog waits for nerve; Release, its days picked in the dialog, its lead and members picked among Web's members who are not its guests, is refused under Docs's name in another case, the dialog keeping what it holds and saying why, and is made under its own; a workspace admin who is Web's guest has no Add Module", async ({
  api,
  baseURL,
  browser,
  browserName,
  db,
  signedInPage,
}, testInfo) => {
  // mem's page, and four other browsers, each signed in anew
  test.setTimeout(90_000);
  const { admin, mem, nia, gus, wanda, slug, web, token } = await modulePagesWorld(api, testInfo, false);
  const path = modulesPath(slug, web.id);
  const listed = `/api/v0/projects/${web.id}/modules`;

  // Web's modules are off: mem's list says so and reads none of them; only an admin may open Web's features.
  const off = (manage: "enabled" | "disabled") => async (other: Page, seen: PageWatch) => {
    const features = other.getByRole("button", { name: "Manage features", exact: true });
    expect(
      await sentWithin(other, "GET", listed, () =>
        expect(other.getByText("Modules are not enabled for the project.", { exact: true })).toBeVisible()
      )
    ).toBe(false);
    expect(seen.apiRequests).not.toContain(`GET ${listed}`);
    await (manage === "enabled" ? expect(features).toBeEnabled() : expect(features).toBeDisabled());
  };
  await seenBy(browser, baseURL ?? "", api, mem.email, path, off("disabled"));
  await seenBy(browser, baseURL ?? "", api, admin.email, path, off("enabled"));

  // Turned on, Web has no module yet: its guest's list says so, and offers him neither its first module nor Add Module.
  await turnModules(api, token, web, true);
  await seenBy(browser, baseURL ?? "", api, gus.email, path, async (other) => {
    await expect(noModulesYet(other)).toBeVisible();
    await expect(other.getByRole("button", { name: "Set your first module", exact: true })).toBeDisabled();
    await expect(other.getByRole("button", { name: "Add Module", exact: true })).toHaveCount(0);
  });

  // mem's list offers the first module, by the dialog its header's Add Module opens too.
  const page = await signedInPage(mem.tokens);
  const watch = await watchPage(page);
  await page.goto(path);
  await expect(noModulesYet(page)).toBeVisible();
  await page.getByRole("button", { name: "Set your first module", exact: true }).click();
  const dialog = modalDialog(page);
  await expect(dialog).toHaveAccessibleName("Create Module");
  expect(await closedByEscape(page)).toBe(true);

  // Opened from the keyboard, the dialog has the focus on its name; Tab goes round its controls in their order, and
  // Escape closes it, the focus back on Add Module.
  const add = page.getByRole("button", { name: "Add Module", exact: true });
  await add.focus();
  await page.keyboard.press("Enter");
  await expect(dialog).toHaveAccessibleName("Create Module");
  const name = dialog.getByRole("textbox", { name: "Title", exact: true });
  await expect(name).toBeFocused();
  expect(await tabStops(page, dialog, tabKey(browserName))).toEqual([
    "Description",
    "Date range",
    "Backlog",
    "Lead",
    "Members",
    "Cancel",
    "Create Module",
    "Project: Web",
    "Title",
  ]);
  expect(await closedByEscape(page)).toBe(true);
  await expect(add).toBeFocused();

  // A name left empty is said under it, which has the focus, and nothing is sent; one of spaces nerve refuses (422),
  // said under it again, the dialog open and nothing written.
  await page.keyboard.press("Enter");
  await expect(name).toBeFocused();
  const create = dialog.getByRole("button", { name: "Create Module", exact: true });
  expect(await sentWithin(page, "POST", listed, () => create.click())).toBe(false);
  await expect(name).toHaveAccessibleDescription("Title is required");
  await expect(name).toBeFocused();
  await expect(name).toHaveAttribute("aria-invalid", "true");
  await name.fill("   ");
  const spaces = await sentTo(page, "POST", listed, () => create.click());
  expect([spaces.answer.status(), spaces.body]).toEqual([422, { name: "   ", status: "backlog" }]);
  expect(await closedWithin(page)).toBe(false);
  await expect(name).toHaveAccessibleDescription("Too short");
  await expect(name).toBeFocused();
  await expect(name).toHaveAttribute("aria-invalid", "true");
  await expectModules(db, web.id, []);

  // Docs, of a name alone: sent with the status the dialog holds, in the backlog, and without days. The dialog waits for
  // nerve, Escape not closing it nor Cancel enabled; once nerve has made the module it closes and says so, the focus
  // back on Add Module, and Docs is listed.
  await name.fill("Docs");
  const docsSent = await sentHeld(page, "POST", listed, () => create.click());
  expect(docsSent.body).toEqual({ name: "Docs", status: "backlog" });
  expect(await closedByEscape(page)).toBe(false);
  expect(await enabledWithin(dialog.getByRole("button", { name: "Cancel", exact: true }))).toBe(false);
  const docsMade = await docsSent.release();
  expect(docsMade.status()).toBe(201);
  const docs: string = (await docsMade.json()).id;
  expect(await closedWithin(page)).toBe(true);
  await expect(toast(page, "Module created successfully.")).toBeVisible();
  await expect(add).toBeFocused();
  const listedIds = () => linkedModules(page.getByRole("main").last());
  await expect.poll(listedIds).toEqual([docs]);
  let rows = [madeModule("Docs", "backlog", mem.email)];
  await expectModules(db, web.id, rows);

  // Release, under Docs's name in another case: described, its days picked in the dialog, in progress, led by the
  // admin, mem and nia its members, each list of Web's members offering them, not its guests. nerve refuses the name
  // (409): the dialog stays with what it holds and says why, read out beside it; nothing is written.
  await dismissToasts(page);
  await page.keyboard.press("Enter");
  await expect(name).toBeFocused();
  await name.fill("docs");
  await dialog.getByPlaceholder("Description", { exact: true }).fill("Checkout");
  const days = dialog.getByRole("button", { name: "Date range", exact: true });
  await pickDays(page, days, ["2031-03-01", "2031-03-31"]);
  await name.click();
  await expect(calendarOf(page)).toBeHidden();
  await dialog.getByRole("button", { name: "Backlog", exact: true }).click();
  await page.getByRole("option", { name: "In Progress", exact: true }).click();
  const [you, adminName, niaName] = ["You", shownNameOf(admin.email), shownNameOf(nia.email)];
  await dialog.getByRole("button", { name: "Lead", exact: true }).click();
  await expectMembersOffered(page, you, adminName, niaName);
  await page.getByRole("option").filter({ hasText: adminName }).click();
  await dialog.getByRole("button", { name: "Members", exact: true }).click();
  await expectMembersOffered(page, you, adminName, niaName);
  await page.getByRole("option").filter({ hasText: you }).click();
  await page.getByRole("option").filter({ hasText: niaName }).click();
  await name.click();
  await expect(page.getByRole("option")).toHaveCount(0);
  const release = {
    status: "in-progress",
    description: "Checkout",
    start_date: "2031-03-01",
    target_date: "2031-03-31",
    lead_id: admin.id,
    member_ids: [mem.id, nia.id],
  };
  const taken = await sentTo(page, "POST", listed, () => create.click());
  expect([taken.answer.status(), taken.body]).toEqual([409, { name: "docs", ...release }]);
  const refusal = toast(page, "A module of this project already has this name.");
  await expect(refusal).toBeVisible();
  expect(await announced(refusal)).toBe(true);
  expect(await closedWithin(page)).toBe(false);
  await expect(name).toHaveValue("docs");
  await expect(dialog.getByPlaceholder("Description", { exact: true })).toHaveValue("Checkout");
  await expect(days).toContainText(shownDay("2031-03-01"));
  await expect(days).toContainText(shownDay("2031-03-31"));
  await expect(dialog.getByRole("button", { name: "In Progress", exact: true })).toBeVisible();
  await expectModules(db, web.id, rows);

  // Named Release, it is made with all it holds, and the dialog closes. Its row shows its days, its status and its lead.
  await name.fill("Release");
  const releaseSent = await sentTo(page, "POST", listed, () => create.click());
  expect([releaseSent.answer.status(), releaseSent.body]).toEqual([201, { name: "Release", ...release }]);
  const releaseId: string = (await releaseSent.answer.json()).id;
  expect(await closedWithin(page)).toBe(true);
  await expect.poll(listedIds).toEqual([docs, releaseId]);
  const row = moduleRow(page, releaseId, "Release");
  await expect(row).toContainText(shownDays("2031-03-01", "2031-03-31"));
  await expect(row.getByRole("button", { name: "In Progress", exact: true })).toBeVisible();
  await expect(row.getByRole("img", { name: adminName, exact: true })).toBeVisible();
  rows = [
    ...rows,
    madeModule("Release", "in-progress", mem.email, {
      description: "Checkout",
      start: "2031-03-01",
      target: "2031-03-31",
      lead: admin.email,
      members: [mem.email, nia.email],
      sort_order: 55535,
    }),
  ];
  await expectModules(db, web.id, rows);

  // A workspace admin who is Web's guest, whom nerve shows only her own work items there, sees the modules and may make
  // none: no Add Module.
  await seenBy(browser, baseURL ?? "", api, wanda.email, path, async (other) => {
    await expect.poll(() => linkedModules(other.getByRole("main").last())).toEqual([docs, releaseId]);
    await expect(other.getByRole("button", { name: "Add Module", exact: true })).toHaveCount(0);
  });
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([
    [`422 POST ${listed}`, `409 POST ${listed}`],
    [],
    [],
  ]);
  await expectQuietConsole(page, watch, { errors: failedLoads(browserName, 422, 409) });
});

test("G9 (page): mem's dialog of Alpha, named for a change and holding it, sends its new name and description alone and waits for nerve; on Sprint A's page, its sidebar's status, lead, members and days are each sent alone, among Web's members who are not its guests, and shown once nerve has made them, the control busy meanwhile; its status and days changed while Web's modules were turned off are refused, said, and shown as nerve has them", async ({
  api,
  browserName,
  db,
  signedInPage,
}, testInfo) => {
  const { admin, mem, nia, slug, web, token } = await modulePagesWorld(api, testInfo);
  // Alpha, the admin's, planned for January 2031, led by mem, mem and the admin its members.
  const alpha = await createModule(api, token, web.id, {
    name: "Alpha",
    start_date: "2031-01-01",
    target_date: "2031-01-31",
    lead_id: mem.id,
    member_ids: [mem.id, admin.id],
  });
  let rows: ModuleRow[] = [
    madeModule("Alpha", "planned", admin.email, {
      start: "2031-01-01",
      target: "2031-01-31",
      lead: mem.email,
      members: [admin.email, mem.email],
    }),
  ];
  await expectModules(db, web.id, rows);

  // Alpha's dialog, named for a change, holds Alpha; renamed Sprint A and described, it sends those alone and waits for
  // nerve; it closes once nerve has changed it, and says so.
  const page = await signedInPage(mem.tokens);
  const watch = await watchPage(page);
  await page.goto(modulesPath(slug, web.id));
  await expect.poll(() => linkedModules(page.getByRole("main").last())).toEqual([alpha.id]);
  await chooseFromMenu(page, "Alpha", "Edit");
  const dialog = modalDialog(page);
  await expect(dialog).toHaveAccessibleName("Update Module");
  const name = dialog.getByRole("textbox", { name: "Title", exact: true });
  await expect(name).toHaveValue("Alpha");
  await name.fill("Sprint A");
  await dialog.getByPlaceholder("Description", { exact: true }).fill("Sign-in");
  const alphaPath = `/api/v0/modules/${alpha.id}`;
  const renamed = await sentHeld(page, "PATCH", alphaPath, () =>
    dialog.getByRole("button", { name: "Update Module", exact: true }).click()
  );
  expect(renamed.body).toEqual({ name: "Sprint A", description: "Sign-in" });
  expect(await closedByEscape(page)).toBe(false);
  expect(await enabledWithin(dialog.getByRole("button", { name: "Cancel", exact: true }))).toBe(false);
  expect((await renamed.release()).status()).toBe(200);
  expect(await closedWithin(page)).toBe(true);
  await expect(toast(page, "Module updated successfully.")).toBeVisible();
  await expect(moduleRow(page, alpha.id, "Sprint A")).toBeVisible();
  rows = changedModule(rows, "Alpha", { name: "Sprint A", description: "Sign-in", by: mem.email });
  await expectModules(db, web.id, rows);

  // On Sprint A's page, its status in progress: the select is busy and shows nerve's status until nerve has changed it,
  // and only then says so.
  await dismissToasts(page);
  await page.locator(`a[href="${modulePath(slug, web.id, alpha.id)}"]`).click();
  await expect(page).toHaveURL(modulePath(slug, web.id, alpha.id));
  await expect(page.getByRole("heading", { name: "Sprint A", exact: true })).toBeVisible();
  const updated = toast(page, "Module updated successfully.");
  await page.getByRole("button", { name: "Planned", exact: true }).click();
  const started = await sentHeld(page, "PATCH", alphaPath, () =>
    page.getByRole("option", { name: "In Progress", exact: true }).click()
  );
  expect(started.body).toEqual({ status: "in-progress" });
  await expect(page.getByRole("button", { name: "Planned", exact: true })).toBeDisabled();
  expect(await shownWithin(updated)).toBe(false);
  expect((await started.release()).status()).toBe(200);
  await expect(updated).toBeVisible();
  const status = page.getByRole("button", { name: "In Progress", exact: true });
  await expect(status).toBeEnabled();

  // Its lead, nia, picked among Web's members who are not its guests: sent alone.
  await dismissToasts(page);
  const [you, adminName, niaName] = ["You", shownNameOf(admin.email), shownNameOf(nia.email)];
  await sidebarDetail(page, "Lead").getByRole("button").click();
  await expectMembersOffered(page, you, adminName, niaName);
  const led = await sentTo(page, "PATCH", alphaPath, () =>
    page.getByRole("option").filter({ hasText: niaName }).click()
  );
  expect([led.answer.status(), led.body]).toEqual([200, { lead_id: nia.id }]);
  await expect(updated).toBeVisible();
  await expect.poll(() => avatarNames(sidebarDetail(page, "Lead"))).toEqual([niaName]);

  // nia added to its members: the whole set nerve has, and nia, sent alone; the members busy until nerve has them.
  await dismissToasts(page);
  const members = sidebarDetail(page, "Members").getByRole("button");
  await members.click();
  await expectMembersOffered(page, you, adminName, niaName);
  const joined = await sentHeld(page, "PATCH", alphaPath, () =>
    page.getByRole("option").filter({ hasText: niaName }).click()
  );
  expect(sortedMembers(joined.body)).toEqual({ member_ids: [admin.id, mem.id, nia.id].toSorted() });
  await expect(members).toBeDisabled();
  expect(await shownWithin(updated)).toBe(false);
  expect((await joined.release()).status()).toBe(200);
  await expect(updated).toBeVisible();
  await expect(members).toBeEnabled();
  await expect
    .poll(() => avatarNames(sidebarDetail(page, "Members")))
    .toEqual([adminName, shownNameOf(mem.email), niaName].toSorted());

  // Its end moved to the 25th: the days picked show and are busy until nerve has changed them, sent as a pair.
  await dismissToasts(page);
  const days = page.getByRole("button", { name: "Date range", exact: true });
  const moved = await sentHeld(page, "PATCH", alphaPath, () => pickDays(page, days, ["2031-01-25"]));
  expect(moved.body).toEqual({ start_date: "2031-01-01", target_date: "2031-01-25" });
  await expect(days).toBeDisabled();
  await expect(days).toContainText(shownDay("2031-01-25"));
  expect(await shownWithin(updated)).toBe(false);
  expect((await moved.release()).status()).toBe(200);
  await expect(updated).toBeVisible();
  await expect(days).toBeEnabled();
  rows = changedModule(rows, "Sprint A", {
    status: "in-progress",
    lead: nia.email,
    members: [admin.email, mem.email, nia.email],
    target: "2031-01-25",
  });
  await expectModules(db, web.id, rows);

  // Web's modules turned off meanwhile: its status paused and its end moved to the 20th are refused, each said, and the
  // sidebar shows nerve's status and days again, saying no success; nothing is written.
  await dismissToasts(page);
  await turnModules(api, token, web, false);
  await status.click();
  const paused = await sentTo(page, "PATCH", alphaPath, () =>
    page.getByRole("option", { name: "Paused", exact: true }).click()
  );
  expect([paused.answer.status(), paused.body]).toEqual([409, { status: "paused" }]);
  const refusal = toast(page, "Modules are turned off in this project. Turn them on to change them.");
  await expect(refusal).toBeVisible();
  await expect(status).toBeEnabled();
  await dismissToasts(page);
  const shortened = await sentTo(page, "PATCH", alphaPath, () => pickDays(page, days, ["2031-01-20"]));
  expect([shortened.answer.status(), shortened.body]).toEqual([
    409,
    { start_date: "2031-01-01", target_date: "2031-01-20" },
  ]);
  await expect(refusal).toBeVisible();
  await expect(days).toBeEnabled();
  await expect(days).toContainText(shownDay("2031-01-25"));
  expect(await shownWithin(updated)).toBe(false);
  await expectModules(db, web.id, rows);
  await turnModules(api, token, web, true);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([
    [`409 PATCH ${alphaPath}`, `409 PATCH ${alphaPath}`],
    [],
    [],
  ]);
  await expectQuietConsole(page, watch, { errors: failedLoads(browserName, 409, 409) });
});
