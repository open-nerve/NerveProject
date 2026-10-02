import {
  accept,
  createProject,
  createWorkspace,
  invite,
  inviteAndAccept,
  membershipOf,
  slugFor,
} from "../../fixtures/api";
import { expectMembership, expectMembershipEnded, expectWrittenLastBy } from "../../fixtures/assert/workspace";
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";
import { nerveUsers } from "../../fixtures/users";
import { nerveWorkspaces, nerveWorkspacesFails } from "../../fixtures/workspaces";

// W12, the administrator restores a removed member (M3 design 2, 3.11), with the old invitation of Codex S2: an
// invitation never changes an active membership (3.8). There is no page version.

/** The arguments of nerve workspaces reactivate-member for the account of email in the workspace of slug. */
const reactivate = (slug: string, email: string) => ["reactivate-member", "--slug", slug, "--email", email];

test("W12: nerve workspaces reactivate-member restores a removed admin's membership with its role, his project membership ended until he joins; an old guest invitation then changes nothing; an unknown workspace or account, or a never-member, changes nothing", async ({
  api,
  db,
}, testInfo) => {
  const aEmail = emailFor(testInfo, "a");
  const a = (await createPAT(api, (await register(api, aEmail)).access_token)).token;
  const bEmail = emailFor(testInfo, "b");
  const b = (await createPAT(api, (await register(api, bEmail)).access_token)).token;
  const carolEmail = emailFor(testInfo, "carol");
  const carol = (await createPAT(api, (await register(api, carolEmail)).access_token)).token;
  const strangerEmail = emailFor(testInfo, "stranger");
  await register(api, strangerEmail);
  const slug = slugFor(testInfo);
  await createWorkspace(api, a, { name: "Acme", slug });
  await inviteAndAccept(api, a, slug, { email: bEmail, token: b }, 20);
  await inviteAndAccept(api, a, slug, { email: carolEmail, token: carol }, 15);
  // B, acme's second admin, is the admin of Ops and of Old; carol joins Ops, then leaves acme, which ends her
  // membership of Ops.
  const ops = await createProject(api, b, slug, { name: "Ops", identifier: "OPS" });
  const old = await createProject(api, b, slug, { name: "Old", identifier: "OLD" });
  const join = async (projectId: string, token: string) =>
    (
      await api.POST("/api/v0/projects/{project_id}/join", {
        params: { path: { project_id: projectId } },
        headers: bearer(token),
      })
    ).response.status;
  expect(await join(ops.id, carol), "carol joins Ops").toBe(200);
  const leave = async (target: string, token: string) => {
    const left = await api.POST("/api/v0/workspaces/{slug}/leave", {
      params: { path: { slug: target } },
      headers: bearer(token),
    });
    return [left.response.status, left.error?.code];
  };
  expect(await leave(slug, carol), "carol leaves acme").toEqual([204, undefined]);
  // B is also Other's member and Lab's admin there, carol Lab's member; Spare, A's third workspace, has a pending
  // invitation to him.
  const other = slugFor(testInfo, "other");
  const spare = slugFor(testInfo, "spare");
  await createWorkspace(api, a, { name: "Other", slug: other });
  await inviteAndAccept(api, a, other, { email: bEmail, token: b }, 15);
  const lab = await createProject(api, b, other, { name: "Lab", identifier: "LAB" });
  await inviteAndAccept(api, a, other, { email: carolEmail, token: carol }, 15);
  expect(await join(lab.id, carol), "carol joins Lab").toBe(200);
  await createWorkspace(api, a, { name: "Spare", slug: spare });
  const [spares] = await invite(api, a, spare, [{ email: bEmail, role: 15 }]);
  if (!spares) {
    throw new Error("the invitation to B in Spare was not created");
  }
  const bMembership = await membershipOf(api, a, slug, await accountId(api, b));
  // The membership of the account of email of the project of projectId: the row's id, its role, whether it is
  // active, and who wrote it last.
  const projectRow = async (projectId: string, email: string) =>
    (
      await db.query<{ id: string }>(
        `SELECT m.id, m.role, m.is_active, w.email AS by FROM project_members m
           JOIN users u ON u.id = m.member_id JOIN users w ON w.id = m.updated_by_id WHERE m.project_id = $1 AND u.email = $2`,
        [projectId, email]
      )
    )[0];
  // B's membership of acme and his of Ops, as projectRow has them.
  const bs = async () => {
    const [workspace] = await db.query(
      `SELECT m.id, m.role, m.is_active, w.email AS by FROM workspace_members m JOIN workspaces s ON s.id = m.workspace_id
         JOIN users u ON u.id = m.member_id JOIN users w ON w.id = m.updated_by_id WHERE s.slug = $1 AND u.email = $2`,
      [slug, bEmail]
    );
    return { workspace, project: await projectRow(ops.id, bEmail) };
  };
  const opsMembership = (await bs()).project?.id;
  const carolsOps = await projectRow(ops.id, carolEmail);
  // The tables a reactivation could write: it writes the first alone.
  const memberships = async () => ({
    workspace: await db.query("SELECT * FROM workspace_members ORDER BY id"),
    projects: await db.query("SELECT * FROM project_members ORDER BY id"),
    invitations: await db.query("SELECT * FROM workspace_member_invites ORDER BY id"),
  });

  // A removes B: his memberships of acme end, their rows kept, and nothing of his elsewhere changes. He wrote each of
  // them last, so that A's writing them shows.
  await expectWrittenLastBy(db, slug, bEmail, { workspace: bEmail, OLD: bEmail, OPS: bEmail });
  const removed = await api.DELETE("/api/v0/workspace-members/{workspace_member_id}", {
    params: { path: { workspace_member_id: bMembership } },
    headers: bearer(a),
  });
  expect(removed.response.status).toBe(204);
  await expectMembershipEnded(db, slug, bEmail, aEmail, 20, [
    { identifier: "OLD", role: 20 },
    { identifier: "OPS", role: 20 },
  ]);
  await expectMembership(db, other, bEmail, { role: 15, is_active: true });
  expect(await projectRow(lab.id, bEmail), "B's membership of Lab").toMatchObject({
    role: 20,
    is_active: true,
    by: bEmail,
  });
  expect(
    await db.query("SELECT responded_at, deleted_at FROM workspace_member_invites WHERE id = $1", [spares.id]),
    "the invitation to B in Spare"
  ).toEqual([{ responded_at: null, deleted_at: null }]);
  // A, now acme's only active admin, cannot leave it.
  expect(await leave(slug, a), "A leaves acme").toEqual([409, "workspace.sole_admin"]);
  // A joins Old and deletes it, B's membership with it. carol leaves Other, then B; his account is deactivated;
  // meanwhile A invites him back to acme as a guest: the old invitation.
  expect(await join(old.id, a), "A joins Old").toBe(200);
  const oldDeleted = await api.DELETE("/api/v0/projects/{project_id}", {
    params: { path: { project_id: old.id } },
    headers: bearer(a),
  });
  expect(oldDeleted.response.status, "A deletes Old").toBe(204);
  expect(await leave(other, carol), "carol leaves Other").toEqual([204, undefined]);
  expect(await leave(other, b), "B leaves Other").toEqual([204, undefined]);
  await nerveUsers(db, ["deactivate", "--email", bEmail]);
  const [oldInvitation] = await invite(api, a, slug, [{ email: bEmail, role: 5 }]);
  if (!oldInvitation) {
    throw new Error("the invitation to B was not created");
  }

  // An unknown workspace, an unknown account and an account that was never acme's member change nothing.
  const before = await memberships();
  await nerveWorkspacesFails(db, reactivate(slugFor(testInfo, "nowhere"), bEmail), "No workspace has this slug.");
  await nerveWorkspacesFails(db, reactivate(slug, emailFor(testInfo, "nobody")), "No account has this e-mail address.");
  await nerveWorkspacesFails(
    db,
    reactivate(slug, strangerEmail),
    "The account has never been a member of this workspace."
  );
  expect(await memberships(), "the three tables after the refusals").toEqual(before);

  // The command restores B's row of acme alone, with its role, though his account is deactivated, and says what is
  // next: his membership of Ops is the one of acme's projects still ended, Old's deleted.
  expect(await nerveWorkspaces(db, reactivate(slug, bEmail.toUpperCase()))).toBe(
    `reactivated ${bEmail} in ${slug} as admin; project memberships still ended: 1, each restored when the member joins or is added to its project; the account is deactivated: run nerve users activate --email ${bEmail} next\n`
  );
  const asReactivated = {
    workspace: { id: bMembership, role: 20, is_active: true, by: aEmail },
    project: { id: opsMembership, role: 20, is_active: false, by: aEmail },
  };
  expect(await bs(), "B's memberships, reactivated").toEqual(asReactivated);
  await expectMembership(db, slug, carolEmail, { role: 15, is_active: false });
  await expectMembership(db, other, bEmail, { role: 15, is_active: false });
  await nerveUsers(db, ["activate", "--email", bEmail]);
  const reactivated = await memberships();
  expect(await nerveWorkspaces(db, reactivate(slug, bEmail))).toBe(
    `${bEmail} is an active member of ${slug} already; nothing changed\n`
  );
  expect(await memberships(), "the three tables after the second run").toEqual(reactivated);

  // B, acme's admin, who may join any of its projects (3.5), joins Ops, his membership of which is still ended: his old
  // row, an admin's, the lower of its role and his workspace role; carol's stays ended.
  expect(await bs(), "B's memberships, before he joins Ops").toEqual(asReactivated);
  expect(await join(ops.id, b), "B joins Ops").toBe(200);
  const restored = {
    workspace: { id: bMembership, role: 20, is_active: true, by: aEmail },
    project: { id: opsMembership, role: 20, is_active: true, by: bEmail },
  };
  expect(await bs(), "B's memberships, Ops joined").toEqual(restored);
  expect(await projectRow(ops.id, carolEmail), "carol's membership of Ops").toEqual(carolsOps);

  // A leaves, and B, acme's only admin, accepts the old guest invitation: it is consumed, and changes nothing.
  expect(await leave(slug, a), "A leaves acme").toEqual([204, undefined]);
  expect(await accept(api, b, oldInvitation)).toMatchObject({ slug, role: 20, total_members: 1 });
  expect(await bs(), "B's memberships, the old invitation accepted").toEqual(restored);
  // The database compares the moments: a Date holds milliseconds, a timestamptz microseconds.
  expect(
    await db.query(
      `SELECT accepted, responded_at IS NOT NULL AS responded, deleted_at = responded_at AS deleted_when_answered
         FROM workspace_member_invites WHERE id = $1`,
      [oldInvitation.id]
    ),
    "the old invitation"
  ).toEqual([{ accepted: true, responded: true, deleted_when_answered: true }]);

  // carol declines B's invitation back; reactivated by the command, she keeps the declined invitation, and so she
  // does when B removes her again: an ending deletes pending invitations alone.
  const [declined] = await invite(api, b, slug, [{ email: carolEmail, role: 5 }]);
  if (!declined) {
    throw new Error("the invitation to carol was not created");
  }
  const declining = await api.POST("/api/v0/workspace-invitations/{invitation_id}/decline", {
    params: { path: { invitation_id: declined.id } },
    body: { token: declined.token },
    headers: bearer(carol),
  });
  expect(declining.response.status).toBe(204);
  const declinedRow = () =>
    db.query("SELECT to_jsonb(i) AS row FROM workspace_member_invites i WHERE i.id = $1", [declined.id]);
  const kept = await declinedRow();
  expect(await nerveWorkspaces(db, reactivate(slug, carolEmail))).toBe(
    `reactivated ${carolEmail} in ${slug} as member; project memberships still ended: 1, each restored when the member joins or is added to its project\n`
  );
  expect(await declinedRow(), "the declined invitation to carol, she reactivated").toEqual(kept);
  const removedAgain = await api.DELETE("/api/v0/workspace-members/{workspace_member_id}", {
    params: { path: { workspace_member_id: await membershipOf(api, b, slug, await accountId(api, carol)) } },
    headers: bearer(b),
  });
  expect(removedAgain.response.status).toBe(204);
  expect(await declinedRow(), "the declined invitation to carol").toEqual(kept);
});
