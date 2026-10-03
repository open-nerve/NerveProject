import {
  addProjectMembers,
  amidAnotherWorkspace,
  createProject,
  createWorkspace,
  inviteAndAccept,
  membershipOf,
  slugFor,
  type Api,
  type ProjectMemberNew,
} from "../../fixtures/api";
import { expectMembers, type MemberRow } from "../../fixtures/assert/project";
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// P5, a project's members (M3 design 2, 3.5, 3.7): adding them, changing a
// role, removing a member, leaving. The page version comes with the
// project's members page (P10).

/** A write's answer: its status, and the problem's code and fields for a refusal. */
interface Answer {
  status: number;
  code?: string;
  errors?: { field: string; code: string }[];
}

function answer(
  response: Response,
  error?: { code: string; errors?: { field: string; code: string }[] | null } | null
): Answer {
  return error
    ? {
        status: response.status,
        code: error.code,
        errors: error.errors?.map((e) => ({ field: e.field, code: e.code })),
      }
    : { status: response.status };
}

/** The writes on a project's members, each by the caller of token. */
function writes(api: Api, projectId: string) {
  return {
    add: async (token: string, members: ProjectMemberNew[]) => {
      const { error, response } = await api.POST("/api/v0/projects/{project_id}/members", {
        params: { path: { project_id: projectId } },
        body: { members },
        headers: bearer(token),
      });
      return answer(response, error);
    },
    change: async (token: string, membership: string, role: 5 | 15 | 20) => {
      const { error, response } = await api.PATCH("/api/v0/project-members/{project_member_id}", {
        params: { path: { project_member_id: membership } },
        body: { role },
        headers: bearer(token),
      });
      return answer(response, error);
    },
    remove: async (token: string, membership: string) => {
      const { error, response } = await api.DELETE("/api/v0/project-members/{project_member_id}", {
        params: { path: { project_member_id: membership } },
        headers: bearer(token),
      });
      return answer(response, error);
    },
    leave: async (token: string) => {
      const { error, response } = await api.POST("/api/v0/projects/{project_id}/leave", {
        params: { path: { project_id: projectId } },
        headers: bearer(token),
      });
      return answer(response, error);
    },
  };
}

test("P5 (API): the admin adds a member and a guest at once, and cannot leave, the only admin; another admin makes the member a guest, removes the guest, and removes a third admin, who joins again as a member, his row back; a member leaves, his membership of the workspace's other project kept, and the workspace's making him a guest makes his ended membership a guest's; an add of one who is no workspace member, of a workspace guest or admin as a member, a member's change of a role and an admin's change of another admin's change nothing", async ({
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
  // acme's members mia, pam, ray and tom, its guest gus, its other admin wanda; olga is no member of it.
  const [mia, gus, pam, ray, tom, wanda, olga] = await Promise.all(
    ["mia", "gus", "pam", "ray", "tom", "wanda", "olga"].map(account)
  );
  if (!mia || !gus || !pam || !ray || !tom || !wanda || !olga) {
    throw new Error("the accounts were not registered");
  }
  await Promise.all(
    (
      [
        [mia, 15],
        [gus, 5],
        [pam, 15],
        [ray, 15],
        [tom, 15],
        [wanda, 20],
      ] as const
    ).map(([who, role]) => inviteAndAccept(api, admin.token, slug, who, role))
  );
  const web = await amidAnotherWorkspace(api, admin.token, testInfo, () =>
    createProject(api, admin.token, slug, { name: "Web", identifier: "WEB" })
  );
  const { add, change, remove, leave } = writes(api, web.id);
  // Ops, acme's other project: wanda its other admin, none of Web's.
  const ops = await createProject(api, admin.token, slug, { name: "Ops", identifier: "OPS" });
  await addProjectMembers(api, admin.token, ops.id, [{ member_id: wanda.id, role: 20 }]);
  // Each membership of Web or Ops, as expectMembers reads it, with its display settings, the admin's writing, as the
  // adds and the creations made them: at 65535, or at 55535 in Ops for the admin and tom, whose second project of acme
  // it is (10000 before the first, M3 design 3.18).
  const row = (
    who: { email: string },
    role: number,
    is_active: boolean,
    by: { email: string },
    sort_order = 65535
  ): MemberRow => ({
    email: who.email,
    role,
    is_active,
    by: by.email,
    sort_order,
    settings_by: admin.email,
  });
  let members = [row(admin, 20, true, admin)];
  let opsMembers = [row(admin, 20, true, admin, 55535), row(wanda, 20, true, admin)];
  await expectMembers(db, ops.id, opsMembers);

  // Refused, each adding nothing: olga, no member of acme; gus, its guest, as a member; wanda, its admin, as a member.
  expect(
    [
      await add(admin.token, [{ member_id: olga.id, role: 15 }]),
      await add(admin.token, [{ member_id: gus.id, role: 15 }]),
      await add(admin.token, [{ member_id: wanda.id, role: 15 }]),
    ],
    "the adds of olga, of gus and of wanda as members"
  ).toEqual([
    { status: 422, code: "validation_failed", errors: [{ field: "members[0].member_id", code: "not_allowed" }] },
    { status: 422, code: "validation_failed", errors: [{ field: "members[0].role", code: "not_allowed" }] },
    { status: 422, code: "validation_failed", errors: [{ field: "members[0].role", code: "not_allowed" }] },
  ]);
  await expectMembers(db, web.id, members);

  // The admin adds mia as a member and gus as a guest, at once: two memberships and two display settings, his.
  const added = await addProjectMembers(api, admin.token, web.id, [
    { member_id: mia.id, role: 15 },
    { member_id: gus.id, role: 5 },
  ]);
  expect(added.map((m) => [m.member_id, m.role])).toEqual([
    [mia.id, 15],
    [gus.id, 5],
  ]);
  const [mias, guss] = added.map((m) => m.id);
  if (!mias || !guss) {
    throw new Error("the add answered no memberships");
  }
  members = [...members, row(mia, 15, true, admin), row(gus, 5, true, admin)];
  await expectMembers(db, web.id, members);

  // Web's only admin cannot leave it; mia, its member, cannot change gus's role.
  expect(
    [await leave(admin.token), await change(mia.token, guss, 5)],
    "the admin's leaving, mia's change of gus"
  ).toEqual([
    { status: 409, code: "project.sole_admin" },
    { status: 403, code: "forbidden" },
  ]);
  await expectMembers(db, web.id, members);

  // The admin adds pam and ray as admins, tom as a member: pam, who is no workspace admin, cannot change ray's role,
  // another admin's.
  const more = await addProjectMembers(api, admin.token, web.id, [
    { member_id: pam.id, role: 20 },
    { member_id: ray.id, role: 20 },
    { member_id: tom.id, role: 15 },
  ]);
  const rays = more[1]?.id;
  if (!rays) {
    throw new Error("the add answered no membership of ray");
  }
  members = [...members, row(pam, 20, true, admin), row(ray, 20, true, admin), row(tom, 15, true, admin)];
  await expectMembers(db, web.id, members);
  // tom is Ops's member too.
  await addProjectMembers(api, admin.token, ops.id, [{ member_id: tom.id, role: 15 }]);
  opsMembers = [...opsMembers, row(tom, 15, true, admin, 55535)];
  await expectMembers(db, ops.id, opsMembers);
  expect(await change(pam.token, rays, 15), "pam's change of ray, an admin").toEqual({
    status: 403,
    code: "project.role_too_high",
  });
  await expectMembers(db, web.id, members);

  // pam makes mia a guest and removes gus: each row written by pam, gus's ended with its role and his display
  // settings kept.
  expect(
    [await change(pam.token, mias, 5), await remove(pam.token, guss)],
    "pam's change of mia, her removal of gus"
  ).toEqual([{ status: 200 }, { status: 204 }]);
  members = members.map((m) =>
    m.email === mia.email ? row(mia, 5, true, pam) : m.email === gus.email ? row(gus, 5, false, pam) : m
  );
  await expectMembers(db, web.id, members);

  // tom leaves Web: his membership ends, by him; his membership of Ops stays. The admin then makes him acme's guest:
  // his ended membership of Web becomes a guest's too, and so does his membership of Ops (M3 design 2, W7).
  expect(await leave(tom.token), "tom's leaving").toEqual({ status: 204 });
  members = members.map((m) => (m.email === tom.email ? row(tom, 15, false, tom) : m));
  await expectMembers(db, web.id, members);
  await expectMembers(db, ops.id, opsMembers);
  const demoted = await api.PATCH("/api/v0/workspace-members/{workspace_member_id}", {
    params: { path: { workspace_member_id: await membershipOf(api, admin.token, slug, tom.id) } },
    body: { role: 5 },
    headers: bearer(admin.token),
  });
  expect(demoted.response.status, "the admin's making tom acme's guest").toBe(200);
  members = members.map((m) => (m.email === tom.email ? row(tom, 5, false, admin) : m));
  await expectMembers(db, web.id, members);
  opsMembers = opsMembers.map((m) => (m.email === tom.email ? row(tom, 5, true, admin, 55535) : m));
  await expectMembers(db, ops.id, opsMembers);

  // pam removes ray, another admin and a member of acme; he joins Web again: his row is back, a member's, the 20 it
  // kept no more than his workspace role, 15 (M3 design 3.5).
  expect(await remove(pam.token, rays), "pam's removal of ray").toEqual({ status: 204 });
  members = members.map((m) => (m.email === ray.email ? row(ray, 20, false, pam) : m));
  await expectMembers(db, web.id, members);
  const joined = await api.POST("/api/v0/projects/{project_id}/join", {
    params: { path: { project_id: web.id } },
    headers: bearer(ray.token),
  });
  expect([joined.response.status, joined.data?.member_role], "ray's joining Web again").toEqual([200, 15]);
  members = members.map((m) => (m.email === ray.email ? row(ray, 15, true, ray) : m));
  await expectMembers(db, web.id, members);
  const listed = await api.GET("/api/v0/projects/{project_id}/members", {
    params: { path: { project_id: web.id } },
    headers: bearer(admin.token),
  });
  expect(
    listed.data?.data.map((m) => [m.member_id, m.role, m.id === rays]).toSorted(),
    "Web's members, ray's membership his row of before"
  ).toEqual(
    (
      [
        [admin.id, 20, false],
        [mia.id, 5, false],
        [pam.id, 20, false],
        [ray.id, 15, true],
      ] as const
    ).toSorted()
  );
});
