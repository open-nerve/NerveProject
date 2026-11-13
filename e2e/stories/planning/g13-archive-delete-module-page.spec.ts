import { createProject, createWorkspace, slugFor } from "../../fixtures/api";
import { expectModules } from "../../fixtures/assert/planning";
import { emailFor } from "../../fixtures/auth";
import { expectQuietConsole, watchPage } from "../../fixtures/browser";
import {
  chooseFromMenu,
  dismissToasts,
  failedLoads,
  itemMenu,
  menuEntries,
  recordApi,
  seenBy,
  toast,
} from "../../fixtures/cycle-pages";
import {
  announced,
  closedByEscape,
  closedWithin,
  enabledWithin,
  modalDialog,
  shownWithin,
} from "../../fixtures/dialogs";
import {
  archivedModulesPath,
  changedModule,
  listedOn,
  madeModule,
  modulePagesWorld,
  modulePath,
  modulesPath,
  noModulesYet,
  openModulesList,
  turnModules,
} from "../../fixtures/module-pages";
import { createModule, moduleCalls } from "../../fixtures/modules";
import { APP, PROJECT, PROJECT_MEMBER, PROJECT_MODULES, WORKSPACE, valued } from "../../fixtures/mounts";
import { moveWithinApp, registerOnboarded, sentHeld, sentTo } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// G13's page version (M6 P6a spec 2.6, R5, R11, R12): a project's closed modules archived from its list, its archive of
// modules, a module's preview there, and its restore, by its member in his browser; a module's deletion, by those nerve
// lets delete it; and the two pages S2 covered until the module pages read nerve's API (P6a T8, T11): a module nerve
// does not have, and the archive while it holds no module. The API version is g13-archive-delete-module.spec.ts; both
// assert the same rows.

test("G13 (page): Web's archive of modules, empty, says so at once, never a placeholder that waits, and asks nerve's API alone; on Web's list, Doing, in progress, offers its archive disabled with the reason, Done and Dropped theirs; Done's archive waits for nerve and goes, said once nerve has archived it; Dropped's, refused as Web's modules were turned off meanwhile, stays, saying why and never success, then goes; the archive lists them, Done previewed as nerve reads it, read only; Done's restore, from its row while it is previewed, is said once nerve has restored it, the archive shown without the preview; an archived module's menu offers no deletion, even to the admin", async ({
  api,
  baseURL,
  browser,
  browserName,
  db,
  signedInPage,
}, testInfo) => {
  // mem's page, then another browser, signed in anew
  test.setTimeout(60_000);
  const { admin, mem, slug, web, token } = await modulePagesWorld(api, testInfo);
  const doing = await createModule(api, token, web.id, { name: "Doing", status: "in-progress" });
  const done = await createModule(api, token, web.id, { name: "Done", status: "completed" });
  const dropped = await createModule(api, token, web.id, { name: "Dropped", status: "cancelled" });
  let rows = [
    madeModule("Doing", "in-progress", admin.email),
    madeModule("Done", "completed", admin.email, { sort_order: 55535 }),
    madeModule("Dropped", "cancelled", admin.email, { sort_order: 45535 }),
  ];
  await expectModules(db, web.id, rows);

  // Web's archive of modules holds none: mem's page says so as soon as nerve has answered, never a placeholder that
  // waits for good. It asks nerve's API alone, for what a workspace's page loads and the archive (S2's archived modules
  // entry, M6 P6a T11): the archive is not one of the project's pages that read the project and its own resources.
  const page = await signedInPage(mem.tokens);
  const watch = await watchPage(page);
  const recorded = recordApi(page);
  const archive = archivedModulesPath(slug, web.id);
  await page.goto(archive);
  await expect(page.getByText("No archived Modules yet", { exact: true })).toBeVisible();
  await expect(page.getByRole("main").last().locator(".animate-pulse")).toHaveCount(0);
  const expected = [...APP, ...WORKSPACE, `GET /api/v0/projects/{project}/modules?archived=true`].map((request) =>
    valued(request, { [slug]: "{slug}", [web.id]: "{project}" })
  );
  await expect.poll(recorded).toEqual({ pending: 0, requests: expected.toSorted() });
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);

  // On Web's list, opened from the sidebar, Doing, in progress, offers its archive disabled, with the reason; Done,
  // completed, and Dropped, cancelled, theirs.
  await openModulesList(page, slug, web.id);
  const listed = listedOn(page);
  await expect.poll(listed).toEqual([doing.id, done.id, dropped.id]);
  const reasoned = page.getByRole("menu").getByRole("menuitem", { name: /^Archive/ });
  const archiveOffered = async (name: string) => {
    await itemMenu(page, name).click();
    const offered = [(await reasoned.isDisabled()) ? "disabled" : "enabled", (await reasoned.textContent()) ?? ""];
    await page.keyboard.press("Escape");
    await expect(reasoned).toHaveCount(0);
    return offered;
  };
  expect([await archiveOffered("Doing"), await archiveOffered("Done"), await archiveOffered("Dropped")]).toEqual([
    ["disabled", "ArchiveOnly completed or cancelled modules can be archived"],
    ["enabled", "Archive"],
    ["enabled", "Archive"],
  ]);

  // Done's archive, named by it, waits for nerve: Escape does not close it, Cancel is disabled. Once nerve has archived
  // Done, the dialog closes and says so, and Done's row has gone.
  await chooseFromMenu(page, "Done", "Archive");
  const dialog = modalDialog(page);
  await expect(dialog).toHaveAccessibleName("Archive module Done");
  const confirm = dialog.getByRole("button", { name: "Archive", exact: true });
  const doneArchive = `/api/v0/modules/${done.id}/archive`;
  const archiving = await sentHeld(page, "POST", doneArchive, () => confirm.click());
  expect(await closedByEscape(page)).toBe(false);
  expect(await enabledWithin(dialog.getByRole("button", { name: "Cancel", exact: true }))).toBe(false);
  expect((await archiving.release()).status()).toBe(200);
  expect(await closedWithin(page)).toBe(true);
  const archived = toast(page, "Your archives can be found in project archives.");
  await expect(archived).toBeVisible();
  await expect.poll(listed).toEqual([doing.id, dropped.id]);
  rows = changedModule(rows, "Done", { archived: true, by: mem.email });
  await expectModules(db, web.id, rows);

  // Dropped's archive asked for, Web's modules turned off meanwhile: nerve refuses it, said beside the dialog, which
  // stays and says no success; Cancel closes it. Turned on again, Dropped is archived.
  await dismissToasts(page);
  await chooseFromMenu(page, "Dropped", "Archive");
  await expect(dialog).toHaveAccessibleName("Archive module Dropped");
  await turnModules(api, token, web, false);
  const droppedArchive = `/api/v0/modules/${dropped.id}/archive`;
  expect((await sentTo(page, "POST", droppedArchive, () => confirm.click())).answer.status()).toBe(409);
  const refusal = toast(page, "Modules are turned off in this project. Turn them on to change them.");
  await expect(refusal).toBeVisible();
  expect(await announced(refusal)).toBe(true);
  expect(await closedWithin(page)).toBe(false);
  expect(await shownWithin(archived)).toBe(false);
  await expectModules(db, web.id, rows);
  await dialog.getByRole("button", { name: "Cancel", exact: true }).click();
  expect(await closedWithin(page)).toBe(true);
  await turnModules(api, token, web, true);
  await chooseFromMenu(page, "Dropped", "Archive");
  expect((await sentTo(page, "POST", droppedArchive, () => confirm.click())).answer.status()).toBe(200);
  expect(await closedWithin(page)).toBe(true);
  await expect.poll(listed).toEqual([doing.id]);
  rows = changedModule(rows, "Dropped", { archived: true, by: mem.email });
  await expectModules(db, web.id, rows);

  // The archive lists Done and Dropped, in the list's order, by name. Done's row previews it, read as nerve has it,
  // completed, and read only.
  await dismissToasts(page);
  await moveWithinApp(page, archive);
  await expect.poll(listed).toEqual([done.id, dropped.id]);
  await page.locator(`a[href="${modulePath(slug, web.id, done.id)}"]`).click();
  await expect(page).toHaveURL(`${archive}?peekModule=${done.id}`);
  const previewed = page.getByRole("heading", { name: "Done", exact: true });
  await expect(previewed).toBeVisible();
  // the preview's sidebar: the innermost part that holds the module's name and its details
  const preview = page
    .locator("div")
    .filter({ has: previewed })
    .filter({ has: page.locator("dl") })
    .last();
  await expect(preview.getByRole("button", { name: "Completed", exact: true })).toBeDisabled();
  await expect(preview.getByRole("button", { name: "Date range", exact: true })).toBeDisabled();

  // Done restored from its row while it is previewed: said once nerve has restored it, and the archive shows without
  // the preview, Done's row gone from it.
  const restored = toast(page, "Your module can be found in project modules.");
  const restoring = await sentHeld(page, "POST", `/api/v0/modules/${done.id}/unarchive`, () =>
    chooseFromMenu(page, "Done", "Restore")
  );
  expect(await shownWithin(restored)).toBe(false);
  expect((await restoring.release()).status()).toBe(200);
  await expect(restored).toBeVisible();
  await expect(page).toHaveURL(archive);
  await expect.poll(listed).toEqual([dropped.id]);
  rows = changedModule(rows, "Done", { archived: false });
  await expectModules(db, web.id, rows);

  // Archived, Dropped offers its restore and no deletion, to mem and to the admin, who may delete a module.
  const archivedEntries = ["Open in new tab", "Copy link", "Restore"];
  expect(await menuEntries(page, "Dropped")).toEqual(archivedEntries);
  await seenBy(browser, baseURL ?? "", api, admin.email, archive, async (other) => {
    await expect.poll(listedOn(other)).toEqual([dropped.id]);
    expect(await menuEntries(other, "Dropped")).toEqual(archivedEntries);
  });
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[`409 POST ${droppedArchive}`], [], []]);
  await expectQuietConsole(page, watch, { errors: failedLoads(browserName, 409) });
});

test("G13 (page): Launch, mem's, offers him its deletion, and Beta, the admin's, does not; the admin may delete both, and nia, another member, neither, nor may Web's guest or a workspace admin who is its guest; Launch's deletion, refused as Web's modules were turned off meanwhile, stays, saying why; on Launch's page, its deletion waits for nerve, and once nerve has deleted it, says so and goes back to Web's list", async ({
  api,
  baseURL,
  browser,
  browserName,
  db,
  signedInPage,
}, testInfo) => {
  // mem's page, then four other browsers, each signed in anew
  test.setTimeout(60_000);
  const { admin, mem, nia, gus, wanda, slug, web, token } = await modulePagesWorld(api, testInfo);
  const path = modulesPath(slug, web.id);
  const beta = await createModule(api, token, web.id, { name: "Beta" });
  const launch = await createModule(api, mem.tokens.access_token, web.id, { name: "Launch" });
  let rows = [
    madeModule("Beta", "planned", admin.email),
    madeModule("Launch", "planned", mem.email, { sort_order: 55535 }),
  ];
  await expectModules(db, web.id, rows);

  // mem's menus: Launch, his, offers its deletion; Beta, the admin's, does not. Neither may be archived, planned.
  const page = await signedInPage(mem.tokens);
  const watch = await watchPage(page);
  await page.goto(path);
  const listed = listedOn(page);
  await expect.poll(listed).toEqual([beta.id, launch.id]);
  const writes = ["Edit", "Open in new tab", "Copy link", "Archive (disabled)"];
  const reads = ["Open in new tab", "Copy link"];
  expect([await menuEntries(page, "Beta"), await menuEntries(page, "Launch")]).toEqual([writes, [...writes, "Delete"]]);

  // The others' menus, each in a browser of his own: the admin may delete both; nia, another member, changes them and
  // deletes neither; Web's guest, and a workspace admin who is its guest, change none.
  const menusOf = (email: string, want: string[][]) =>
    seenBy(browser, baseURL ?? "", api, email, path, async (other) => {
      await expect.poll(listedOn(other)).toEqual([beta.id, launch.id]);
      expect([await menuEntries(other, "Beta"), await menuEntries(other, "Launch")], email).toEqual(want);
    });
  await menusOf(admin.email, [
    [...writes, "Delete"],
    [...writes, "Delete"],
  ]);
  await menusOf(nia.email, [writes, writes]);
  await menusOf(gus.email, [reads, reads]);
  await menusOf(wanda.email, [reads, reads]);

  // Launch's deletion asked for, Web's modules turned off meanwhile: nerve refuses it, said beside the dialog, which
  // stays; Cancel closes it.
  await chooseFromMenu(page, "Launch", "Delete");
  const dialog = modalDialog(page);
  await expect(dialog).toHaveAccessibleName("Delete module");
  await expect(dialog).toContainText("Launch");
  await turnModules(api, token, web, false);
  const confirm = dialog.getByRole("button", { name: "Delete", exact: true });
  const launchPath = `/api/v0/modules/${launch.id}`;
  expect((await sentTo(page, "DELETE", launchPath, () => confirm.click())).answer.status()).toBe(409);
  const refusal = toast(page, "Modules are turned off in this project. Turn them on to change them.");
  await expect(refusal).toBeVisible();
  expect(await announced(refusal)).toBe(true);
  expect(await closedWithin(page)).toBe(false);
  const deleted = toast(page, "Module deleted successfully.");
  expect(await shownWithin(deleted)).toBe(false);
  await expectModules(db, web.id, rows);
  await dialog.getByRole("button", { name: "Cancel", exact: true }).click();
  expect(await closedWithin(page)).toBe(true);
  await turnModules(api, token, web, true);

  // On Launch's page, its header's menu deletes it: the deletion waits for nerve, Escape not closing its dialog nor
  // Cancel enabled; once nerve has deleted it, the dialog closes, says so, and Web's list shows, without Launch.
  await dismissToasts(page);
  await page.locator(`a[href="${modulePath(slug, web.id, launch.id)}"]`).click();
  await expect(page).toHaveURL(modulePath(slug, web.id, launch.id));
  await expect(page.getByRole("heading", { name: "Launch", exact: true })).toBeVisible();
  await chooseFromMenu(page, "Launch", "Delete");
  await expect(dialog).toHaveAccessibleName("Delete module");
  const deletion = await sentHeld(page, "DELETE", launchPath, () => confirm.click());
  expect(await closedByEscape(page)).toBe(false);
  expect(await enabledWithin(dialog.getByRole("button", { name: "Cancel", exact: true }))).toBe(false);
  expect((await deletion.release()).status()).toBe(204);
  expect(await closedWithin(page)).toBe(true);
  await expect(deleted).toBeVisible();
  await expect(page).toHaveURL(path);
  await expect.poll(listed).toEqual([beta.id]);
  rows = changedModule(rows, "Launch", { deleted: true });
  await expectModules(db, web.id, rows);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[`409 DELETE ${launchPath}`], [], []]);
  await expectQuietConsole(page, watch, { errors: failedLoads(browserName, 409) });
});

test("G13 (page): the address of Gone, a module of Web deleted, says that the module does not exist and leads to Web's modules; the page asks nerve's API alone, Web's modules among its project's reads, its reads of the module, its progress and its links not found, and logs nothing else", async ({
  api,
  browserName,
  signedInPage,
}, testInfo) => {
  const tokens = await registerOnboarded(api, emailFor(testInfo, "admin"));
  const token = tokens.access_token;
  const slug = slugFor(testInfo);
  await createWorkspace(api, token, { name: "Acme", slug });
  const web = await createProject(api, token, slug, { name: "Web", identifier: "WEB" });
  await turnModules(api, token, web, true);
  const gone = await createModule(api, token, web.id, { name: "Gone" });
  expect(await moduleCalls(api).remove(token, gone.id), "Gone deleted").toEqual({ status: 204 });

  const page = await signedInPage(tokens);
  const watch = await watchPage(page);
  const recorded = recordApi(page);
  await page.goto(modulePath(slug, web.id, gone.id));
  await expect(page.getByText("Module does not exist", { exact: true })).toBeVisible();
  // What the page loads, exactly: the app's, the workspace's, the project's and its modules, then the module, its
  // progress and its links, each not found; nothing outside /api/v0 (S2's module entry, M6 P6a T8).
  const module = `/api/v0/modules/${gone.id}`;
  const expected = [
    ...APP,
    ...WORKSPACE,
    ...PROJECT,
    ...PROJECT_MEMBER,
    ...PROJECT_MODULES,
    `GET ${module}`,
    `GET ${module}/progress`,
    `GET ${module}/links`,
  ].map((request) => valued(request, { [slug]: "{slug}", [web.id]: "{project}" }));
  await expect.poll(recorded).toEqual({ pending: 0, requests: expected.toSorted() });
  expect(watch.apiFailures.toSorted()).toEqual(
    [`404 GET ${module}`, `404 GET ${module}/progress`, `404 GET ${module}/links`].toSorted()
  );
  expect([watch.oldApiRequests, watch.pageErrors, watch.cspViolations]).toEqual([[], [], []]);
  // Its button leads to Web's modules, of which there is none.
  await page.getByRole("button", { name: "View other modules", exact: true }).click();
  await expect(page).toHaveURL(modulesPath(slug, web.id));
  await expect(noModulesYet(page)).toBeVisible();
  await expectQuietConsole(page, watch, { errors: failedLoads(browserName, 404, 404, 404) });
});
