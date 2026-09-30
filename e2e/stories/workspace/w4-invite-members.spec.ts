import { createWorkspace, invite, inviteAndAccept, slugFor, type InvitationCreate } from "../../fixtures/api";
import { expectInvitations } from "../../fixtures/assert/workspace";
import { bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// W4, invite members (M3 design 2, 3.8; decision 4: admins only). The page
// version comes with the members page (P9).

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
