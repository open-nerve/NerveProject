import {
  createWorkspace,
  decline,
  invitationTo,
  invite,
  inviteAndAccept,
  slugFor,
  type InvitationCreate,
} from "../../fixtures/api";
import { expectInvitations } from "../../fixtures/assert/workspace";
import { bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import { registerOnboarded, sentTo } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";
import {
  anotherBrowser,
  invitationLinkOf,
  invitationRow,
  memberRow,
  removeInvitation,
  sendInvitations,
} from "../../fixtures/workspace-pages";

// W4, invite members (M3 design 2, 3.8; decision 4: admins only).

test("W4 (API): the admin invites a batch, changes a role and deletes an invitation; an active member's address, a repeated or declined one is refused; members and guests may not invite", async ({
  api,
  db,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = (await createPAT(api, (await register(api, adminEmail)).access_token)).token;
  const memberEmail = emailFor(testInfo, "member");
  const member = (await createPAT(api, (await register(api, memberEmail)).access_token)).token;
  const guestEmail = emailFor(testInfo, "guest");
  const guest = (await createPAT(api, (await register(api, guestEmail)).access_token)).token;
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin, { name: "Acme", slug });
  await inviteAndAccept(api, admin, slug, { email: memberEmail, token: member }, 15);
  await inviteAndAccept(api, admin, slug, { email: guestEmail, token: guest }, 5);
  const accepted = [
    { email: memberEmail, role: 15, accepted: true, responded: true, deleted: true },
    { email: guestEmail, role: 5, accepted: true, responded: true, deleted: true },
  ];

  // Another workspace, frank's: he is its active member and carol is invited there. Neither counts in Acme.
  const frank = emailFor(testInfo, "frank");
  const franksToken = (await register(api, frank)).access_token;
  const carol = emailFor(testInfo, "carol");
  const other = slugFor(testInfo, "other");
  await createWorkspace(api, franksToken, { name: "Other", slug: other });
  const othersInvitations = await invite(api, franksToken, other, [{ email: carol, role: 5 }]);

  // A batch of two, the addresses as a person types them: stored normalized, pending, in the request's order.
  const dave = emailFor(testInfo, "dave");
  const created = await invite(api, admin, slug, [
    { email: ` ${carol.toUpperCase()} `, role: 15 },
    { email: dave, role: 5 },
  ]);
  expect(created.map((i) => [i.email, i.role, i.accepted, i.responded_at])).toEqual([
    [carol, 15, false, null],
    [dave, 5, false, null],
  ]);
  // Every link the story holds: no row stores any of them.
  const tokens = [...othersInvitations, ...created].map((i) => i.token);
  await expectInvitations(
    db,
    slug,
    adminEmail,
    [
      ...accepted,
      { email: carol, role: 15, accepted: false, responded: false, deleted: false },
      { email: dave, role: 5, accepted: false, responded: false, deleted: false },
    ],
    tokens
  );

  // One role changed, the other invitation deleted; the list shows what is left, with its link.
  const [carolsInvitation, davesInvitation] = created;
  const changed = await api.PATCH("/api/v0/workspace-invitations/{invitation_id}", {
    params: { path: { invitation_id: carolsInvitation?.id ?? "" } },
    body: { role: 20 },
    headers: bearer(admin),
  });
  expect(changed.response.status).toBe(200);
  expect(changed.data).toMatchObject({ email: carol, role: 20, token: carolsInvitation?.token });
  const deleted = await api.DELETE("/api/v0/workspace-invitations/{invitation_id}", {
    params: { path: { invitation_id: davesInvitation?.id ?? "" } },
    headers: bearer(admin),
  });
  expect(deleted.response.status).toBe(204);
  const listed = await api.GET("/api/v0/workspaces/{slug}/invitations", {
    params: { path: { slug } },
    headers: bearer(admin),
  });
  expect(listed.data?.data.map((i) => [i.email, i.role, i.token])).toEqual([[carol, 20, carolsInvitation?.token]]);

  // A declined invitation holds its address until it is deleted.
  const erinEmail = emailFor(testInfo, "erin");
  const erin = (await register(api, erinEmail)).access_token;
  const [erinsInvitation] = await invite(api, admin, slug, [{ email: erinEmail, role: 15 }]);
  if (!erinsInvitation) {
    throw new Error("the invitation of erin was not created");
  }
  tokens.push(erinsInvitation.token);
  const declined = await api.POST("/api/v0/workspace-invitations/{invitation_id}/decline", {
    params: { path: { invitation_id: erinsInvitation.id } },
    body: { token: erinsInvitation.token },
    headers: bearer(erin),
  });
  expect(declined.response.status).toBe(204);

  // Refused as a whole, each address at its index. A batch that repeats an address is refused as it is sent,
  // before the workspace is read, alone: the active member's address in it is not looked at. Then an active member's
  // address, a pending and a declined invitation's. frank, a member of another workspace, is not refused.
  const refuse = (invitations: InvitationCreate[]) =>
    api.POST("/api/v0/workspaces/{slug}/invitations", {
      params: { path: { slug } },
      body: { invitations },
      headers: bearer(admin),
    });
  const repeated = await refuse([
    { email: frank, role: 15 },
    { email: frank.toUpperCase(), role: 5 },
    { email: memberEmail, role: 15 },
  ]);
  expect([repeated.response.status, repeated.error?.errors?.map((e) => [e.field, e.code])]).toEqual([
    422,
    [["invitations[1].email", "duplicate"]],
  ]);
  const taken = await refuse([
    { email: frank, role: 15 },
    { email: memberEmail, role: 15 },
    { email: carol, role: 5 },
    { email: erinEmail, role: 5 },
  ]);
  expect([taken.response.status, taken.error?.errors?.map((e) => [e.field, e.code])]).toEqual([
    422,
    [
      ["invitations[1].email", "not_allowed"],
      ["invitations[2].email", "duplicate"],
      ["invitations[3].email", "duplicate"],
    ],
  ]);

  // Members and guests may not invite, list, change or delete.
  const invitationId = { invitation_id: carolsInvitation?.id ?? "" };
  const byOthers = await Promise.all(
    [member, guest].flatMap((token) => [
      api.POST("/api/v0/workspaces/{slug}/invitations", {
        params: { path: { slug } },
        body: { invitations: [{ email: emailFor(testInfo, "gina"), role: 5 }] },
        headers: bearer(token),
      }),
      api.GET("/api/v0/workspaces/{slug}/invitations", { params: { path: { slug } }, headers: bearer(token) }),
      api.PATCH("/api/v0/workspace-invitations/{invitation_id}", {
        params: { path: invitationId },
        body: { role: 5 },
        headers: bearer(token),
      }),
      api.DELETE("/api/v0/workspace-invitations/{invitation_id}", {
        params: { path: invitationId },
        headers: bearer(token),
      }),
    ])
  );
  expect(byOthers.map((r) => [r.response.status, r.error?.code])).toEqual(byOthers.map(() => [403, "forbidden"]));
  await expectInvitations(
    db,
    slug,
    adminEmail,
    [
      ...accepted,
      { email: carol, role: 20, accepted: false, responded: false, deleted: false },
      { email: dave, role: 5, accepted: false, responded: false, deleted: true },
      { email: erinEmail, role: 15, accepted: false, responded: true, deleted: false },
    ],
    tokens
  );
});

test("W4 (page): the admin invites a member and a guest, changes a role and copies a link; an invitation deleted meanwhile is refused with the reason; an active member's address and a declined invitation's are refused under their rows; the declined one deleted, its address is invited again; a member sees no invitation and asks for none", async ({
  api,
  baseURL,
  browser,
  context,
  db,
  signedInPage,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = await registerOnboarded(api, adminEmail);
  const memberEmail = emailFor(testInfo, "member");
  const member = await registerOnboarded(api, memberEmail);
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.access_token, { name: "Acme", slug });
  await inviteAndAccept(api, admin.access_token, slug, { email: memberEmail, token: member.access_token }, 15);
  // erin declined an invitation to Acme.
  const erinEmail = emailFor(testInfo, "erin");
  await invite(api, admin.access_token, slug, [{ email: erinEmail, role: 15 }]);
  const toErin = await invitationTo(api, admin.access_token, slug, erinEmail);
  await decline(api, (await registerOnboarded(api, erinEmail)).access_token, toErin);
  const [dave, frank] = [emailFor(testInfo, "dave"), emailFor(testInfo, "frank")];
  const invitations = `/api/v0/workspaces/${slug}/invitations`;

  // The member's page shows no invitation, and asks nerve for none.
  const theMember = await anotherBrowser(browser, baseURL ?? "", member);
  const memberWatch = await watchPage(theMember.page);
  await theMember.page.goto(`/${slug}/settings/members`);
  await expect(memberRow(theMember.page, adminEmail)).toBeVisible();
  await expect(theMember.page.getByText("Pending invites")).toHaveCount(0);
  await expect(theMember.page.getByRole("button", { name: "Add member" })).toHaveCount(0);
  expect(memberWatch.apiRequests.filter((request) => request.endsWith(invitations))).toEqual([]);
  expect([memberWatch.apiFailures, memberWatch.oldApiRequests, memberWatch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(theMember.page, memberWatch, { warnings: [EMOJI_CHECK_WARNING] });
  await theMember.close();

  await context.grantPermissions(["clipboard-read", "clipboard-write"]);
  const page = await signedInPage(admin);
  const watch = await watchPage(page);
  await page.goto(`/${slug}/settings/members`);
  // erin's invitation says it was declined, and offers neither a role nor a link.
  await expect(invitationRow(page, erinEmail)).toContainText("Declined");
  await expect(invitationRow(page, erinEmail).getByRole("button", { name: "Member" })).toHaveCount(0);

  // dave as a member, frank as a guest: one request, nerve's WorkspaceInvitationsCreate, each role a number.
  const sent = await sendInvitations(page, slug, [
    { email: dave, role: "Member" },
    { email: frank, role: "Guest" },
  ]);
  expect([sent.answer.status(), sent.body]).toEqual([
    201,
    {
      invitations: [
        { email: dave, role: 15 },
        { email: frank, role: 5 },
      ],
    },
  ]);
  await expect(page.getByText("Invitations sent successfully")).toBeVisible();
  const davesInvitation = await invitationTo(api, admin.access_token, slug, dave);
  const franksInvitation = await invitationTo(api, admin.access_token, slug, frank);
  await expect(invitationRow(page, dave)).toContainText("Pending");

  // frank becomes a member; dave's link goes to the clipboard.
  await invitationRow(page, frank).getByRole("button", { name: "Guest" }).click();
  const changed = await sentTo(page, "PATCH", `/api/v0/workspace-invitations/${franksInvitation.id}`, () =>
    page.getByRole("option", { name: "Member", exact: true }).click()
  );
  expect([changed.answer.status(), changed.body]).toEqual([200, { role: 15 }]);
  await invitationRow(page, dave).getByRole("button").last().click();
  await page.getByText("Copy link", { exact: true }).click();
  await expect(page.getByText("Invite link copied to clipboard")).toBeVisible();
  expect(await page.evaluate(() => navigator.clipboard.readText())).toBe(
    `${baseURL}${invitationLinkOf(davesInvitation)}`
  );
  // frank's invitation, deleted meanwhile through the API: the page's removal is refused, and says why.
  const removal = { method: "DELETE", path: `/api/v0/workspace-invitations/${franksInvitation.id}` };
  const meanwhile = await api.DELETE("/api/v0/workspace-invitations/{invitation_id}", {
    params: { path: { invitation_id: franksInvitation.id } },
    headers: bearer(admin.access_token),
  });
  expect(meanwhile.response.status).toBe(204);
  expect((await removeInvitation(page, frank, removal)).status()).toBe(404);
  await expect(page.getByText("The invitation does not exist, or its link is not valid.")).toBeVisible();

  // The member's address and erin's: refused, each under its row, and the form stays open.
  const refused = await sendInvitations(page, slug, [
    { email: memberEmail, role: "Member" },
    { email: erinEmail, role: "Member" },
  ]);
  expect(refused.answer.status()).toBe(422);
  const form = page.getByRole("dialog");
  await expect(form.getByText("Already a member of this workspace.")).toBeVisible();
  await expect(
    form.getByText("This address is invited already (delete that invitation first), or given twice here.")
  ).toBeVisible();
  await form.getByRole("button", { name: "Cancel" }).click();
  // erin's declined invitation deleted, her address is invited again.
  const erins = { method: "DELETE", path: `/api/v0/workspace-invitations/${toErin.id}` };
  expect((await removeInvitation(page, erinEmail, erins)).status()).toBe(204);
  const again = await sendInvitations(page, slug, [{ email: erinEmail, role: "Guest" }]);
  expect([again.answer.status(), again.body]).toEqual([201, { invitations: [{ email: erinEmail, role: 5 }] }]);
  await expect(invitationRow(page, erinEmail)).toContainText("Pending");

  const accepted = { email: memberEmail, role: 15, accepted: true, responded: true, deleted: true };
  await expectInvitations(db, slug, adminEmail, [
    accepted,
    { email: dave, role: 15, accepted: false, responded: false, deleted: false },
    { email: erinEmail, role: 15, accepted: false, responded: true, deleted: true },
    { email: erinEmail, role: 5, accepted: false, responded: false, deleted: false },
    { email: frank, role: 15, accepted: false, responded: false, deleted: true },
  ]);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([
    [`404 ${removal.method} ${removal.path}`, `422 POST ${invitations}`],
    [],
    [],
  ]);
  await expectQuietConsole(page, watch, {
    warnings: [EMOJI_CHECK_WARNING],
    errors: [
      "Failed to load resource: the server responded with a status of 404 (Not Found)",
      "Failed to load resource: the server responded with a status of 422 (Unprocessable Entity)",
    ],
  });
});
