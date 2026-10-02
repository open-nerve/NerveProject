import {
  addProjectMembers,
  amidAnotherWorkspace,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
  type Api,
  type ProjectPreferences,
  type ProjectPreferencesUpdate,
} from "../../fixtures/api";
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import type { Database } from "../../fixtures/db";
import { expect, test } from "../../fixtures/test";

// P8, a member's display settings in a project (M3 design 2, 3.18). The
// page version comes with the project's header and the sidebar (P10).

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
