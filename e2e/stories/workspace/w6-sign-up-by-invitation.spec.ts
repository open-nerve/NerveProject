import { accept, createApi, createWorkspace, invite, slugFor, type WorkspaceInvitation } from "../../fixtures/api";
import { countIdentity, expectNothingAdded, expectRegistered } from "../../fixtures/assert/identity";
import { expectInvitations, expectMembership, lastWorkspaceOf } from "../../fixtures/assert/workspace";
import { bearer, createPAT, emailFor, password, register, type RegisterInvitation } from "../../fixtures/auth";
import { fillSignUp, formAlert, submitSignIn, submitSignUp } from "../../fixtures/auth-pages";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import { registerOnboarded, sentTo } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";
import { invitationLinkOf } from "../../fixtures/workspace-pages";

// W6, sign up by an invitation while sign-up is off (M3 design 2, 3.8, 7.4; decision 1).

test("W6 (API): with sign-up off, the address an invitation was sent to registers with its link and then accepts it; without a link, with a wrong token, another address, a declined or a deleted invitation it may not", async ({
  api,
  db,
  nerveWith,
}, testInfo) => {
  // The links are of the nerve that made them: without a key file each nerve has its own key (README,
  // deployment), so the workspace, the invitations and the answers all go through the closed nerve, with
  // personal access tokens, which every nerve on the database accepts.
  const closed = createApi((await nerveWith({ NERVE_AUTH__SIGNUP_ENABLED: "false" })).baseURL);
  const adminEmail = emailFor(testInfo, "admin");
  const admin = (await createPAT(api, (await register(api, adminEmail)).access_token)).token;
  const daveEmail = emailFor(testInfo, "dave");
  const dave = (await createPAT(api, (await register(api, daveEmail)).access_token)).token;
  const slug = slugFor(testInfo);
  await createWorkspace(closed, admin, { name: "Acme", slug });
  const carolEmail = emailFor(testInfo, "carol");
  const erinEmail = emailFor(testInfo, "erin");
  const [toCarol, toDave, toErin] = await invite(closed, admin, slug, [
    { email: carolEmail, role: 15 },
    { email: daveEmail, role: 15 },
    { email: erinEmail, role: 5 },
  ]);
  if (!toCarol || !toDave || !toErin) {
    throw new Error("the invitations were not created");
  }
  const declined = await closed.POST("/api/v0/workspace-invitations/{invitation_id}/decline", {
    params: { path: { invitation_id: toDave.id } },
    body: { token: toDave.token },
    headers: bearer(dave),
  });
  expect(declined.response.status).toBe(204);
  const deleted = await closed.DELETE("/api/v0/workspace-invitations/{invitation_id}", {
    params: { path: { invitation_id: toErin.id } },
    headers: bearer(admin),
  });
  expect(deleted.response.status).toBe(204);

  const userAgent = "nerve-e2e/W6";
  const signUp = (email: string, invitation?: RegisterInvitation) =>
    closed.POST("/api/v0/auth/register", {
      body: { email, password, invitation },
      headers: { "User-Agent": userAgent },
    });
  const changed = toCarol.token.slice(0, 10) + (toCarol.token[10] === "A" ? "B" : "A") + toCarol.token.slice(11);
  const before = await countIdentity(db);
  // Each is refused as sign-up being off. dave's address is taken: an invitation that let it through would
  // answer 409.
  const refusals = {
    "no invitation": signUp(carolEmail),
    "a wrong token": signUp(carolEmail, { id: toCarol.id, token: changed }),
    "another address": signUp(emailFor(testInfo, "mallory"), { id: toCarol.id, token: toCarol.token }),
    "a declined invitation": signUp(daveEmail, { id: toDave.id, token: toDave.token }),
    "a deleted invitation": signUp(erinEmail, { id: toErin.id, token: toErin.token }),
    "no such invitation": signUp(carolEmail, { id: crypto.randomUUID(), token: toCarol.token }),
  };
  const answers = await Promise.all(
    Object.entries(refusals).map(async ([label, refused]) => {
      const r = await refused;
      return { label, status: r.response.status, body: r.error };
    })
  );
  expect(answers.map((a) => [a.label, a.status, a.body?.code])).toEqual(
    Object.keys(refusals).map((label) => [label, 403, "identity.signup_disabled"])
  );
  // The whole answer is the same, sign-up being off: none says which check failed.
  expect(answers.map((a) => a.body)).toEqual(answers.map(() => answers[0]?.body));
  await expectNothingAdded(db, before);

  // carol registers with her link, the address as she types it: A1's account and session, the address
  // normalized. The invitation stays to be answered.
  const registered = await signUp(` ${carolEmail.toUpperCase()} `, { id: toCarol.id, token: toCarol.token });
  expect(registered.response.status).toBe(201);
  await expectRegistered(db, {
    email: carolEmail,
    refreshToken: registered.data?.refresh_token ?? "",
    userAgent,
    ip: "127.0.0.1",
  });
  const tokens = [toCarol.token, toDave.token, toErin.token];
  const dismissed = [
    { email: daveEmail, role: 15, accepted: false, responded: true, deleted: false },
    { email: erinEmail, role: 5, accepted: false, responded: false, deleted: true },
  ];
  await expectInvitations(
    db,
    slug,
    adminEmail,
    [{ email: carolEmail, role: 15, accepted: false, responded: false, deleted: false }, ...dismissed],
    tokens
  );
  await expectMembership(db, slug, carolEmail, null);

  // She accepts it: a member, and the invitation accepted and deleted at the moment of the answer.
  expect(await accept(closed, registered.data?.access_token ?? "", toCarol)).toMatchObject({
    slug,
    role: 15,
    total_members: 2,
  });
  await expectMembership(db, slug, carolEmail, { role: 15, is_active: true });
  await expectInvitations(
    db,
    slug,
    adminEmail,
    [{ email: carolEmail, role: 15, accepted: true, responded: true, deleted: true }, ...dismissed],
    tokens
  );
});

/** The sign-in or sign-up page for an invitation's link, which comes back to it. */
const authPathOf = (path: "/" | "/sign-up", invitation: Pick<WorkspaceInvitation, "id" | "token">) => {
  const query = { invitation_id: invitation.id, token: invitation.token, next_path: invitationLinkOf(invitation) };
  return `${path}?${new URLSearchParams(query).toString()}`;
};

test("W6 (page): with sign-up off, the sign-up page says so; the invitee opens her link signed out and signs up to accept it, under the workspace's name, with the link's invitation; back at the link she accepts, takes the profile step alone and lands in the workspace", async ({
  api,
  browser,
  db,
  nerveWith,
}, testInfo) => {
  // The workspace and the invitation are the closed nerve's, as in W6's API version.
  const closed = await nerveWith({ NERVE_AUTH__SIGNUP_ENABLED: "false" });
  const closedApi = createApi(closed.baseURL);
  const adminEmail = emailFor(testInfo, "admin");
  const admin = (await createPAT(api, (await register(api, adminEmail)).access_token)).token;
  const slug = slugFor(testInfo);
  const acme = await createWorkspace(closedApi, admin, { name: "Acme", slug });
  const carolEmail = emailFor(testInfo, "carol");
  const [toCarol] = await invite(closedApi, admin, slug, [{ email: carolEmail, role: 15 }]);
  if (!toCarol) {
    throw new Error("the invitation was not created");
  }
  const context = await browser.newContext({ baseURL: closed.baseURL });
  const page = await context.newPage();
  const watch = await watchPage(page);

  // Without a link, M2's sign-up page: nerve says sign-up is closed.
  await page.goto("/sign-up");
  expect(await submitSignUp(page, carolEmail, password)).toBe(403);
  await expect(formAlert(page)).toHaveText("Sign-up is closed.");

  // With her link: she signs up to accept it, and the registration carries the invitation.
  await page.goto(invitationLinkOf(toCarol));
  await page.getByRole("link", { name: "Sign up to accept" }).click();
  await expect(page).toHaveURL(authPathOf("/sign-up", toCarol));
  // the heading: "Join", the workspace's logo (its initial while it has no image), its name
  await expect(page.getByText(/^Join\s+A\s+Acme$/)).toBeVisible();
  await fillSignUp(page, carolEmail, password);
  const registered = await sentTo(page, "POST", "/api/v0/auth/register", () =>
    page.getByRole("button", { name: "Create account", exact: true }).click()
  );
  expect([registered.answer.status(), registered.body]).toEqual([
    201,
    { email: carolEmail, password, invitation: { id: toCarol.id, token: toCarol.token } },
  ]);

  // Back at the link she accepts; her onboarding is the profile step alone, and she lands in Acme.
  await expect(page).toHaveURL(invitationLinkOf(toCarol));
  await page.getByRole("button", { name: "Accept" }).click();
  await expect(page).toHaveURL("/onboarding");
  await page.getByLabel("Name", { exact: true }).fill("Carol");
  await page.getByRole("button", { name: "Continue", exact: true }).click();
  await expect(page).toHaveURL(`/${slug}`);
  await expectMembership(db, slug, carolEmail, { role: 15, is_active: true });
  expect(await lastWorkspaceOf(db, carolEmail)).toBe(acme.id);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([
    ["403 POST /api/v0/auth/register"],
    [],
    [],
  ]);
  // Acme's home logs the hint once; the browser reports the refusal.
  await expectQuietConsole(page, watch, {
    warnings: [EMOJI_CHECK_WARNING],
    errors: ["Failed to load resource: the server responded with a status of 403 (Forbidden)"],
  });
  await context.close();
});

test("W6 (page): with sign-up on, one who has an account goes from the sign-up page of her link to the sign-in page, which keeps the link and comes back to it", async ({
  api,
  baseURL,
  browser,
}, testInfo) => {
  const admin = await registerOnboarded(api, emailFor(testInfo, "admin"));
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.access_token, { name: "Acme", slug });
  const daveEmail = emailFor(testInfo, "dave");
  await registerOnboarded(api, daveEmail);
  const [toDave] = await invite(api, admin.access_token, slug, [{ email: daveEmail, role: 15 }]);
  if (!toDave) {
    throw new Error("the invitation was not created");
  }
  const context = await browser.newContext({ baseURL });
  const page = await context.newPage();
  await page.goto(authPathOf("/sign-up", toDave));
  await page.getByRole("link", { name: "Sign in", exact: true }).click();
  await expect(page).toHaveURL(authPathOf("/", toDave));
  await expect(page.getByText(/^Join\s+A\s+Acme$/)).toBeVisible();
  expect(await submitSignIn(page, daveEmail, password)).toBe(200);
  await expect(page).toHaveURL(invitationLinkOf(toDave));
  await context.close();
});
