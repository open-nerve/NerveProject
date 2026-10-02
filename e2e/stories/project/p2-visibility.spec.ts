import {
  addProjectMembers,
  amidAnotherWorkspace,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
  type Api,
  type Project,
} from "../../fixtures/api";
import { expectMember } from "../../fixtures/assert/project";
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// P2, the projects' list and who sees them, and joining (M3 design 2, 3.4,
// 3.5, 3.19). The page version, with the "join the project" screen, comes
// with the projects' pages (P10).

/** The names of the projects of slug that the caller of token lists, in their order; archived: the archived ones. */
async function listed(api: Api, token: string, slug: string, archived = false): Promise<string[]> {
  const { data, error, response } = await api.GET("/api/v0/workspaces/{slug}/projects", {
    params: { path: { slug }, query: { archived } },
    headers: bearer(token),
  });
  expect(response.status, `list ${slug}'s projects: ${JSON.stringify(error)}`).toBe(200);
  return (data?.data ?? []).map((p) => p.name);
}

/** The answer of POST /api/v0/projects/{project_id}/join: its status, and the project or the problem's code. */
async function join(
  api: Api,
  token: string,
  id: string
): Promise<{ status: number; project?: Project; code?: string }> {
  const { data, error, response } = await api.POST("/api/v0/projects/{project_id}/join", {
    params: { path: { project_id: id } },
    headers: bearer(token),
  });
  return data ? { status: response.status, project: data } : { status: response.status, code: error?.code };
}

test("P2 (API): the admin lists every project, a member the public ones and his own, a guest his own, none the archived ones unless asked; a member joins a public project as a member at 65535, not before his own, and again changes nothing; a member cannot join a private project, nor a guest a public one, nor a guest the project he is a member of", async ({
  api,
  db,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = (await createPAT(api, (await register(api, adminEmail)).access_token)).token;
  const memberEmail = emailFor(testInfo, "member");
  const member = (await createPAT(api, (await register(api, memberEmail)).access_token)).token;
  const guestEmail = emailFor(testInfo, "guest");
  const guest = (await createPAT(api, (await register(api, guestEmail)).access_token)).token;
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin, { name: "Acme", slug });
  await inviteAndAccept(api, admin, slug, { email: memberEmail, token: member }, 15);
  await inviteAndAccept(api, admin, slug, { email: guestEmail, token: guest }, 5);
  const adminId = await accountId(api, admin);
  const memberId = await accountId(api, member);
  const guestId = await accountId(api, guest);
  // Each new project goes first in its creator's sidebar: the admin's Old, Docs, Secret, Web. Old is archived; the
  // guest is Docs' guest. The member made Notes, private, of which the admin is no member: the member's own, at 65535.
  const { web, secret, docs, notes } = await amidAnotherWorkspace(api, admin, testInfo, async () => {
    const made = {
      web: await createProject(api, admin, slug, { name: "Web", identifier: "WEB", network: 2 }),
      secret: await createProject(api, admin, slug, { name: "Secret", identifier: "SEC", network: 0 }),
      docs: await createProject(api, admin, slug, { name: "Docs", identifier: "DOCS", network: 2 }),
      notes: await createProject(api, member, slug, { name: "Notes", identifier: "NOTES", network: 0 }),
    };
    const old = await createProject(api, admin, slug, { name: "Old", identifier: "OLD", network: 2 });
    const archived = await api.POST("/api/v0/projects/{project_id}/archive", {
      params: { path: { project_id: old.id } },
      headers: bearer(admin),
    });
    expect(archived.response.status, "archive Old").toBe(200);
    await addProjectMembers(api, admin, made.docs.id, [{ member_id: guestId, role: 5 }]);
    return made;
  });

  // Each caller's places first, then the projects he has none in, by name.
  expect(await listed(api, admin, slug)).toEqual(["Docs", "Secret", "Web", "Notes"]);
  expect(await listed(api, member, slug)).toEqual(["Notes", "Docs", "Web"]);
  expect(await listed(api, guest, slug)).toEqual(["Docs"]);
  expect(await listed(api, admin, slug, true)).toEqual(["Old"]);
  expect(await listed(api, member, slug, true)).toEqual(["Old"]);
  expect(await listed(api, guest, slug, true)).toEqual([]);
  // The admin sees Notes as he sees every project, though he is no member of it.
  const seen = await api.GET("/api/v0/projects/{project_id}", {
    params: { path: { project_id: notes.id } },
    headers: bearer(admin),
  });
  expect({ status: seen.response.status, project: seen.data }).toMatchObject({
    status: 200,
    project: { id: notes.id, member_role: null, sort_order: null, member_ids: [memberId] },
  });

  // The member joins Web: a member's membership, with his display settings, both by him, at 65535 beside Notes: a
  // joiner's place is the default one, not before his other projects, which would be 55535 (M3 design 3.18).
  const joined = await join(api, member, web.id);
  expect(joined).toMatchObject({ status: 200, project: { id: web.id, member_role: 15, sort_order: 65535 } });
  expect(joined.project?.member_ids?.toSorted()).toEqual([adminId, memberId].toSorted());
  const membership = { role: 15, is_active: true, sort_order: 65535, by: memberEmail };
  await expectMember(db, web.id, memberEmail, membership);
  expect(await listed(api, member, slug)).toEqual(["Notes", "Web", "Docs"]);
  // Joining again changes nothing.
  expect(await join(api, member, web.id)).toMatchObject({
    status: 200,
    project: { member_role: 15, sort_order: 65535 },
  });
  await expectMember(db, web.id, memberEmail, membership);

  // Refused, nothing written: the member does not see Secret, the guest does not see Web; the guest sees Docs, as
  // its guest, and is refused as one.
  expect(await join(api, member, secret.id)).toEqual({ status: 404, code: "project.not_found" });
  await expectMember(db, secret.id, memberEmail, null);
  expect(await join(api, guest, web.id)).toEqual({ status: 404, code: "project.not_found" });
  await expectMember(db, web.id, guestEmail, null);
  expect(await join(api, guest, docs.id)).toEqual({ status: 403, code: "forbidden" });
  await expectMember(db, docs.id, guestEmail, { role: 5, is_active: true, sort_order: 65535, by: adminEmail });
});
