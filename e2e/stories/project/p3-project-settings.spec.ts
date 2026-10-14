import {
  addProjectMembers,
  amidAnotherWorkspace,
  changeProject,
  createProject,
  createWorkspace,
  inviteAndAccept,
  projectMemberWrites,
  slugFor,
} from "../../fixtures/api";
import { expectMember, projectSettingsOf } from "../../fixtures/assert/project";
import { accountId, bearer, emailFor, registeredPAT } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// P3, a project's settings (M3 design 2, 3.4, 3.5, 3.19, 7.6). The page version is
// p3-project-settings-page.spec.ts.

test("P3 (API): the project's admin adds a member and a guest, whom the member lists with him, then changes every setting, the member its lead and default assignee; its member may not; a lead or default assignee who is its guest or no member, and archive_in 13, change nothing", async ({
  api,
  db,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = (await registeredPAT(api, adminEmail)).token;
  const memberEmail = emailFor(testInfo, "member");
  const member = (await registeredPAT(api, memberEmail)).token;
  const guestEmail = emailFor(testInfo, "guest");
  const guest = (await registeredPAT(api, guestEmail)).token;
  const otherEmail = emailFor(testInfo, "other");
  const other = (await registeredPAT(api, otherEmail)).token;
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
  expect(await projectMemberWrites(api, web.id).add(member, [{ member_id: otherId, role: 15 }])).toEqual({
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
    await changeProject(api, admin, web.id, {
      ...settings,
      identifier: "site",
      project_lead_id: memberId,
      default_assignee_id: memberId,
    })
  ).toMatchObject({ status: 200, project: { ...settings, project_lead_id: memberId, default_assignee_id: memberId } });
  const changed = { ...settings, lead: memberEmail, default_assignee: memberEmail, by: adminEmail };
  expect(await projectSettingsOf(db, web.id)).toEqual(changed);

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
    await Promise.all(refusals.map(({ token, body }) => changeProject(api, token, web.id, body))),
    "the refusals"
  ).toEqual(refusals.map((r) => r.want));
  expect(await projectSettingsOf(db, web.id)).toEqual(changed);

  // Both cleared with null.
  expect(await changeProject(api, admin, web.id, { project_lead_id: null, default_assignee_id: null })).toMatchObject({
    status: 200,
    project: { project_lead_id: null, default_assignee_id: null },
  });
  expect(await projectSettingsOf(db, web.id)).toEqual({ ...changed, lead: null, default_assignee: null });
});
