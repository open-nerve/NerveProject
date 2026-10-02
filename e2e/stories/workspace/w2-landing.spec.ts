import {
  addProjectMembers,
  createProject,
  createWorkspace,
  invite,
  inviteAndAccept,
  membershipOf,
  slugFor,
  type Workspace,
} from "../../fixtures/api";
import { expectMembershipEnded, expectWorkspaceDeleted, expectWrittenLastBy } from "../../fixtures/assert/workspace";
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// W2, where an account lands once signed in (M3 design 2). The landing rule is the web app's (P9); the API version
// checks the list it rests on.

/** A workspace as GET /api/v0/workspaces lists it for its member of role, with members active members. */
const entry = (w: Workspace, role: number, members: number) => ({
  slug: w.slug,
  role,
  total_members: members,
  created_at: w.created_at,
});

test("W2 (API): an account's workspaces are those it is an active member of, with its role, members and creation time; one fewer once it deletes one, none once it leaves the other after making a member its admin", async ({
  api,
  db,
}, testInfo) => {
  const aliceEmail = emailFor(testInfo, "alice");
  const alice = (await createPAT(api, (await register(api, aliceEmail)).access_token)).token;
  const bobEmail = emailFor(testInfo, "bob");
  const bob = (await createPAT(api, (await register(api, bobEmail)).access_token)).token;
  const bobId = await accountId(api, bob);
  const first = slugFor(testInfo, "first");
  const second = slugFor(testInfo, "second");
  // First holds a row of each table its deletion writes: alice's membership and bob's, which he wrote, so that the
  // deletion's writing it shows; a pending invitation (bob's, accepted, was deleted alone); her display settings; and
  // her project Web with its own rows.
  const firstWorkspace = await createWorkspace(api, alice, { name: "First", slug: first });
  await inviteAndAccept(api, alice, first, { email: bobEmail, token: bob }, 15);
  await invite(api, alice, first, [{ email: emailFor(testInfo, "invitee"), role: 15 }]);
  const settings = await api.PATCH("/api/v0/me/workspaces/{slug}/preferences", {
    params: { path: { slug: first } },
    body: { navigation_project_limit: 3 },
    headers: bearer(alice),
  });
  expect(settings.response.status).toBe(200);
  await createProject(api, alice, first, { name: "Web", identifier: "WEB" });
  // Second has bob as its member, and his project Ops, alice its admin by his adding, so both are its admins.
  const secondWorkspace = await createWorkspace(api, alice, { name: "Second", slug: second });
  await inviteAndAccept(api, alice, second, { email: bobEmail, token: bob }, 15);
  const ops = await createProject(api, bob, second, { name: "Ops", identifier: "OPS" });
  await addProjectMembers(api, bob, ops.id, [{ member_id: await accountId(api, alice), role: 20 }]);
  // Second changes once, so that the time listed can be told from its last change's.
  const changed = await api.PATCH("/api/v0/workspaces/{slug}", {
    params: { path: { slug: second } },
    body: { timezone: "Europe/Berlin" },
    headers: bearer(alice),
  });
  expect([changed.response.status, changed.data?.updated_at === secondWorkspace.created_at]).toEqual([200, false]);

  const listed = async (token: string) => {
    const { data, response } = await api.GET("/api/v0/workspaces", { headers: bearer(token) });
    expect(response.status).toBe(200);
    return data?.data.map((w) => ({
      slug: w.slug,
      role: w.role,
      total_members: w.total_members,
      created_at: w.created_at,
    }));
  };
  expect(await listed(alice)).toEqual([entry(firstWorkspace, 20, 2), entry(secondWorkspace, 20, 2)]);
  expect(await listed(bob), "bob's, a member's").toEqual([entry(firstWorkspace, 15, 2), entry(secondWorkspace, 15, 2)]);

  await expectWrittenLastBy(db, first, bobEmail, { workspace: bobEmail });
  const deleted = await api.DELETE("/api/v0/workspaces/{slug}", {
    params: { path: { slug: first } },
    headers: bearer(alice),
  });
  expect(deleted.response.status).toBe(204);
  await expectWorkspaceDeleted(db, first, aliceEmail, ["workspace_member_invites"]);
  expect(await listed(alice)).toEqual([entry(secondWorkspace, 20, 2)]);
  expect(await listed(bob)).toEqual([entry(secondWorkspace, 15, 2)]);

  // Second's only admin cannot leave it; once bob is its admin too, she leaves.
  const alone = await api.POST("/api/v0/workspaces/{slug}/leave", {
    params: { path: { slug: second } },
    headers: bearer(alice),
  });
  expect([alone.response.status, alone.error?.code]).toEqual([409, "workspace.sole_admin"]);
  const promoted = await api.PATCH("/api/v0/workspace-members/{workspace_member_id}", {
    params: { path: { workspace_member_id: await membershipOf(api, alice, second, bobId) } },
    body: { role: 20 },
    headers: bearer(alice),
  });
  expect(promoted.response.status).toBe(200);
  // bob, an admin now, sets her role as it is: he wrote her memberships last, so that her leaving's writing shows.
  const rewritten = await api.PATCH("/api/v0/workspace-members/{workspace_member_id}", {
    params: { path: { workspace_member_id: await membershipOf(api, bob, second, await accountId(api, alice)) } },
    body: { role: 20 },
    headers: bearer(bob),
  });
  expect(rewritten.response.status).toBe(200);
  await expectWrittenLastBy(db, second, aliceEmail, { workspace: bobEmail, OPS: bobEmail });
  const left = await api.POST("/api/v0/workspaces/{slug}/leave", {
    params: { path: { slug: second } },
    headers: bearer(alice),
  });
  expect(left.response.status).toBe(204);
  await expectMembershipEnded(db, second, aliceEmail, aliceEmail, 20, [{ identifier: "OPS", role: 20 }]);
  expect(await listed(alice)).toEqual([]);
  expect(await listed(bob)).toEqual([entry(secondWorkspace, 20, 1)]);
});
