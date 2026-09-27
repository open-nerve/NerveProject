import { accountOf, expectPasswordChanged, tokensOf } from "../../fixtures/assert/identity";
import { bearer, createPAT, emailFor, login, password, recordOf, register } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import { registerOnboarded, submitPasswordChange } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// A7, changing the password (M2 design 2). The page keeps its own session,
// which a token has not: the two versions differ in that alone, which the
// shared assertion takes as survivingSession (Codex M-6).

const newPassword = "N3w-Passw0rd!";

/** What the browser logs of an answer of 422, which the page shows under a field. */
const refusedResourceError =
  "Failed to load resource: the server responded with a status of 422 (Unprocessable Entity)";

test("A7 (page): the security page changes the password, keeps its own session, and lists the tokens it leaves alone", async ({
  api,
  db,
  signedInPage,
}, testInfo) => {
  const email = emailFor(testInfo);
  const tokens = await registerOnboarded(api, email);
  // Another session of the account, which the change ends, and a token, which it leaves (M2 design 3.5).
  await login(api, email);
  const pat = await createPAT(api, tokens.access_token, { label: "deploy" });
  const page = await signedInPage(tokens);
  const watch = await watchPage(page);
  await page.goto("/settings/profile/security");

  // The page lists the account's tokens, and says the change leaves them.
  await expect(page.getByText("deploy", { exact: true })).toBeVisible();
  await expect(page.getByText("Changing your password does not revoke these tokens", { exact: false })).toBeVisible();
  const before = await accountOf(db, email);
  const tokensBefore = await tokensOf(db, before.id);

  // A wrong current password shows under its field; a new password too common, under the new one. Nothing
  // changes.
  expect(await submitPasswordChange(page, "Wr0ng-password", newPassword)).toBe(422);
  await expect(page.getByText("The current password is wrong.")).toBeVisible();
  expect(await submitPasswordChange(page, password, "Password1!")).toBe(422);
  await expect(page.getByText("This password is too common")).toBeVisible();
  await expect(page.getByText("The current password is wrong.")).toHaveCount(0);
  expect(await accountOf(db, email)).toEqual(before);

  expect(await submitPasswordChange(page, password, newPassword)).toBe(204);
  await expect(page.getByText("Password changed successfully.")).toBeVisible();
  // The page's session goes on; every other ends; the token is as it was.
  const held = (await recordOf(page))?.refresh_token;
  expect(held, "the page's refresh token").toBeDefined();
  await expectPasswordChanged(db, before, tokensBefore, held);
  await page.reload();
  await expect(page).toHaveURL("/settings/profile/security");
  await expect(page.getByText("deploy", { exact: true })).toBeVisible();
  expect((await api.GET("/api/v0/me", { headers: bearer(pat.token) })).response.status).toBe(200);
  const old = await api.POST("/api/v0/auth/login", { body: { email, password } });
  expect(old.response.status).toBe(401);

  expect(watch.apiFailures).toEqual(["422 POST /api/v0/me/change-password", "422 POST /api/v0/me/change-password"]);
  expect(watch.pageErrors).toEqual([]);
  await expectQuietConsole(page, watch, {
    errors: [refusedResourceError, refusedResourceError],
    warnings: [EMOJI_CHECK_WARNING, EMOJI_CHECK_WARNING],
  });
});

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
