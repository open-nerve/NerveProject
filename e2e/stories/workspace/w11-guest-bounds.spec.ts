import {
  addProjectMembers,
  answer,
  createProject,
  createWorkspace,
  invite,
  inviteAndAccept,
  listMembers,
  slugFor,
} from "../../fixtures/api";
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// W11, a guest's bounds (M3 design 2, 9.2): the permission matrix's backend
// test holds every cell; this story samples four of them through the API.
// The page version is P11's.

test("W11 (API): a guest of a workspace may not list its invitations nor create a state in the project he is a guest of, does not see a private project he is not a member of, and reads no member's address, his own neither; none of it changes a row of the workspace's", async ({
  api,
  db,
}, testInfo) => {
  const account = async (label: string) => {
    const email = emailFor(testInfo, label);
    const token = (await createPAT(api, (await register(api, email)).access_token)).token;
    return { email, token, id: await accountId(api, token) };
  };
  const admin = await account("admin");
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  const gus = await account("gus");
  await inviteAndAccept(api, admin.token, slug, gus, 5);
  // Web, public, gus its guest; Secret, private, the admin's alone; an invitation to olga, pending.
  const web = await createProject(api, admin.token, slug, { name: "Web", identifier: "WEB" });
  await addProjectMembers(api, admin.token, web.id, [{ member_id: gus.id, role: 5 }]);
  const secret = await createProject(api, admin.token, slug, { name: "Secret", identifier: "SECRET", network: 0 });
  await invite(api, admin.token, slug, [{ email: emailFor(testInfo, "olga"), role: 15 }]);
  // The tables of the workspace's rows, whole: none of gus's calls writes one.
  const tables = async () => ({
    members: await db.query("SELECT * FROM workspace_members ORDER BY id"),
    invitations: await db.query("SELECT * FROM workspace_member_invites ORDER BY id"),
    projects: await db.query("SELECT * FROM projects ORDER BY id"),
    projectMembers: await db.query("SELECT * FROM project_members ORDER BY id"),
    states: await db.query("SELECT * FROM states ORDER BY id"),
  });
  const before = await tables();

  const invitations = await api.GET("/api/v0/workspaces/{slug}/invitations", {
    params: { path: { slug } },
    headers: bearer(gus.token),
  });
  const created = await api.POST("/api/v0/projects/{project_id}/states", {
    params: { path: { project_id: web.id } },
    body: { name: "Review", color: "#8B5CF6", group: "started" },
    headers: bearer(gus.token),
  });
  const read = await api.GET("/api/v0/projects/{project_id}", {
    params: { path: { project_id: secret.id } },
    headers: bearer(gus.token),
  });
  expect(
    [
      answer(invitations.response, invitations.error),
      answer(created.response, created.error),
      answer(read.response, read.error),
    ],
    "gus's listing of acme's invitations, his creation of a state in Web, his reading of Secret"
  ).toEqual([
    { status: 403, code: "forbidden" },
    { status: 403, code: "forbidden" },
    { status: 404, code: "project.not_found" },
  ]);

  // acme's members as gus lists them: no address, his own neither; the admin reads both.
  const addresses = async (token: string) =>
    (await listMembers(api, token, slug)).map((m) => [m.member.id, m.member.email]).toSorted();
  expect(await addresses(gus.token), "acme's members as gus lists them").toEqual(
    [
      [admin.id, null],
      [gus.id, null],
    ].toSorted()
  );
  expect(await addresses(admin.token), "acme's members as the admin lists them").toEqual(
    [
      [admin.id, admin.email],
      [gus.id, gus.email],
    ].toSorted()
  );
  expect(await tables(), "the tables after gus's calls").toEqual(before);
});
