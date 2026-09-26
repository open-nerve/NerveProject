import { accountOf, accountStateOf, expectEmailChanged, sessionOf, tokensOf } from "../../fixtures/assert/identity";
import { bearer, createPAT, emailFor, login, password, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";
import { nerveUsers, nerveUsersFails } from "../../fixtures/users";

// A16, the administrator changes an address (M2 design 2, decision 1): the
// only way to change one. There is no page version.

test("A16: nerve users set-email changes the address and signs every session out; the tokens stay", async ({
  api,
  db,
}, testInfo) => {
  const email = emailFor(testInfo);
  const first = await register(api, email);
  const pat = await createPAT(api, first.access_token);
  // Another account, which nothing here changes.
  const taken = emailFor(testInfo, "taken");
  await register(api, taken);
  const takenBefore = await accountStateOf(db, taken);
  const newEmail = emailFor(testInfo, "new");
  const before = await accountOf(db, email);
  const tokensBefore = await tokensOf(db, before.id);

  // Another account's address, whatever its case: exit code 1, and nothing changes.
  await nerveUsersFails(
    db,
    ["set-email", "--email", email, "--new-email", taken.toUpperCase()],
    "An account with this e-mail address already exists."
  );
  expect(await accountOf(db, email)).toEqual(before);
  expect((await sessionOf(db, first.refresh_token)).revoked_at).toBeNull();
  expect(await accountStateOf(db, taken), "the other account").toEqual(takenBefore);

  expect(await nerveUsers(db, ["set-email", "--email", email, "--new-email", newEmail.toUpperCase()])).toBe(
    `email changed from ${email} to ${newEmail}: revoked 1 sessions\n`
  );
  await expectEmailChanged(db, before, newEmail, tokensBefore);
  expect(await accountStateOf(db, taken), "the other account").toEqual(takenBefore);

  // The old address no longer signs in, the new one does; the old refresh
  // token fails; the token goes on, and sees the new address.
  expect((await api.POST("/api/v0/auth/login", { body: { email, password } })).response.status).toBe(401);
  await login(api, newEmail);
  const refreshed = await api.POST("/api/v0/auth/refresh", { body: { refresh_token: first.refresh_token } });
  expect(refreshed.response.status).toBe(401);
  const me = await api.GET("/api/v0/me", { headers: bearer(pat.token) });
  expect(me.response.status).toBe(200);
  expect(me.data?.email).toBe(newEmail);
});
