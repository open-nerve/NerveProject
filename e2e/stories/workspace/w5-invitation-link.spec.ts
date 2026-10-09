import { createWorkspace, invitationTo, invite, slugFor, type Api, type WorkspaceInvitation } from "../../fixtures/api";
import { expectInvitations, expectMembership, lastWorkspaceOf } from "../../fixtures/assert/workspace";
import { bearer, createPAT, emailFor, password, register } from "../../fixtures/auth";
import { submitSignIn } from "../../fixtures/auth-pages";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import { answerTo, registerOnboarded, sentTo } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";
import { invitationLinkOf } from "../../fixtures/workspace-pages";

// W5, accept or decline an invitation by its link (M3 design 2, 3.8; decision 1).

/** Accepts or declines invitation with token, as the account of bearerToken. */
async function answer(
  api: Api,
  bearerToken: string,
  which: "accept" | "decline",
  invitation: WorkspaceInvitation,
  token = invitation.token
) {
  return api.POST(`/api/v0/workspace-invitations/{invitation_id}/${which}`, {
    params: { path: { invitation_id: invitation.id } },
    body: { token },
    headers: bearer(bearerToken),
  });
}

test("W5 (API): the link shows the workspace and the role without the address; the invitee accepts or declines; another address, a wrong token and an answered invitation change nothing", async ({
  api,
  db,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = (await createPAT(api, (await register(api, adminEmail)).access_token)).token;
  // The admin has another workspace, with an invitation of its own: each link and each acceptance must name its
  // own workspace, whichever row the database reads first.
  const other = slugFor(testInfo, "other");
  await createWorkspace(api, admin, { name: "Other", slug: other });
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin, { name: "Acme", slug });
  const carolEmail = emailFor(testInfo, "carol");
  const carol = (await createPAT(api, (await register(api, carolEmail)).access_token)).token;
  const daveEmail = emailFor(testInfo, "dave");
  const dave = (await createPAT(api, (await register(api, daveEmail)).access_token)).token;
  const erinEmail = emailFor(testInfo, "erin");
  const erin = (await register(api, erinEmail)).access_token;
  const [toCarol, toDave] = await invite(api, admin, slug, [
    { email: carolEmail, role: 5 },
    { email: daveEmail, role: 15 },
  ]);
  const [toErin] = await invite(api, admin, other, [{ email: erinEmail, role: 15 }]);
  if (!toCarol || !toDave || !toErin) {
    throw new Error("the invitations were not created");
  }

  // The public view: with the token, the workspace and the role, never the address; the same with a bearer
  // token; without the token 400; with a changed token or of no invitation, the same 404.
  const view = (id: string, token?: string, headers: Record<string, string> = {}) =>
    api.GET("/api/v0/workspace-invitations/{invitation_id}", {
      params: { path: { invitation_id: id }, query: { token } as { token: string } },
      headers,
    });
  const shown = await view(toCarol.id, toCarol.token);
  expect(shown.response.status).toBe(200);
  expect(shown.data).toEqual({
    id: toCarol.id,
    role: 5,
    declined: false,
    workspace_name: "Acme",
    workspace_slug: slug,
    workspace_logo_url: null,
  });
  expect(JSON.stringify(shown.data)).not.toContain(carolEmail);
  expect((await view(toErin.id, toErin.token)).data).toEqual({
    id: toErin.id,
    role: 15,
    declined: false,
    workspace_name: "Other",
    workspace_slug: other,
    workspace_logo_url: null,
  });
  expect((await view(toCarol.id, toCarol.token, bearer(dave))).data).toEqual(shown.data);
  expect((await view(toCarol.id)).response.status).toBe(400);
  const changed = toCarol.token.slice(0, 10) + (toCarol.token[10] === "A" ? "B" : "A") + toCarol.token.slice(11);
  const wrong = await view(toCarol.id, changed);
  const missing = await view(crypto.randomUUID(), toCarol.token);
  expect([wrong.response.status, wrong.error]).toEqual([404, missing.error]);
  expect(wrong.error?.code).toBe("workspace.invitation_not_found");

  // Another account's answer: 403 without the address, and nothing changes.
  const refused = await Promise.all([answer(api, dave, "accept", toCarol), answer(api, dave, "decline", toCarol)]);
  expect(refused.map((r) => [r.response.status, r.error?.code])).toEqual([
    [403, "workspace.invitation_email_mismatch"],
    [403, "workspace.invitation_email_mismatch"],
  ]);
  expect(JSON.stringify(refused.map((r) => r.error))).not.toContain(carolEmail);
  // The invited address without the link's token: 400, nothing read. A wrong token: the 404 of an invitation
  // that does not exist.
  const noToken = await api.POST("/api/v0/workspace-invitations/{invitation_id}/accept", {
    params: { path: { invitation_id: toCarol.id } },
    body: {} as never,
    headers: bearer(carol),
  });
  expect(noToken.response.status).toBe(400);
  const badToken = await answer(api, carol, "accept", toCarol, changed);
  expect([badToken.response.status, badToken.error]).toEqual([404, wrong.error]);
  await expectMembership(db, slug, carolEmail, null);

  // carol accepts: a member with the invitation's role; the invitation is used up.
  const accepted = await answer(api, carol, "accept", toCarol);
  expect(accepted.response.status).toBe(200);
  expect(accepted.data).toMatchObject({ slug, role: 5, total_members: 2 });
  await expectMembership(db, slug, carolEmail, { role: 5, is_active: true });
  // erin accepts hers: the answer is Other.
  const erinAccepted = await answer(api, erin, "accept", toErin);
  expect([erinAccepted.response.status, erinAccepted.data]).toEqual([
    200,
    expect.objectContaining({ slug: other, role: 15, total_members: 2 }),
  ]);
  await expectMembership(db, other, erinEmail, { role: 15, is_active: true });
  // dave declines: no membership; the link shows it declined.
  expect((await answer(api, dave, "decline", toDave)).response.status).toBe(204);
  await expectMembership(db, slug, daveEmail, null);
  expect((await view(toDave.id, toDave.token)).data?.declined).toBe(true);
  await expectInvitations(
    db,
    slug,
    adminEmail,
    [
      { email: carolEmail, role: 5, accepted: true, responded: true, deleted: true },
      { email: daveEmail, role: 15, accepted: false, responded: true, deleted: false },
    ],
    [toCarol.token, toDave.token, toErin.token]
  );

  // Answered already: carol's is gone, 404; dave's is declined, 409 for either answer.
  expect((await answer(api, carol, "accept", toCarol)).error?.code).toBe("workspace.invitation_not_found");
  expect((await view(toCarol.id, toCarol.token)).response.status).toBe(404);
  const again = await Promise.all([answer(api, dave, "accept", toDave), answer(api, dave, "decline", toDave)]);
  expect(again.map((r) => [r.response.status, r.error?.code])).toEqual([
    [409, "workspace.invitation_responded"],
    [409, "workspace.invitation_responded"],
  ]);
  await expectMembership(db, slug, daveEmail, null);
});

test("W5 (page): signed out, the link shows the workspace and the role and sends the invitee to sign in and back; she accepts and lands in the workspace, written as the one she opened last; another invitation she ignores says she declined it; one deleted while its link is open is refused with the reason, and is no longer valid", async ({
  api,
  baseURL,
  browser,
  db,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = await registerOnboarded(api, adminEmail);
  const slug = slugFor(testInfo);
  const other = slugFor(testInfo, "other");
  const acme = await createWorkspace(api, admin.access_token, { name: "Acme", slug });
  await createWorkspace(api, admin.access_token, { name: "Other", slug: other });
  const carolEmail = emailFor(testInfo, "carol");
  await registerOnboarded(api, carolEmail);
  await invite(api, admin.access_token, slug, [{ email: carolEmail, role: 5 }]);
  await invite(api, admin.access_token, other, [{ email: carolEmail, role: 15 }]);
  const toAcme = await invitationTo(api, admin.access_token, slug, carolEmail);
  const toOther = await invitationTo(api, admin.access_token, other, carolEmail);
  const third = slugFor(testInfo, "third");
  await createWorkspace(api, admin.access_token, { name: "Third", slug: third });
  await invite(api, admin.access_token, third, [{ email: carolEmail, role: 15 }]);
  const toThird = await invitationTo(api, admin.access_token, third, carolEmail);

  const context = await browser.newContext({ baseURL });
  const page = await context.newPage();
  const watch = await watchPage(page);
  await page.goto(invitationLinkOf(toAcme));
  await expect(page.getByText("You have been invited to Acme as Guest.")).toBeVisible();
  await expect(page.getByText(carolEmail)).toHaveCount(0);
  await page.getByRole("button", { name: "Sign in to accept" }).click();
  expect(await submitSignIn(page, carolEmail, password)).toBe(200);
  await expect(page).toHaveURL(invitationLinkOf(toAcme));

  // She accepts: the page sends the link's token, and opens Acme, which it writes as the one she opened last.
  const accepted = await sentTo(page, "POST", `/api/v0/workspace-invitations/${toAcme.id}/accept`, () =>
    page.getByRole("button", { name: "Accept" }).click()
  );
  expect([accepted.answer.status(), accepted.body]).toEqual([200, { token: toAcme.token }]);
  await expect(page).toHaveURL(`/${slug}`);
  await expectMembership(db, slug, carolEmail, { role: 5, is_active: true });
  await expect.poll(() => lastWorkspaceOf(db, carolEmail)).toBe(acme.id);

  // She ignores Other's: the page reads the invitation again, which says she declined it.
  await page.goto(invitationLinkOf(toOther));
  const declined = await sentTo(page, "POST", `/api/v0/workspace-invitations/${toOther.id}/decline`, () =>
    page.getByRole("button", { name: "Ignore" }).click()
  );
  expect([declined.answer.status(), declined.body]).toEqual([204, { token: toOther.token }]);
  await expect(page.getByText("You declined this invitation.")).toBeVisible();
  await expectMembership(db, other, carolEmail, null);
  await expectInvitations(db, other, adminEmail, [
    { email: carolEmail, role: 15, accepted: false, responded: true, deleted: false },
  ]);

  // Third's, deleted while its link is open: nerve refuses the acceptance and says why; the page reads the link again.
  await page.goto(invitationLinkOf(toThird));
  await expect(page.getByRole("button", { name: "Accept" })).toBeVisible();
  const deleted = await api.DELETE("/api/v0/workspace-invitations/{invitation_id}", {
    params: { path: { invitation_id: toThird.id } },
    headers: bearer(admin.access_token),
  });
  expect(deleted.response.status).toBe(204);
  const accept = `/api/v0/workspace-invitations/${toThird.id}/accept`;
  const gone = await answerTo(page, "POST", accept, () => page.getByRole("button", { name: "Accept" }).click());
  expect(gone.status()).toBe(404);
  await expect(page.getByText("The invitation does not exist, or its link is not valid.")).toBeVisible();
  await expect(page.getByText("This invitation link is not valid.")).toBeVisible();
  await expectMembership(db, third, carolEmail, null);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([
    [`404 POST ${accept}`, `404 GET /api/v0/workspace-invitations/${toThird.id}`],
    [],
    [],
  ]);
  // Acme's home logs the hint once; the browser reports the two refusals.
  await expectQuietConsole(page, watch, {
    warnings: [EMOJI_CHECK_WARNING],
    errors: [
      "Failed to load resource: the server responded with a status of 404 (Not Found)",
      "Failed to load resource: the server responded with a status of 404 (Not Found)",
    ],
  });
  await context.close();
});

test("W5 (page): another address's invitation, once nerve refuses the answer, says it was sent to another address, offers no answer but signing out, and changes nothing; a link with a changed token, or none, is not valid", async ({
  api,
  db,
  signedInPage,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = await registerOnboarded(api, adminEmail);
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.access_token, { name: "Acme", slug });
  const carolEmail = emailFor(testInfo, "carol");
  await invite(api, admin.access_token, slug, [{ email: carolEmail, role: 15 }]);
  const toCarol = await invitationTo(api, admin.access_token, slug, carolEmail);
  const daveEmail = emailFor(testInfo, "dave");
  const page = await signedInPage(await registerOnboarded(api, daveEmail));
  const watch = await watchPage(page);

  await page.goto(invitationLinkOf(toCarol));
  const accept = `/api/v0/workspace-invitations/${toCarol.id}/accept`;
  const refused = await answerTo(page, "POST", accept, () => page.getByRole("button", { name: "Accept" }).click());
  expect(refused.status()).toBe(403);
  await expect(page.getByText("This invitation was sent to another email address.")).toBeVisible();
  await expect(page.getByText(carolEmail)).toHaveCount(0);
  await expect(page.getByRole("button", { name: /^(Accept|Ignore)$/ })).toHaveCount(0);
  await expectMembership(db, slug, daveEmail, null);
  await expectInvitations(db, slug, adminEmail, [
    { email: carolEmail, role: 15, accepted: false, responded: false, deleted: false },
  ]);
  // Signed out, the link offers to sign in as the invitee.
  await page.getByRole("button", { name: "Sign out" }).click();
  await expect(page.getByRole("button", { name: "Sign in to accept" })).toBeVisible();

  // A token changed in one character: nerve finds no invitation. No token: the page does not ask.
  const changed = toCarol.token.slice(0, 10) + (toCarol.token[10] === "A" ? "B" : "A") + toCarol.token.slice(11);
  await page.goto(invitationLinkOf({ id: toCarol.id, token: changed }));
  await expect(page.getByText("This invitation link is not valid.")).toBeVisible();
  await page.goto(`/workspace-invitations?invitation_id=${toCarol.id}`);
  await expect(page.getByText("This invitation link is not valid.")).toBeVisible();
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([
    [`403 POST ${accept}`, `404 GET /api/v0/workspace-invitations/${toCarol.id}`],
    [],
    [],
  ]);
  await expectQuietConsole(page, watch, {
    errors: [
      "Failed to load resource: the server responded with a status of 403 (Forbidden)",
      "Failed to load resource: the server responded with a status of 404 (Not Found)",
    ],
  });
});
