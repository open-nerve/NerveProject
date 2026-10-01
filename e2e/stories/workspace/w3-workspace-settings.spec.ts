import { createProject, createWorkspace, invite, inviteAndAccept, slugFor, type Workspace } from "../../fixtures/api";
import { expectProjectCreated } from "../../fixtures/assert/project";
import {
  expectInvitations,
  expectMembership,
  expectPreferences,
  expectWorkspaceDeleted,
} from "../../fixtures/assert/workspace";
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// W3, the workspace's settings (M3 design 2). The page version, with the
// session switch of 7.1, comes with the general page (P9).

test("W3 (API): the admin changes the workspace and deletes it with its members, invitations, settings and projects at one moment; a member may do neither, and the slug never changes", async ({
  api,
  db,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = (await createPAT(api, (await register(api, adminEmail)).access_token)).token;
  const memberEmail = emailFor(testInfo, "member");
  const member = (await createPAT(api, (await register(api, memberEmail)).access_token)).token;
  const declinerEmail = emailFor(testInfo, "decliner");
  const decliner = (await register(api, declinerEmail)).access_token;
  const inviteeEmail = emailFor(testInfo, "invitee");
  const slug = slugFor(testInfo);
  const other = slugFor(testInfo, "other");
  // Two workspaces alike, each with the member, a pending and a declined invitation, the member's settings and a
  // project the admin created with the member its lead; only Acme is deleted. Both exist before either is
  // furnished, and the member is Other's admin and Acme's member: each answer, role and row must be the
  // workspace's own, whichever row the database reads first.
  const [adminId, memberId] = await Promise.all([admin, member].map((token) => accountId(api, token)));
  const web = { name: "Web", identifier: "WEB", description: "", network: 2, timezone: "UTC", logo_props: {} };
  const otherWorkspace = await createWorkspace(api, admin, { name: "Other", slug: other });
  const acme = await createWorkspace(api, admin, { name: "Acme", slug });
  const tokens: string[] = [];
  const furnish = async ({ slug: target, id }: Workspace, memberRole: 15 | 20) => {
    expect(await inviteAndAccept(api, admin, target, { email: memberEmail, token: member }, memberRole)).toMatchObject({
      slug: target,
      role: memberRole,
      total_members: 2,
    });
    const created = await invite(api, admin, target, [
      { email: inviteeEmail, role: 5 },
      { email: declinerEmail, role: 15 },
    ]);
    tokens.push(...created.map((i) => i.token));
    const [, toDecline] = created;
    if (!toDecline) {
      throw new Error(`the invitations to ${target} were not created`);
    }
    const declined = await api.POST("/api/v0/workspace-invitations/{invitation_id}/decline", {
      params: { path: { invitation_id: toDecline.id } },
      body: { token: toDecline.token },
      headers: bearer(decliner),
    });
    expect(declined.response.status).toBe(204);
    const settings = await api.PATCH("/api/v0/me/workspaces/{slug}/preferences", {
      params: { path: { slug: target } },
      body: { navigation_project_limit: 3 },
      headers: bearer(member),
    });
    expect(settings.response.status).toBe(200);
    // The answer is the workspace's own new project as its creator sees it, first in both admins' sidebars.
    expect(
      await createProject(api, admin, target, { name: "Web", identifier: "web", project_lead_id: memberId })
    ).toMatchObject({
      workspace_id: id,
      identifier: "WEB",
      member_role: 20,
      sort_order: 65535,
      member_ids: [adminId, memberId],
    });
  };
  await furnish(otherWorkspace, 20);
  await furnish(acme, 15);
  // The member reads Other as its admin, whatever becomes of Acme.
  const readOther = async () => {
    const read = await api.GET("/api/v0/workspaces/{slug}", {
      params: { path: { slug: other } },
      headers: bearer(member),
    });
    expect([read.response.status, read.data]).toEqual([
      200,
      expect.objectContaining({ name: "Other", slug: other, role: 20, total_members: 2 }),
    ]);
  };

  // A second admin changes it, so that the deletion must record its own author.
  const coAdminEmail = emailFor(testInfo, "co-admin");
  const coAdmin = (await createPAT(api, (await register(api, coAdminEmail)).access_token)).token;
  await inviteAndAccept(api, admin, slug, { email: coAdminEmail, token: coAdmin }, 20);
  const renamed = await api.PATCH("/api/v0/workspaces/{slug}", {
    params: { path: { slug } },
    body: { name: "Acme Corp", organization_size: "11-50", timezone: "Europe/Berlin" },
    headers: bearer(coAdmin),
  });
  expect(renamed.response.status).toBe(200);
  expect(renamed.data).toMatchObject({
    slug,
    name: "Acme Corp",
    organization_size: "11-50",
    timezone: "Europe/Berlin",
    role: 20,
    total_members: 3,
  });
  expect(
    await db.query(
      `SELECT w.name, w.organization_size, w.timezone, w.updated_by_id = u.id AS updated_by_the_co_admin
         FROM workspaces w JOIN users u ON u.email = $2 WHERE w.slug = $1`,
      [slug, coAdminEmail]
    )
  ).toEqual([
    { name: "Acme Corp", organization_size: "11-50", timezone: "Europe/Berlin", updated_by_the_co_admin: true },
  ]);
  await readOther();

  // A member may not change it; a slug in the body is refused before anything is looked at.
  const byMember = await api.PATCH("/api/v0/workspaces/{slug}", {
    params: { path: { slug } },
    body: { name: "Mine" },
    headers: bearer(member),
  });
  expect(byMember.response.status).toBe(403);
  expect(byMember.error?.code).toBe("forbidden");
  const withSlug = await api.PATCH("/api/v0/workspaces/{slug}", {
    params: { path: { slug } },
    body: { slug: "other" } as never,
    headers: bearer(admin),
  });
  expect(withSlug.response.status).toBe(400);
  const memberDeletes = await api.DELETE("/api/v0/workspaces/{slug}", {
    params: { path: { slug } },
    headers: bearer(member),
  });
  expect(memberDeletes.response.status).toBe(403);
  expect(memberDeletes.error?.code).toBe("forbidden");

  const deleted = await api.DELETE("/api/v0/workspaces/{slug}", { params: { path: { slug } }, headers: bearer(admin) });
  expect(deleted.response.status).toBe(204);
  await expectWorkspaceDeleted(db, slug, adminEmail);
  // The invitations accepted before keep the moment they were answered; the declined one goes with the pending
  // one; the other workspace keeps everything.
  const invitations = (memberRole: number, withTheWorkspace: boolean) => [
    { email: memberEmail, role: memberRole, accepted: true, responded: true, deleted: true },
    { email: inviteeEmail, role: 5, accepted: false, responded: false, deleted: withTheWorkspace },
    { email: declinerEmail, role: 15, accepted: false, responded: true, deleted: withTheWorkspace },
  ];
  await expectInvitations(
    db,
    slug,
    adminEmail,
    [...invitations(15, true), { email: coAdminEmail, role: 20, accepted: true, responded: true, deleted: true }],
    tokens
  );
  await expectInvitations(db, other, adminEmail, invitations(20, false), tokens);
  await expectMembership(db, other, memberEmail, { role: 20, is_active: true });
  await expectProjectCreated(db, other, web, adminEmail, memberEmail, [
    { email: adminEmail, sort_order: 65535 },
    { email: memberEmail, sort_order: 65535 },
  ]);
  await expectPreferences(db, other, memberEmail, {
    navigation_control_preference: "ACCORDION",
    navigation_project_limit: 3,
  });
  await readOther();
  const gone = await Promise.all(
    [admin, member].map((token) =>
      api.GET("/api/v0/workspaces/{slug}", { params: { path: { slug } }, headers: bearer(token) })
    )
  );
  expect(gone.map((g) => [g.response.status, g.error?.code])).toEqual([
    [404, "workspace.not_found"],
    [404, "workspace.not_found"],
  ]);
});
