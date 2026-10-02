import {
  accept,
  addProjectMembers,
  createProject,
  createWorkspace,
  invite,
  inviteAndAccept,
  listMembers,
  membershipOf,
  slugFor,
} from "../../fixtures/api";
import { expectMembership, expectMembershipEnded, expectWrittenLastBy } from "../../fixtures/assert/workspace";
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// W7, the members' management (M3 design 2). The page version comes with the members' page (P9).

test("W7 (API): the admin makes a member a guest in every project, removes another, ending her project memberships and no other invitation, and invites her back as a guest; nobody changes or removes his own membership, and the only admin cannot leave", async ({
  api,
  db,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = (await createPAT(api, (await register(api, adminEmail)).access_token)).token;
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin, { name: "Acme", slug });
  // carol's address had an invitation to acme, which the admin deleted before she joined.
  const carolEmail = emailFor(testInfo, "carol");
  const [revoked] = await invite(api, admin, slug, [{ email: carolEmail, role: 15 }]);
  if (!revoked) {
    throw new Error("the invitation to carol was not created");
  }
  const revoking = await api.DELETE("/api/v0/workspace-invitations/{invitation_id}", {
    params: { path: { invitation_id: revoked.id } },
    headers: bearer(admin),
  });
  expect(revoking.response.status).toBe(204);
  const people = Object.fromEntries(
    await Promise.all(
      ["bob", "carol", "dave", "erin"].map(async (name) => {
        const email = emailFor(testInfo, name);
        const token = (await createPAT(api, (await register(api, email)).access_token)).token;
        await inviteAndAccept(api, admin, slug, { email, token }, 15);
        return [name, { email, token, id: await accountId(api, token) }] as const;
      })
    )
  );
  const { bob, carol, dave, erin } = people as Record<
    "bob" | "carol" | "dave" | "erin",
    { email: string; token: string; id: string }
  >;
  // bob is the admin of a workspace of his own, Other, which invites carol; acme invites eve.
  const other = slugFor(testInfo, "other");
  await createWorkspace(api, bob.token, { name: "Other", slug: other });
  const [elsewhere] = await invite(api, bob.token, other, [{ email: carol.email, role: 15 }]);
  const [eves] = await invite(api, admin, slug, [{ email: emailFor(testInfo, "eve"), role: 5 }]);
  if (!elsewhere || !eves) {
    throw new Error("the invitations to carol in Other and to eve in acme were not created");
  }
  // bob and carol join Web; Ops is bob's own, carol its member; Docs is carol's own, dave its member, and erin was its
  // admin until she left acme.
  const join = async (projectId: string, token: string) =>
    (
      await api.POST("/api/v0/projects/{project_id}/join", {
        params: { path: { project_id: projectId } },
        headers: bearer(token),
      })
    ).response.status;
  const web = await createProject(api, admin, slug, { name: "Web", identifier: "WEB" });
  const ops = await createProject(api, bob.token, slug, { name: "Ops", identifier: "OPS" });
  const docs = await createProject(api, carol.token, slug, { name: "Docs", identifier: "DOCS" });
  expect(
    [
      await join(web.id, bob.token),
      await join(web.id, carol.token),
      await join(ops.id, carol.token),
      await join(docs.id, dave.token),
    ],
    "bob and carol join Web, carol Ops, dave Docs"
  ).toEqual([200, 200, 200, 200]);
  await addProjectMembers(api, carol.token, docs.id, [{ member_id: erin.id, role: 20 }]);
  await expectWrittenLastBy(db, slug, erin.email, { workspace: erin.email, DOCS: carol.email });
  const erinLeaves = await api.POST("/api/v0/workspaces/{slug}/leave", {
    params: { path: { slug } },
    headers: bearer(erin.token),
  });
  expect(erinLeaves.response.status, "erin leaves acme").toBe(204);
  const membership = (id: string) => membershipOf(api, admin, slug, id);
  const adminId = await accountId(api, admin);
  // The account of email's memberships of acme's projects, ended ones too: each one's role, whether it is active,
  // and who wrote it last.
  const projectRoles = (email: string) =>
    db.query(
      `SELECT p.identifier, m.role, m.is_active, b.email AS by
         FROM project_members m JOIN projects p ON p.id = m.project_id JOIN workspaces w ON w.id = p.workspace_id
         JOIN users u ON u.id = m.member_id JOIN users b ON b.id = m.updated_by_id
        WHERE w.slug = $1 AND u.email = $2 AND m.deleted_at IS NULL ORDER BY p.identifier COLLATE "C"`,
      [slug, email]
    );
  expect(await projectRoles(erin.email), "erin's membership of Docs, an admin's, ended by her leaving").toEqual([
    { identifier: "DOCS", role: 20, is_active: false, by: erin.email },
  ]);

  // Nobody changes or removes his own membership; the only admin cannot leave, though Other has an admin.
  const own = await membership(adminId);
  const ownChange = await api.PATCH("/api/v0/workspace-members/{workspace_member_id}", {
    params: { path: { workspace_member_id: own } },
    body: { role: 15 },
    headers: bearer(admin),
  });
  const ownRemoval = await api.DELETE("/api/v0/workspace-members/{workspace_member_id}", {
    params: { path: { workspace_member_id: own } },
    headers: bearer(admin),
  });
  const leaving = await api.POST("/api/v0/workspaces/{slug}/leave", {
    params: { path: { slug } },
    headers: bearer(admin),
  });
  expect(
    [ownChange, ownRemoval, leaving].map((r) => [r.response.status, r.error?.code]),
    "the admin's own change, his own removal, his leaving"
  ).toEqual([
    [409, "workspace.own_membership"],
    [409, "workspace.own_membership"],
    [409, "workspace.sole_admin"],
  ]);
  await expectMembership(db, slug, adminEmail, { role: 20, is_active: true });

  // bob becomes a guest, and a guest in each of his projects: Ops is left without an admin. He wrote each of those
  // rows last, so that the admin's writing them shows.
  await expectWrittenLastBy(db, slug, bob.email, { workspace: bob.email, OPS: bob.email, WEB: bob.email });
  expect(await projectRoles(bob.email), "bob's memberships of the projects, Ops's admin's").toEqual([
    { identifier: "OPS", role: 20, is_active: true, by: bob.email },
    { identifier: "WEB", role: 15, is_active: true, by: bob.email },
  ]);
  const demoted = await api.PATCH("/api/v0/workspace-members/{workspace_member_id}", {
    params: { path: { workspace_member_id: await membership(bob.id) } },
    body: { role: 5 },
    headers: bearer(admin),
  });
  expect(demoted.response.status).toBe(200);
  await expectMembership(db, slug, bob.email, { role: 5, is_active: true });
  await expectWrittenLastBy(db, slug, bob.email, { workspace: adminEmail, OPS: adminEmail, WEB: adminEmail });
  expect(await projectRoles(bob.email), "bob's memberships of the projects").toEqual([
    { identifier: "OPS", role: 5, is_active: true, by: adminEmail },
    { identifier: "WEB", role: 5, is_active: true, by: adminEmail },
  ]);

  // carol, Docs's only active admin, dave its member, cannot be removed until the admin joins Docs; then her
  // memberships end, their rows kept, and no invitation but a pending one to her address in acme is touched: none is.
  const carolsMembership = await membership(carol.id);
  const remove = () =>
    api.DELETE("/api/v0/workspace-members/{workspace_member_id}", {
      params: { path: { workspace_member_id: carolsMembership } },
      headers: bearer(admin),
    });
  const refused = await remove();
  expect([refused.response.status, refused.error?.code]).toEqual([409, "project.sole_admin"]);
  await expectMembership(db, slug, carol.email, { role: 15, is_active: true });
  expect(await join(docs.id, admin), "the admin joins Docs").toBe(200);
  const invitations = () =>
    db.query("SELECT to_jsonb(i) AS row FROM workspace_member_invites i WHERE i.id = ANY ($1) ORDER BY i.id", [
      [revoked.id, elsewhere.id, eves.id],
    ]);
  const untouched = await invitations();
  expect(
    await db.query(
      `SELECT email FROM workspace_member_invites
        WHERE id = ANY ($1) AND responded_at IS NULL AND deleted_at IS NULL ORDER BY email`,
      [[elsewhere.id, eves.id]]
    ),
    "the invitations to carol in Other and to eve in acme, pending before the removal"
  ).toEqual([{ email: carol.email }, { email: emailFor(testInfo, "eve") }]);
  await expectWrittenLastBy(db, slug, carol.email, {
    workspace: carol.email,
    DOCS: carol.email,
    OPS: carol.email,
    WEB: carol.email,
  });
  expect((await remove()).response.status).toBe(204);
  await expectMembershipEnded(db, slug, carol.email, adminEmail, 15, [
    { identifier: "DOCS", role: 20 },
    { identifier: "OPS", role: 15 },
    { identifier: "WEB", role: 15 },
  ]);
  expect(await invitations(), "the deleted invitation to carol, hers to Other, eve's").toEqual(untouched);
  expect(
    (await listMembers(api, admin, slug)).map((m) => [m.member.id, m.is_active]),
    "acme's members, carol's membership ended"
  ).toContainEqual([carol.id, false]);
  const webMembers = await api.GET("/api/v0/projects/{project_id}/members", {
    params: { path: { project_id: web.id } },
    headers: bearer(admin),
  });
  expect(webMembers.data?.data.map((m) => m.member_id).toSorted()).toEqual([adminId, bob.id].toSorted());

  // Invited back as a guest, she accepts: her membership's row is active again as a guest's, and her ended project
  // memberships stay ended, a guest's: she sees none of the projects.
  const [invitation] = await invite(api, admin, slug, [{ email: carol.email, role: 5 }]);
  if (!invitation) {
    throw new Error("the invitation to carol was not created");
  }
  expect(await accept(api, carol.token, invitation)).toMatchObject({ slug, role: 5 });
  expect(await membership(carol.id), "carol's membership, restored").toBe(carolsMembership);
  await expectMembership(db, slug, carol.email, { role: 5, is_active: true });
  expect(await projectRoles(carol.email), "carol's memberships of the projects").toEqual([
    { identifier: "DOCS", role: 5, is_active: false, by: carol.email },
    { identifier: "OPS", role: 5, is_active: false, by: carol.email },
    { identifier: "WEB", role: 5, is_active: false, by: carol.email },
  ]);
  const reads = await Promise.all(
    [web, ops, docs].map((project) =>
      api.GET("/api/v0/projects/{project_id}", {
        params: { path: { project_id: project.id } },
        headers: bearer(carol.token),
      })
    )
  );
  expect(
    reads.map((read) => [read.response.status, read.error?.code]),
    "carol reads Web, Ops and Docs"
  ).toEqual([
    [404, "project.not_found"],
    [404, "project.not_found"],
    [404, "project.not_found"],
  ]);
});
