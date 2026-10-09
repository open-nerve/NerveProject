import {
  addProjectMembers,
  createProject,
  createWorkspace,
  furnishWorkspace,
  inviteAndAccept,
  membershipOf,
  slugFor,
  type Workspace,
} from "../../fixtures/api";
import {
  expectMembershipEnded,
  expectWorkspaceDeleted,
  expectWrittenLastBy,
  lastWorkspaceOf,
} from "../../fixtures/assert/workspace";
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole } from "../../fixtures/browser";
import { registerOnboarded } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";
import {
  deleteFromGeneralPage,
  endMembership,
  pickRole,
  signInAnew,
  switchWorkspace,
} from "../../fixtures/workspace-pages";

// W2, where an account lands once signed in (M3 design 2). The landing rule is the web app's: the page version
// follows it; the API version checks the list it rests on.

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
  // deletion's writing it shows; the invitee's pending invitation (bob's own was deleted when he accepted it, before
  // the workspace); her display settings; and her project Web with its own rows, its label Bug among them.
  const firstWorkspace = await createWorkspace(api, alice, { name: "First", slug: first });
  await furnishWorkspace(api, alice, first, { email: bobEmail, token: bob }, emailFor(testInfo, "invitee"));
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

test("W2 (page): signed in, an account lands on the workspace it opened last; once that is deleted, on its other one; once it has made a member that one's admin and left it, on the page that creates one; no request fails", async ({
  api,
  baseURL,
  browser,
  db,
}, testInfo) => {
  const aliceEmail = emailFor(testInfo, "alice");
  const alice = await registerOnboarded(api, aliceEmail);
  const bobEmail = emailFor(testInfo, "bob");
  const bob = await registerOnboarded(api, bobEmail);
  const first = slugFor(testInfo, "first");
  const second = slugFor(testInfo, "second");
  await createWorkspace(api, alice.access_token, { name: "First", slug: first });
  await inviteAndAccept(api, alice.access_token, first, { email: bobEmail, token: bob.access_token }, 15);
  const secondWorkspace = await createWorkspace(api, alice.access_token, { name: "Second", slug: second });
  const bobsMembership = await membershipOf(api, alice.access_token, first, await accountId(api, bob.access_token));
  /** Signs alice in, in a browser of her own, and checks where she lands. */
  const signIn = async (landing: string) => {
    const signedIn = await signInAnew(browser, baseURL ?? "", aliceEmail);
    await expect(signedIn.page).toHaveURL(landing);
    return signedIn;
  };
  /**
   * Nothing failed in the browser of signedIn, where loads documents opened a workspace's page (each logs
   * EMOJI_CHECK_WARNING); then it closes.
   */
  const nothingFailed = async ({ page, watch, close }: Awaited<ReturnType<typeof signIn>>, loads: number) => {
    expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
    await expectQuietConsole(page, watch, { warnings: Array.from({ length: loads }, () => EMOJI_CHECK_WARNING) });
    await close();
  };

  // She has opened neither: she lands on First, created first. She opens Second through the workspace menu, and the
  // web app writes it down as the one she opened last.
  const one = await signIn(`/${first}`);
  const switched = await switchWorkspace(one.page, secondWorkspace.id);
  expect([switched.answer.status(), switched.body]).toEqual([200, { last_workspace_id: secondWorkspace.id }]);
  await expect(one.page).toHaveURL(`/${second}`);
  expect(await lastWorkspaceOf(db, aliceEmail)).toBe(secondWorkspace.id);
  await nothingFailed(one, 1);

  // Signed in again, she lands on Second, and deletes it from its general page: the root lands her on First.
  const two = await signIn(`/${second}`);
  await two.page.goto(`/${second}/settings`);
  expect((await deleteFromGeneralPage(two.page, second, "Second")).status()).toBe(204);
  await expect(two.page).toHaveURL(`/${first}`);
  await nothingFailed(two, 2);

  // Signed in again, she lands on First. She makes bob its admin, and leaves it: the root lands her on the page that
  // creates a workspace, as it does when she signs in again.
  const three = await signIn(`/${first}`);
  await three.page.goto(`/${first}/settings/members`);
  const promoted = await pickRole(three.page, bobEmail, bobsMembership, { from: "Member", to: "Admin" });
  expect([promoted.answer.status(), promoted.body]).toEqual([200, { role: 20 }]);
  const leaving = { method: "POST", path: `/api/v0/workspaces/${first}/leave` };
  expect((await endMembership(three.page, aliceEmail, "Leave", leaving)).status()).toBe(204);
  await expect(three.page).toHaveURL("/create-workspace");
  await nothingFailed(three, 2);
  await nothingFailed(await signIn("/create-workspace"), 0);
});
