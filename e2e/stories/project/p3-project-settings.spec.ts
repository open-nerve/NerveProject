import {
  addProjectMembers,
  amidAnotherWorkspace,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
  type Api,
  type ProjectUpdate,
} from "../../fixtures/api";
import { expectMember } from "../../fixtures/assert/project";
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import type { Database } from "../../fixtures/db";
import { expect, test } from "../../fixtures/test";

// P3, a project's settings (M3 design 2, 3.4, 3.5, 3.19). The page version
// comes with the project's settings pages (P10).

/** The answer of PATCH /api/v0/projects/{project_id}: its status, the project, or the problem's code and fields. */
async function change(api: Api, token: string, id: string, body: ProjectUpdate) {
  const { data, error, response } = await api.PATCH("/api/v0/projects/{project_id}", {
    params: { path: { project_id: id } },
    body,
    headers: bearer(token),
  });
  return data
    ? { status: response.status, project: data }
    : {
        status: response.status,
        code: error?.code,
        errors: error?.errors?.map((e) => ({ field: e.field, code: e.code })),
      };
}

/** The project's settings as stored, its lead and default assignee by address, and who changed it last. */
async function stored(db: Database, id: string): Promise<unknown> {
  const [row] = await db.query(
    `SELECT p.name, p.identifier, p.description, p.network, p.timezone, p.logo_props, p.cycle_view, p.module_view,
            p.issue_views_view, p.intake_view, p.guest_view_all_features, p.archive_in, l.email AS lead,
            a.email AS default_assignee, u.email AS by
       FROM projects p
       JOIN users u ON u.id = p.updated_by_id
       LEFT JOIN users l ON l.id = p.project_lead_id
       LEFT JOIN users a ON a.id = p.default_assignee_id
      WHERE p.id = $1`,
    [id]
  );
  return row;
}

test("P3 (API): the project's admin adds a member and a guest, whom the member lists with him, then changes every setting, the member its lead and default assignee; its member may not; a lead or default assignee who is its guest or no member, and archive_in 13, change nothing", async ({
  api,
  db,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = (await createPAT(api, (await register(api, adminEmail)).access_token)).token;
  const memberEmail = emailFor(testInfo, "member");
  const member = (await createPAT(api, (await register(api, memberEmail)).access_token)).token;
  const guestEmail = emailFor(testInfo, "guest");
  const guest = (await createPAT(api, (await register(api, guestEmail)).access_token)).token;
  const otherEmail = emailFor(testInfo, "other");
  const other = (await createPAT(api, (await register(api, otherEmail)).access_token)).token;
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin, { name: "Acme", slug });
  await inviteAndAccept(api, admin, slug, { email: memberEmail, token: member }, 15);
  await inviteAndAccept(api, admin, slug, { email: guestEmail, token: guest }, 5);
  await inviteAndAccept(api, admin, slug, { email: otherEmail, token: other }, 15);
  const memberId = await accountId(api, member);
  const guestId = await accountId(api, guest);
  const otherId = await accountId(api, other);
  // other is a member of Acme and of Ops, not of Web: a membership of another project makes no one Web's assignee.
  // The member is Ops' member too, at 65535 in his sidebar: Web, added later, goes before it.
  const web = await amidAnotherWorkspace(api, admin, testInfo, async () => {
    const ops = await createProject(api, admin, slug, { name: "Ops", identifier: "OPS" });
    await addProjectMembers(api, admin, ops.id, [
      { member_id: otherId, role: 15 },
      { member_id: memberId, role: 15 },
    ]);
    return createProject(api, admin, slug, { name: "Web", identifier: "WEB" });
  });

  const added = await addProjectMembers(api, admin, web.id, [
    { member_id: memberId, role: 15 },
    { member_id: guestId, role: 5 },
  ]);
  expect(added.map((m) => ({ project_id: m.project_id, member_id: m.member_id, role: m.role }))).toEqual([
    { project_id: web.id, member_id: memberId, role: 15 },
    { project_id: web.id, member_id: guestId, role: 5 },
  ]);
  await expectMember(db, web.id, memberEmail, { role: 15, is_active: true, sort_order: 55535, by: adminEmail });
  await expectMember(db, web.id, guestEmail, { role: 5, is_active: true, sort_order: 65535, by: adminEmail });
  // Web lists exactly its own three; other, Ops' member, is not among them.
  const members = await api.GET("/api/v0/projects/{project_id}/members", {
    params: { path: { project_id: web.id } },
    headers: bearer(member),
  });
  expect(members.response.status, `Web's members: ${JSON.stringify(members.error)}`).toBe(200);
  expect(
    members.data?.data.map((m) => ({ member_id: m.member_id, role: m.role })).toSorted((a, b) => b.role - a.role)
  ).toEqual([
    { member_id: await accountId(api, admin), role: 20 },
    { member_id: memberId, role: 15 },
    { member_id: guestId, role: 5 },
  ]);
  // Web's member may not add to it: refused, nothing written.
  const addedByMember = await api.POST("/api/v0/projects/{project_id}/members", {
    params: { path: { project_id: web.id } },
    body: { members: [{ member_id: otherId, role: 15 }] },
    headers: bearer(member),
  });
  expect({ status: addedByMember.response.status, code: addedByMember.error?.code }).toEqual({
    status: 403,
    code: "forbidden",
  });
  await expectMember(db, web.id, otherEmail, null);

  const logo = { in_use: "icon", icon: { name: "rocket", color: "#46A758" } } as const;
  const settings = {
    name: "Site",
    identifier: "SITE",
    description: "The site",
    network: 0 as const,
    timezone: "Asia/Shanghai",
    logo_props: logo,
    cycle_view: false,
    module_view: false,
    issue_views_view: false,
    intake_view: true,
    guest_view_all_features: true,
    archive_in: 3,
  };
  expect(
    await change(api, admin, web.id, {
      ...settings,
      identifier: "site",
      project_lead_id: memberId,
      default_assignee_id: memberId,
    })
  ).toMatchObject({ status: 200, project: { ...settings, project_lead_id: memberId, default_assignee_id: memberId } });
  const changed = { ...settings, lead: memberEmail, default_assignee: memberEmail, by: adminEmail };
  expect(await stored(db, web.id)).toEqual(changed);

  // Each refused, all at once: none writes.
  const refusals = [
    { token: member, body: { name: "Mine" }, want: { status: 403, code: "forbidden" } },
    {
      token: admin,
      body: { archive_in: 13 },
      want: { status: 422, code: "validation_failed", errors: [{ field: "archive_in", code: "out_of_range" }] },
    },
    {
      token: admin,
      body: { project_lead_id: guestId },
      want: { status: 422, code: "validation_failed", errors: [{ field: "project_lead_id", code: "not_allowed" }] },
    },
    {
      token: admin,
      body: { default_assignee_id: otherId },
      want: { status: 422, code: "validation_failed", errors: [{ field: "default_assignee_id", code: "not_allowed" }] },
    },
  ];
  expect(
    await Promise.all(refusals.map(({ token, body }) => change(api, token, web.id, body))),
    "the refusals"
  ).toEqual(refusals.map((r) => r.want));
  expect(await stored(db, web.id)).toEqual(changed);

  // Both cleared with null.
  expect(await change(api, admin, web.id, { project_lead_id: null, default_assignee_id: null })).toMatchObject({
    status: 200,
    project: { project_lead_id: null, default_assignee_id: null },
  });
  expect(await stored(db, web.id)).toEqual({ ...changed, lead: null, default_assignee: null });
});
