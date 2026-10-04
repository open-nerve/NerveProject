import {
  addProjectMembers,
  createProject,
  createWorkspace,
  invite,
  inviteAndAccept,
  membershipOf,
  slugFor,
} from "../../fixtures/api";
import { accountOf, expectDeactivated, tokensOf } from "../../fixtures/assert/identity";
import { expectMembership } from "../../fixtures/assert/workspace";
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";
import { nerveUsers, nerveUsersFails } from "../../fixtures/users";
import { nerveWorkspaces } from "../../fixtures/workspaces";

// W9, a deactivation ends every membership of the account (M3 design 2, 3.9; M2 handoff 6): the API version, with the
// command. The page's is P9's.

/** The two refusals' details: the API's problem and the command's line say the same. */
const workspaceRefusal =
  "The workspace would be left without an admin: its only active admin cannot leave it, nor can his membership end while it has other active members. It must first be given another admin, or be deleted.";
const projectRefusal =
  "The project would be left without an admin: its only active admin cannot leave it, nor can his membership end while it has other active members. It must first be given another admin, or be deleted.";

test("W9: a deactivation is refused while the account is the only admin of a workspace or a project with other members, by the API and by the command, and changes nothing; once each has another admin, it ends every membership and deletes every invitation to the address and the pending ones of the workspace it leaves empty; nerve users activate, then reactivate-member, bring the account back a workspace at a time", async ({
  api,
  db,
}, testInfo) => {
  const aEmail = emailFor(testInfo, "a");
  const a = (await createPAT(api, (await register(api, aEmail)).access_token)).token;
  const bEmail = emailFor(testInfo, "b");
  const b = (await createPAT(api, (await register(api, bEmail)).access_token)).token;
  const cEmail = emailFor(testInfo, "c");
  const c = (await createPAT(api, (await register(api, cEmail)).access_token)).token;
  const acme = slugFor(testInfo, "acme");
  const beta = slugFor(testInfo, "beta");
  const gamma = slugFor(testInfo, "gamma");
  const delta = slugFor(testInfo, "delta");
  const epsilon = slugFor(testInfo, "epsilon");
  // B makes acme, A its member; A makes beta, B and C its members; B makes Lab there and adds C as its member. C
  // invites B to gamma, which he declines, and to delta. B makes epsilon, where he is alone, and invites C to it.
  await createWorkspace(api, b, { name: "Acme", slug: acme });
  await inviteAndAccept(api, b, acme, { email: aEmail, token: a }, 15);
  await createWorkspace(api, a, { name: "Beta", slug: beta });
  await inviteAndAccept(api, a, beta, { email: bEmail, token: b }, 15);
  await inviteAndAccept(api, a, beta, { email: cEmail, token: c }, 15);
  const lab = await createProject(api, b, beta, { name: "Lab", identifier: "LAB" });
  await addProjectMembers(api, b, lab.id, [{ member_id: await accountId(api, c), role: 15 }]);
  await createWorkspace(api, c, { name: "Gamma", slug: gamma });
  await createWorkspace(api, c, { name: "Delta", slug: delta });
  const [toGamma] = await invite(api, c, gamma, [{ email: bEmail, role: 15 }]);
  await invite(api, c, delta, [{ email: bEmail, role: 15 }]);
  await createWorkspace(api, b, { name: "Epsilon", slug: epsilon });
  const [toEpsilon] = await invite(api, b, epsilon, [{ email: cEmail, role: 15 }]);
  if (!toGamma || !toEpsilon) {
    throw new Error("the invitations to B in gamma and to C in epsilon were not both created");
  }
  const declined = await api.POST("/api/v0/workspace-invitations/{invitation_id}/decline", {
    params: { path: { invitation_id: toGamma.id } },
    body: { token: toGamma.token },
    headers: bearer(b),
  });
  expect(declined.response.status, "B declines gamma's invitation").toBe(204);
  // The tables a deactivation writes, whole.
  const tables = async () => ({
    users: await db.query("SELECT * FROM users ORDER BY id"),
    profiles: await db.query("SELECT * FROM profiles ORDER BY id"),
    sessions: await db.query("SELECT * FROM auth_sessions ORDER BY id"),
    workspaces: await db.query("SELECT * FROM workspace_members ORDER BY id"),
    projects: await db.query("SELECT * FROM project_members ORDER BY id"),
    invitations: await db.query("SELECT * FROM workspace_member_invites ORDER BY id"),
  });
  const deactivate = async () => {
    const answer = await api.POST("/api/v0/me/deactivate", { headers: bearer(b) });
    return [answer.response.status, answer.error?.code, answer.error?.detail];
  };
  const refusedBoth = async (code: string, detail: string) => {
    const before = await tables();
    expect(await deactivate(), "deactivateMe").toEqual([409, code, detail]);
    expect(await tables(), `the tables after deactivateMe's ${code}`).toEqual(before);
    await nerveUsersFails(db, ["deactivate", "--email", bEmail], detail);
    expect(await tables(), `the tables after the command's ${code}`).toEqual(before);
  };

  // B is acme's only admin, A its member: neither the API nor the command deactivates him.
  await refusedBoth("workspace.sole_admin", workspaceRefusal);

  // B makes A acme's admin; he is Lab's only admin still, C its member, in beta, where he is a member.
  const aInAcme = await api.PATCH("/api/v0/workspace-members/{workspace_member_id}", {
    params: { path: { workspace_member_id: await membershipOf(api, b, acme, await accountId(api, a)) } },
    body: { role: 20 },
    headers: bearer(b),
  });
  expect(aInAcme.response.status, "B makes A acme's admin").toBe(200);
  await refusedBoth("project.sole_admin", projectRefusal);

  // A, beta's admin, joins Lab, as its admin (M3 design 3.5). The API deactivates B: as A12, and every membership of
  // his ends, every invitation to his address is deleted, the declined one too, and his invitation of C to epsilon,
  // where he was alone (M3 design 3.7), at one moment, by him; every row of anyone else is as it was.
  const joined = await api.POST("/api/v0/projects/{project_id}/join", {
    params: { path: { project_id: lab.id } },
    headers: bearer(a),
  });
  expect(joined.response.status, "A joins Lab").toBe(200);
  const account = await accountOf(db, bEmail);
  const tokensBefore = await tokensOf(db, account.id);
  const others = async () => ({
    users: await db.query("SELECT * FROM users WHERE id <> $1 ORDER BY id", [account.id]),
    profiles: await db.query("SELECT * FROM profiles WHERE user_id <> $1 ORDER BY id", [account.id]),
    sessions: await db.query("SELECT * FROM auth_sessions WHERE user_id <> $1 ORDER BY id", [account.id]),
    workspaces: await db.query("SELECT * FROM workspace_members WHERE member_id <> $1 ORDER BY id", [account.id]),
    projects: await db.query("SELECT * FROM project_members WHERE member_id <> $1 ORDER BY id", [account.id]),
    invitations: await db.query("SELECT * FROM workspace_member_invites WHERE email <> $1 AND id <> $2 ORDER BY id", [
      bEmail,
      toEpsilon.id,
    ]),
  });
  const othersBefore = await others();
  expect(await deactivate(), "deactivateMe").toEqual([204, undefined, undefined]);
  await expectDeactivated(db, account, tokensBefore);
  // Each of his rows: where, whether it stands, who wrote it last; and how many moments they were written at.
  const ended = async () =>
    db.query(
      `WITH r AS (
         SELECT w.slug AS place, m.is_active AS active, b.email AS by, m.updated_at AS written
           FROM workspace_members m JOIN workspaces w ON w.id = m.workspace_id JOIN users b ON b.id = m.updated_by_id
          WHERE m.member_id = $1
         UNION ALL SELECT p.identifier, m.is_active, b.email, m.updated_at
           FROM project_members m JOIN projects p ON p.id = m.project_id JOIN users b ON b.id = m.updated_by_id WHERE m.member_id = $1
         UNION ALL SELECT 'invitation to ' || w.slug, i.deleted_at IS NULL, b.email, i.deleted_at
           FROM workspace_member_invites i JOIN workspaces w ON w.id = i.workspace_id JOIN users b ON b.id = i.updated_by_id
          WHERE i.email = $2 AND NOT i.accepted)
       SELECT place, active, by, (SELECT count(DISTINCT written) FROM r) AS moments FROM r ORDER BY place COLLATE "C"`,
      [account.id, bEmail]
    );
  const row = (place: string, active: boolean) => ({ place, active, by: bEmail, moments: "1" });
  expect(await ended(), "B's memberships and the invitations to him").toEqual(
    [
      row("LAB", false),
      row(acme, false),
      row(beta, false),
      row(epsilon, false),
      row(`invitation to ${delta}`, false),
      row(`invitation to ${gamma}`, false),
    ].toSorted((x, y) => (x.place < y.place ? -1 : 1))
  );
  expect(await others(), "every row of anyone else").toEqual(othersBefore);
  expect(
    await db.query(
      `SELECT i.deleted_at = m.updated_at AS with_his_membership, b.email AS by
         FROM workspace_member_invites i JOIN users b ON b.id = i.updated_by_id,
              workspace_members m JOIN workspaces w ON w.id = m.workspace_id
        WHERE i.id = $1 AND w.slug = $2 AND m.member_id = $3`,
      [toEpsilon.id, epsilon, account.id]
    ),
    "his invitation of C to epsilon"
  ).toEqual([{ with_his_membership: true, by: bEmail }]);
  const toEmpty = await api.POST("/api/v0/workspace-invitations/{invitation_id}/accept", {
    params: { path: { invitation_id: toEpsilon.id } },
    body: { token: toEpsilon.token },
    headers: bearer(c),
  });
  expect([toEmpty.response.status, toEmpty.error?.code], "C accepts it").toEqual([
    404,
    "workspace.invitation_not_found",
  ]);

  // The way back: nerve users activate brings the account back alone; reactivate-member, his membership of one
  // workspace, as it was, his project memberships still ended.
  expect(await nerveUsers(db, ["activate", "--email", bEmail])).toBe(
    `activated ${bEmail}: 1 API tokens are usable again\n`
  );
  await expectMembership(db, acme, bEmail, { role: 20, is_active: false });
  expect(await nerveWorkspaces(db, ["reactivate-member", "--slug", acme, "--email", bEmail])).toBe(
    `reactivated ${bEmail} in ${acme} as admin; project memberships still ended: 0, each restored when the member joins or is added to its project\n`
  );
  await expectMembership(db, acme, bEmail, { role: 20, is_active: true });
  await expectMembership(db, beta, bEmail, { role: 15, is_active: false });
  const sees = async (slug: string) =>
    (await api.GET("/api/v0/workspaces/{slug}", { params: { path: { slug } }, headers: bearer(b) })).response.status;
  expect([await sees(acme), await sees(beta)], "B reads acme, not beta").toEqual([200, 404]);

  // The command deactivates him again: his membership of acme, the one active, ends, as the API's did.
  const again = await accountOf(db, bEmail);
  expect(await nerveUsers(db, ["deactivate", "--email", bEmail])).toBe(
    `deactivated ${bEmail}: revoked 0 sessions and ended its memberships; to bring it back, run nerve users activate, then nerve workspaces reactivate-member in each workspace\n`
  );
  await expectDeactivated(db, again, tokensBefore);
  await expectMembership(db, acme, bEmail, { role: 20, is_active: false });
  expect(await others(), "every row of anyone else").toEqual(othersBefore);
});
