import {
  amidAnotherWorkspace,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
  type Api,
} from "../../fixtures/api";
import { expectMember, expectProjectCreated, expectProjectDeleted } from "../../fixtures/assert/project";
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import type { Database } from "../../fixtures/db";
import { expect, test } from "../../fixtures/test";

// P4, archiving, unarchiving and deleting a project (M3 design 2, 3.6,
// 3.19). The page version comes with the project's settings pages (P10).

/** The names of the projects of slug that the caller of token lists; archived: the archived ones. */
async function listed(api: Api, token: string, slug: string, archived = false): Promise<string[]> {
  const { data, error, response } = await api.GET("/api/v0/workspaces/{slug}/projects", {
    params: { path: { slug }, query: { archived } },
    headers: bearer(token),
  });
  expect(response.status, `list ${slug}'s projects: ${JSON.stringify(error)}`).toBe(200);
  return (data?.data ?? []).map((p) => p.name);
}

/** POST /api/v0/projects/{project_id}/{action}, which must be 200: the project's archived_at. */
async function act(api: Api, token: string, id: string, action: "archive" | "unarchive"): Promise<string | null> {
  const init = { params: { path: { project_id: id } }, headers: bearer(token) };
  const { data, error, response } =
    action === "archive"
      ? await api.POST("/api/v0/projects/{project_id}/archive", init)
      : await api.POST("/api/v0/projects/{project_id}/unarchive", init);
  expect(response.status, `${action} ${id}: ${JSON.stringify(error)}`).toBe(200);
  return data?.archived_at ?? null;
}

/**
 * Every operation on the project of id by the caller of token, each once: reading, changing, archiving, unarchiving
 * and deleting it, listing and adding its members (the account of memberId), joining it, reading and changing his
 * display settings in it. Each answer's status and code, in that order.
 */
async function onProject(api: Api, token: string, id: string, memberId: string): Promise<unknown[]> {
  const init = { params: { path: { project_id: id } }, headers: bearer(token) };
  const answers = await Promise.all([
    api.GET("/api/v0/projects/{project_id}", init),
    api.PATCH("/api/v0/projects/{project_id}", { ...init, body: { name: "Site" } }),
    api.POST("/api/v0/projects/{project_id}/archive", init),
    api.POST("/api/v0/projects/{project_id}/unarchive", init),
    api.DELETE("/api/v0/projects/{project_id}", init),
    api.GET("/api/v0/projects/{project_id}/members", init),
    api.POST("/api/v0/projects/{project_id}/members", {
      ...init,
      body: { members: [{ member_id: memberId, role: 15 }] },
    }),
    api.POST("/api/v0/projects/{project_id}/join", init),
    api.GET("/api/v0/me/projects/{project_id}/preferences", init),
    api.PATCH("/api/v0/me/projects/{project_id}/preferences", { ...init, body: { sort_order: 1 } }),
  ]);
  return answers.map(({ error, response }) => ({ status: response.status, code: error?.code }));
}

/** Whether the project is archived as stored, and who changed it last. */
async function archiving(db: Database, id: string): Promise<unknown> {
  const [row] = await db.query(
    `SELECT p.archived_at IS NOT NULL AS archived, p.archived_at IS NOT DISTINCT FROM p.updated_at OR p.archived_at IS NULL AS at_its_change,
            u.email AS by
       FROM projects p JOIN users u ON u.id = p.updated_by_id WHERE p.id = $1`,
    [id]
  );
  return row;
}

test("P4 (API): the admin archives a project, which leaves the list for the archived ones and cannot be changed; unarchives it and archives it again; then deletes it with its members, settings and states at one moment, after which it is not found and its identifier is free", async ({
  api,
  db,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = (await createPAT(api, (await register(api, adminEmail)).access_token)).token;
  const memberEmail = emailFor(testInfo, "member");
  const member = (await createPAT(api, (await register(api, memberEmail)).access_token)).token;
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin, { name: "Acme", slug, timezone: "UTC" });
  await inviteAndAccept(api, admin, slug, { email: memberEmail, token: member }, 15);
  const memberId = await accountId(api, member);
  // Web goes before Ops in the admin's sidebar: Ops 65535, Web 55535. The member joins Web, which is public, so that
  // each table under it has his row too, and his membership and his display settings are his own writing, which the
  // deletion must write again as the admin's.
  const { ops, web } = await amidAnotherWorkspace(api, admin, testInfo, async () => {
    const made = {
      ops: await createProject(api, admin, slug, { name: "Ops", identifier: "OPS" }),
      web: await createProject(api, admin, slug, { name: "Web", identifier: "WEB" }),
    };
    const joined = await api.POST("/api/v0/projects/{project_id}/join", {
      params: { path: { project_id: made.web.id } },
      headers: bearer(member),
    });
    expect(joined.response.status, `the member joins Web: ${JSON.stringify(joined.error)}`).toBe(200);
    return made;
  });
  await expectMember(db, web.id, memberEmail, { role: 15, is_active: true, sort_order: 65535, by: memberEmail });

  expect(await act(api, admin, web.id, "archive")).toEqual(expect.any(String));
  expect(await archiving(db, web.id)).toEqual({ archived: true, at_its_change: true, by: adminEmail });
  expect(await listed(api, admin, slug)).toEqual(["Ops"]);
  expect(await listed(api, admin, slug, true)).toEqual(["Web"]);
  const refused = await api.PATCH("/api/v0/projects/{project_id}", {
    params: { path: { project_id: web.id } },
    body: { name: "Site" },
    headers: bearer(admin),
  });
  expect({ status: refused.response.status, code: refused.error?.code }).toEqual({
    status: 409,
    code: "project.archived",
  });

  expect(await act(api, admin, web.id, "unarchive")).toBeNull();
  expect(await archiving(db, web.id)).toEqual({ archived: false, at_its_change: true, by: adminEmail });
  expect(await listed(api, admin, slug)).toEqual(["Web", "Ops"]);
  expect(await listed(api, admin, slug, true)).toEqual([]);
  expect(await act(api, admin, web.id, "archive")).toEqual(expect.any(String));
  // The member drags the archived Web in his sidebar, as in any project (M3 design 3.19).
  const dragged = await api.PATCH("/api/v0/me/projects/{project_id}/preferences", {
    params: { path: { project_id: web.id } },
    body: { sort_order: 25535 },
    headers: bearer(member),
  });
  expect(dragged.response.status, `the member drags Web: ${JSON.stringify(dragged.error)}`).toBe(200);
  await expectMember(db, web.id, memberEmail, { role: 15, is_active: true, sort_order: 25535, by: memberEmail });

  // An archived project is deleted as any other.
  const deleted = await api.DELETE("/api/v0/projects/{project_id}", {
    params: { path: { project_id: web.id } },
    headers: bearer(admin),
  });
  expect(deleted.response.status, `delete Web: ${JSON.stringify(deleted.error)}`).toBe(204);
  await expectProjectDeleted(db, web.id, adminEmail);
  // Every operation on it is refused as on a project that does not exist, its admin's too.
  expect(await onProject(api, admin, web.id, memberId)).toEqual(
    Array(10).fill({ status: 404, code: "project.not_found" })
  );
  expect(await listed(api, admin, slug)).toEqual(["Ops"]);
  expect(await listed(api, admin, slug, true)).toEqual([]);

  // WEB is free again. The new Web goes before Ops, at 55535: the deleted Web's place, 55535, no longer counts.
  const again = await createProject(api, admin, slug, { name: "Web", identifier: "WEB" });
  const row = { name: "Web", identifier: "WEB", description: "", network: 2, timezone: "UTC", logo_props: {} };
  expect(await expectProjectCreated(db, slug, row, adminEmail, null, [{ email: adminEmail, sort_order: 55535 }])).toBe(
    again.id
  );
  expect(
    await expectProjectCreated(db, slug, { ...row, name: "Ops", identifier: "OPS" }, adminEmail, null, [
      { email: adminEmail, sort_order: 65535 },
    ])
  ).toBe(ops.id);
});
