import { createWorkspace, invite, slugFor, type Api, type WorkspaceInvitation } from "../../fixtures/api";
import { expectInvitations, expectMembership } from "../../fixtures/assert/workspace";
import { bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// W5, accept or decline an invitation by its link (M3 design 2, 3.8; decision 1). The page version comes with
// /workspace-invitations (P9).

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
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin, { name: "Acme", slug });
  const carolEmail = emailFor(testInfo, "carol");
  const carol = (await createPAT(api, (await register(api, carolEmail)).access_token)).token;
  const daveEmail = emailFor(testInfo, "dave");
  const dave = (await createPAT(api, (await register(api, daveEmail)).access_token)).token;
  const [toCarol, toDave] = await invite(api, admin, slug, [
    { email: carolEmail, role: 5 },
    { email: daveEmail, role: 15 },
  ]);
  if (!toCarol || !toDave) {
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
  // dave declines: no membership; the link shows it declined.
  expect((await answer(api, dave, "decline", toDave)).response.status).toBe(204);
  await expectMembership(db, slug, daveEmail, null);
  expect((await view(toDave.id, toDave.token)).data?.declined).toBe(true);
  await expectInvitations(db, slug, adminEmail, [
    { email: carolEmail, role: 5, accepted: true, responded: true, deleted: true },
    { email: daveEmail, role: 15, accepted: false, responded: true, deleted: false },
  ]);

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
