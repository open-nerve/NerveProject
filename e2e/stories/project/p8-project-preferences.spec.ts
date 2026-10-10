import type { Locator, Page, Response } from "@playwright/test";

import {
  addProjectMembers,
  amidAnotherWorkspace,
  changeProject,
  changeWorkspacePreferences,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
  type Api,
  type ProjectPreferences,
  type ProjectPreferencesUpdate,
} from "../../fixtures/api";
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import type { Database } from "../../fixtures/db";
import { answerTo, holdAnswer, registerOnboarded, sentTo, shownWithin } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";
import { anotherBrowser } from "../../fixtures/workspace-pages";

// P8, a member's display settings in a project (M3 design 2, 3.18): through the API, and through the project's header
// and the sidebar (7.6).

/** GET /api/v0/me/projects/{project_id}/preferences, which must be 200. */
async function read(api: Api, token: string, id: string): Promise<unknown> {
  const { data, error, response } = await api.GET("/api/v0/me/projects/{project_id}/preferences", {
    params: { path: { project_id: id } },
    headers: bearer(token),
  });
  expect(response.status, `read the settings in ${id}: ${JSON.stringify(error)}`).toBe(200);
  return data;
}

/** PATCH /api/v0/me/projects/{project_id}/preferences: its status, and the settings or the problem's fields. */
async function change(api: Api, token: string, id: string, body: ProjectPreferencesUpdate) {
  const { data, error, response } = await api.PATCH("/api/v0/me/projects/{project_id}/preferences", {
    params: { path: { project_id: id } },
    body,
    headers: bearer(token),
  });
  return data
    ? { status: response.status, preferences: data }
    : {
        status: response.status,
        code: error?.code,
        errors: error?.errors?.map((e) => ({ field: e.field, code: e.code })),
      };
}

/** The account of email's display settings in the project as stored, and who wrote them last. */
async function stored(db: Database, id: string, email: string): Promise<unknown[]> {
  return db.query(
    `SELECT s.preferences, s.sort_order, b.email AS by
       FROM project_user_properties s JOIN users u ON u.id = s.user_id JOIN users b ON b.id = s.updated_by_id
      WHERE s.project_id = $1 AND u.email = $2 AND s.deleted_at IS NULL`,
    [id, email]
  );
}

/** The tab of name in the project's header on page: a link outside the sidebar, which has links of the same names. */
function tabOf(page: Page, name: string): Locator {
  return page.locator("xpath=//a[not(ancestor::aside)]").filter({ hasText: new RegExp(`^${name}$`) });
}

/**
 * Opens the menu of the tab of name in the project's header on page, which it has once nerve has given the caller's
 * tab bar: resolves with the menu's "Set as default".
 */
async function menuOf(page: Page, name: string): Promise<Locator> {
  const setAsDefault = page.getByRole("menuitem", { name: "Set as default" });
  await expect(async () => {
    await tabOf(page, name).click({ button: "right" });
    await expect(setAsDefault).toBeVisible({ timeout: 1_000 });
  }).toPass({ timeout: 5_000 });
  return setAsDefault;
}

/** The projects of page's sidebar, in its order. */
function sidebarOrder(page: Page): Locator {
  return page.getByRole("complementary", { name: "Main sidebar" }).locator('[id^="sidebar-"][id$="-JOINED"]');
}

/** The project of id in page's sidebar. */
function inSidebar(page: Page, id: string): Locator {
  return page.getByRole("complementary", { name: "Main sidebar" }).locator(`[id="sidebar-${id}-JOINED"]`);
}

/**
 * Drags the project of id in page's sidebar onto the one of onto by its handle, which shows while the pointer is over
 * the project: the drag starts there, as a hand's would. Resolves with the body page sends and nerve's answer.
 */
async function dragOnto(page: Page, id: string, onto: string): Promise<{ body: unknown; answer: Response }> {
  await inSidebar(page, id).hover();
  await inSidebar(page, id).locator("button").first().hover();
  await page.mouse.down();
  await inSidebar(page, id).hover({ position: { x: 60, y: 8 } });
  return sentTo(page, "PATCH", `/api/v0/me/projects/${id}/preferences`, async () => {
    await inSidebar(page, onto).hover();
    await page.mouse.up();
  });
}

test("P8 (API): the admin opens a project on its modules tab, moves views under more and drags it first in his sidebar, which lasts; an unknown or the work items tab changes nothing; his member's settings stay his own", async ({
  api,
  db,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = (await createPAT(api, (await register(api, adminEmail)).access_token)).token;
  const memberEmail = emailFor(testInfo, "member");
  const member = (await createPAT(api, (await register(api, memberEmail)).access_token)).token;
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin, { name: "Acme", slug });
  await inviteAndAccept(api, admin, slug, { email: memberEmail, token: member }, 15);
  // The admin's sidebar: Wiki 35535, Docs 45535, Web 55535, Ops 65535. Web is neither the first nor the last made,
  // so that each settings row the admin reads or writes must be his own in Web, whichever row the database reads first.
  const web = await amidAnotherWorkspace(api, admin, testInfo, async () => {
    await createProject(api, admin, slug, { name: "Ops", identifier: "OPS" });
    const made = await createProject(api, admin, slug, { name: "Web", identifier: "WEB" });
    await createProject(api, admin, slug, { name: "Docs", identifier: "DOCS" });
    await createProject(api, admin, slug, { name: "Wiki", identifier: "WIKI" });
    await addProjectMembers(api, admin, made.id, [{ member_id: await accountId(api, member), role: 15 }]);
    return made;
  });
  const memberSettings: ProjectPreferences = {
    navigation: { default_tab: "work_items", hide_in_more_menu: [] },
    sort_order: 65535,
  };
  expect(await read(api, admin, web.id)).toEqual({ ...memberSettings, sort_order: 55535 });

  const tabs: ProjectPreferences = {
    navigation: { default_tab: "modules", hide_in_more_menu: ["views"] },
    sort_order: 55535,
  };
  expect(await change(api, admin, web.id, { navigation: tabs.navigation })).toEqual({ status: 200, preferences: tabs });
  // Dragged before Wiki (35535).
  const first = { ...tabs, sort_order: 25535 };
  expect(await change(api, admin, web.id, { sort_order: 25535 })).toEqual({ status: 200, preferences: first });
  // After a refresh the page reads them again, and the sidebar lists Web first.
  expect(await read(api, admin, web.id)).toEqual(first);
  const { data } = await api.GET("/api/v0/workspaces/{slug}/projects", {
    params: { path: { slug } },
    headers: bearer(admin),
  });
  expect(data?.data.map((p) => p.name)).toEqual(["Web", "Wiki", "Docs", "Ops"]);
  const row = [{ preferences: { navigation: first.navigation }, sort_order: 25535, by: adminEmail }];
  expect(await stored(db, web.id, adminEmail)).toEqual(row);

  // Each refused, all at once: none writes.
  const refusals: { body: ProjectPreferencesUpdate; field: string }[] = [
    {
      // A tab the contract does not name: the client's types would not send it.
      body: { navigation: { default_tab: "pages", hide_in_more_menu: [] } } as unknown as ProjectPreferencesUpdate,
      field: "navigation.default_tab",
    },
    {
      body: { navigation: { default_tab: "cycles", hide_in_more_menu: ["work_items"] } },
      field: "navigation.hide_in_more_menu[0]",
    },
  ];
  expect(await Promise.all(refusals.map(({ body }) => change(api, admin, web.id, body))), "the refusals").toEqual(
    refusals.map(({ field }) => ({
      status: 422,
      code: "validation_failed",
      errors: [{ field, code: "invalid_format" }],
    }))
  );
  expect(await stored(db, web.id, adminEmail)).toEqual(row);

  // The member's settings in Web are his own, as the admin's addition made them.
  expect(await read(api, member, web.id)).toEqual(memberSettings);
  expect(await stored(db, web.id, memberEmail)).toEqual([
    { preferences: { navigation: memberSettings.navigation }, sort_order: 65535, by: adminEmail },
  ]);
});

test("P8 (page): the admin makes modules the tab Web opens on and moves its views under more, by the menus of its header's tabs, which offer neither until nerve has given his tab bar; he drags his third project first in his sidebar; each lasts after a refresh; Web's guest is offered the menus and the drag as well", async ({
  api,
  baseURL,
  browser,
  db,
  signedInPage,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = await registerOnboarded(api, adminEmail);
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.access_token, { name: "Acme", slug });
  // His sidebar: Docs 45535, Web 55535, Ops 65535. Web shows its modules and its views: its header has their tabs.
  const ops = await createProject(api, admin.access_token, slug, { name: "Ops", identifier: "OPS" });
  const web = await createProject(api, admin.access_token, slug, { name: "Web", identifier: "WEB" });
  const docs = await createProject(api, admin.access_token, slug, { name: "Docs", identifier: "DOCS" });
  const shown = await changeProject(api, admin.access_token, web.id, { module_view: true, issue_views_view: true });
  expect(shown.status).toBe(200);
  // His projects' navigation is the tabbed one, whose project pages have a header of tabs (W8).
  await changeWorkspacePreferences(api, admin.access_token, slug, { navigation_control_preference: "TABBED" });
  // Gus, acme's guest, is Web's guest and then Docs': his sidebar is Docs 55535, Web 65535. nerve lets a project's
  // guests change their tab bar and their sidebar as it lets its admins (project_preferences.update).
  const gusEmail = emailFor(testInfo, "gus");
  const gus = await registerOnboarded(api, gusEmail);
  await inviteAndAccept(api, admin.access_token, slug, { email: gusEmail, token: gus.access_token }, 5);
  const gusId = await accountId(api, gus.access_token);
  await addProjectMembers(api, admin.access_token, web.id, [{ member_id: gusId, role: 5 }]);
  await addProjectMembers(api, admin.access_token, docs.id, [{ member_id: gusId, role: 5 }]);
  await changeWorkspacePreferences(api, gus.access_token, slug, { navigation_control_preference: "TABBED" });

  const page = await signedInPage(admin);
  const watch = await watchPage(page);
  const settings = `/api/v0/me/projects/${web.id}/preferences`;
  const tab = (name: string) => tabOf(page, name);
  // Until nerve has given his tab bar a tab has no menu: a change made to nerve's default would replace his tab bar.
  const release = await holdAnswer(page, "GET", settings);
  await page.goto(`/${slug}/projects/${web.id}/issues`);
  await tab("Modules").click({ button: "right" });
  expect(await shownWithin(page.getByRole("menuitem", { name: "Set as default" }))).toBe(false);
  expect((await answerTo(page, "GET", settings, release)).status()).toBe(200);
  const setAsDefault = await menuOf(page, "Modules");
  const defaulted = await sentTo(page, "PATCH", settings, () => setAsDefault.click());
  expect([defaulted.answer.status(), defaulted.body]).toEqual([
    200,
    { navigation: { default_tab: "modules", hide_in_more_menu: [] } },
  ]);
  await expect(page.getByText("Default tab updated successfully.")).toBeVisible();
  // The second change is made to the tab bar nerve answered the first with.
  await tab("Views").click({ button: "right" });
  const hidden = await sentTo(page, "PATCH", settings, () =>
    page.getByRole("menuitem", { name: "Hide in more menu" }).click()
  );
  expect([hidden.answer.status(), hidden.body]).toEqual([
    200,
    { navigation: { default_tab: "modules", hide_in_more_menu: ["views"] } },
  ]);
  await expect(tab("Views")).toHaveCount(0);

  // He drags Ops, third in his sidebar, onto Docs, the first, by its handle: Ops goes a step before Docs.
  const order = sidebarOrder(page);
  await expect(order).toHaveText([/Docs/, /Web/, /Ops/]);
  const moved = await dragOnto(page, ops.id, docs.id);
  expect([moved.answer.status(), moved.body]).toEqual([200, { sort_order: 35535 }]);
  await expect(order).toHaveText([/Ops/, /Docs/, /Web/]);

  // After a refresh: the same order; Web opens on its modules, and its views are under more.
  await page.reload();
  await expect(order).toHaveText([/Ops/, /Docs/, /Web/]);
  await expect(inSidebar(page, web.id).locator("a").first()).toHaveAttribute(
    "href",
    `/${slug}/projects/${web.id}/modules`
  );
  // the header's tabs show (the first Modules: the header measures the tabs it shows by hidden copies of them)
  await expect(tab("Modules").first()).toBeVisible();
  await expect(tab("Views")).toHaveCount(0);
  expect(await stored(db, web.id, adminEmail)).toEqual([
    {
      preferences: { navigation: { default_tab: "modules", hide_in_more_menu: ["views"] } },
      sort_order: 55535,
      by: adminEmail,
    },
  ]);
  expect(await stored(db, ops.id, adminEmail)).toEqual([
    {
      preferences: { navigation: { default_tab: "work_items", hide_in_more_menu: [] } },
      sort_order: 35535,
      by: adminEmail,
    },
  ]);

  // The work items' page asks Plane's address of the filters, M4's (P8b spec §5), at each of its two loads.
  const filters = `GET /api/workspaces/${slug}/projects/${web.id}/user-properties/`;
  await expect.poll(() => watch.apiFailures).toEqual([`404 ${filters}`, `404 ${filters}`]);
  expect([watch.cspViolations, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [filters, filters], []]);
  await expectQuietConsole(page, watch, {
    warnings: [EMOJI_CHECK_WARNING, EMOJI_CHECK_WARNING],
    errors: [
      "Failed to load resource: the server responded with a status of 404 (Not Found)",
      "Failed to load resource: the server responded with a status of 404 (Not Found)",
    ],
  });

  // Gus is offered the same, on the tabs a guest has (not Modules): he makes views the tab Web opens on, by its tab's
  // menu, and drags Web, second in his sidebar, a step before Docs.
  const theGuest = await anotherBrowser(browser, baseURL ?? "", gus);
  const gusWatch = await watchPage(theGuest.page);
  await theGuest.page.goto(`/${slug}/projects/${web.id}/issues`);
  const gusDefault = await menuOf(theGuest.page, "Views");
  const gusDefaulted = await sentTo(theGuest.page, "PATCH", settings, () => gusDefault.click());
  expect([gusDefaulted.answer.status(), gusDefaulted.body]).toEqual([
    200,
    { navigation: { default_tab: "views", hide_in_more_menu: [] } },
  ]);
  await expect(sidebarOrder(theGuest.page)).toHaveText([/Docs/, /Web/]);
  const gusMoved = await dragOnto(theGuest.page, web.id, docs.id);
  expect([gusMoved.answer.status(), gusMoved.body]).toEqual([200, { sort_order: 45535 }]);
  await expect(sidebarOrder(theGuest.page)).toHaveText([/Web/, /Docs/]);
  expect(await stored(db, web.id, gusEmail)).toEqual([
    { preferences: { navigation: { default_tab: "views", hide_in_more_menu: [] } }, sort_order: 45535, by: gusEmail },
  ]);
  await expect.poll(() => gusWatch.apiFailures).toEqual([`404 ${filters}`]);
  expect([gusWatch.cspViolations, gusWatch.oldApiRequests, gusWatch.pageErrors]).toEqual([[], [filters], []]);
  await expectQuietConsole(theGuest.page, gusWatch, {
    warnings: [EMOJI_CHECK_WARNING],
    errors: ["Failed to load resource: the server responded with a status of 404 (Not Found)"],
  });
  await theGuest.close();
});
