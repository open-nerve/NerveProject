import { accountOf, expectPasswordChanged, tokensOf } from "../../fixtures/assert/identity";
import { bearer, createPAT, emailFor, password, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// A7, changing the password (M2 design 2). The page version, which keeps
// its own session, joins in M2/P5.

const newPassword = "N3w-Passw0rd!";

test("A7 (API): a personal access token changes the password; every session ends, the token goes on", async ({
  api,
  db,
}, testInfo) => {
  const email = emailFor(testInfo);
  const pat = await createPAT(api, (await register(api, email)).access_token);
  const change = (current: string) =>
    api.POST("/api/v0/me/change-password", {
      body: { current_password: current, new_password: newPassword },
      headers: bearer(pat.token),
    });
  const before = await accountOf(db, email);
  const tokensBefore = await tokensOf(db, before.id);

  // A wrong current password: its problem, and nothing changes.
  const wrong = await change("Wr0ng-password");
  expect(wrong.response.status).toBe(422);
  expect(wrong.error?.code).toBe("identity.current_password_incorrect");
  expect(await accountOf(db, email)).toEqual(before);

  const changed = await change(password);
  expect(changed.response.status).toBe(204);
  // A token has no session of its own: every session ends (M2 design 3.5).
  await expectPasswordChanged(db, before, tokensBefore);

  // The token goes on; the old password no longer signs in, the new one does.
  expect((await api.GET("/api/v0/me", { headers: bearer(pat.token) })).response.status).toBe(200);
  const old = await api.POST("/api/v0/auth/login", { body: { email, password } });
  expect(old.response.status).toBe(401);
  const renewed = await api.POST("/api/v0/auth/login", { body: { email, password: newPassword } });
  expect(renewed.response.status).toBe(200);
});
