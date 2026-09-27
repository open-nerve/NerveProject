import { accountOf, accountStateOf, expectPasswordReset } from "../../fixtures/assert/identity";
import { bearer, createPAT, emailFor, login, password, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";
import { nerveUsers } from "../../fixtures/users";

// A13, the administrator resets a password (M2 design 2, 3.5): the way back
// into an account whose credentials leaked. There is no page version.

const newPassword = "N3w-Passw0rd!";

test("A13: nerve users reset-password sets the password from stdin and revokes every session and token", async ({
  api,
  db,
}, testInfo) => {
  const email = emailFor(testInfo);
  const first = await register(api, email);
  const second = await login(api, email);
  const pat = await createPAT(api, first.access_token);
  // Another account, which nothing here changes.
  const other = emailFor(testInfo, "other");
  const otherPAT = await createPAT(api, (await register(api, other)).access_token);
  const otherBefore = await accountStateOf(db, other);
  const before = await accountOf(db, email);

  expect(await nerveUsers(db, ["reset-password", "--email", email], `${newPassword}\n`)).toBe(
    `password reset for ${email}: revoked 2 sessions, 1 API tokens\n`
  );
  await expectPasswordReset(db, before);
  expect(await accountStateOf(db, other), "the other account").toEqual(otherBefore);
  expect((await api.GET("/api/v0/me", { headers: bearer(otherPAT.token) })).response.status).toBe(200);

  // The new password signs in, the old one not; the old refresh token and
  // the old token fail.
  expect((await api.POST("/api/v0/auth/login", { body: { email, password } })).response.status).toBe(401);
  const renewed = await api.POST("/api/v0/auth/login", { body: { email, password: newPassword } });
  expect(renewed.response.status).toBe(200);
  const refreshed = await api.POST("/api/v0/auth/refresh", { body: { refresh_token: second.refresh_token } });
  expect(refreshed.response.status).toBe(401);
  expect((await api.GET("/api/v0/me", { headers: bearer(pat.token) })).response.status).toBe(401);
});
