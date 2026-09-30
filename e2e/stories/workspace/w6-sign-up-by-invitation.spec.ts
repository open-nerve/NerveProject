import { accept, createApi, createWorkspace, invite, slugFor } from "../../fixtures/api";
import { countIdentity, expectNothingAdded } from "../../fixtures/assert/identity";
import { expectInvitations, expectMembership } from "../../fixtures/assert/workspace";
import { bearer, createPAT, emailFor, password, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// W6, sign up by an invitation while sign-up is off (M3 design 2, 3.8; decision 1). The page version comes with
// /workspace-invitations (P9).

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

  const signUp = (email: string, invitation?: { id: string; token: string }) =>
    closed.POST("/api/v0/auth/register", { body: { email, password, invitation } });
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
      return [label, r.response.status, r.error?.code];
    })
  );
  expect(answers).toEqual(Object.keys(refusals).map((label) => [label, 403, "identity.signup_disabled"]));
  await expectNothingAdded(db, before);

  // carol registers with her link, the address as she types it; the invitation stays to be answered.
  const registered = await signUp(` ${carolEmail.toUpperCase()} `, { id: toCarol.id, token: toCarol.token });
  expect(registered.response.status).toBe(201);
  await expectInvitations(db, slug, adminEmail, [
    { email: carolEmail, role: 15, accepted: false, responded: false, deleted: false },
    { email: daveEmail, role: 15, accepted: false, responded: true, deleted: false },
    { email: erinEmail, role: 5, accepted: false, responded: false, deleted: true },
  ]);
  await expectMembership(db, slug, carolEmail, null);
  expect(await accept(closed, registered.data?.access_token ?? "", toCarol)).toMatchObject({
    slug,
    role: 15,
    total_members: 2,
  });
  await expectMembership(db, slug, carolEmail, { role: 15, is_active: true });
});
