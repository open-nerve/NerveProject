import { randomUUID } from "node:crypto";

import type { Api } from "../../fixtures/api";
import { accountOf, accountStateOf, expectDeactivated, tokensOf } from "../../fixtures/assert/identity";
import { bearer, createPAT, emailFor, login, password, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";
import { nerveUsers } from "../../fixtures/users";

// A12, deactivating an account (M2 design 2, decision 3): the API version,
// with the administrator's nerve users activate and deactivate. The page
// version joins in M2/P5.

/** Finishes onboarding, so that starting it over shows. */
async function onboard(api: Api, token: string): Promise<void> {
  const { response } = await api.PATCH("/api/v0/me/profile", {
    body: {
      onboarding_step: { profile_complete: true, workspace_create: true, workspace_invite: true, workspace_join: true },
      is_onboarded: true,
      is_tour_completed: true,
      last_workspace_id: randomUUID(),
    },
    headers: bearer(token),
  });
  expect(response.status).toBe(200);
}

test("A12 (API): a token deactivates the account; nerve users activate brings it and its tokens back, deactivate does as the API did", async ({
  api,
  db,
}, testInfo) => {
  const email = emailFor(testInfo);
  const pat = await createPAT(api, (await register(api, email)).access_token);
  await onboard(api, pat.token);
  // Another account, which nothing here changes.
  const other = emailFor(testInfo, "other");
  await register(api, other);
  const otherBefore = await accountStateOf(db, other);
  const before = await accountOf(db, email);
  const tokensBefore = await tokensOf(db, before.id);

  const deactivated = await api.POST("/api/v0/me/deactivate", { headers: bearer(pat.token) });
  expect(deactivated.response.status).toBe(204);
  await expectDeactivated(db, before, tokensBefore);
  expect(await accountStateOf(db, other), "the other account").toEqual(otherBefore);

  // Nothing authenticates as the account: the token fails, the password is refused.
  expect((await api.GET("/api/v0/me", { headers: bearer(pat.token) })).response.status).toBe(401);
  const refused = await api.POST("/api/v0/auth/login", { body: { email, password } });
  expect(refused.response.status).toBe(403);
  expect(refused.error?.code).toBe("identity.account_deactivated");

  // The administrator activates it: the same token and password work again.
  expect(await nerveUsers(db, ["activate", "--email", email])).toBe(
    `activated ${email}: 1 API tokens are usable again\n`
  );
  expect(await accountStateOf(db, other), "the other account").toEqual(otherBefore);
  expect((await api.GET("/api/v0/me", { headers: bearer(pat.token) })).response.status).toBe(200);
  await login(api, email);

  // The administrator's deactivate leaves the database as the API did.
  await onboard(api, pat.token);
  const again = await accountOf(db, email);
  expect(await nerveUsers(db, ["deactivate", "--email", email])).toBe(`deactivated ${email}: revoked 1 sessions\n`);
  await expectDeactivated(db, again, tokensBefore);
  expect(await accountStateOf(db, other), "the other account").toEqual(otherBefore);
});
