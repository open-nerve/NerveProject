import { randomUUID } from "node:crypto";

import type { Api } from "../../fixtures/api";
import { accountOf, accountStateOf, expectDeactivated, tokensOf } from "../../fixtures/assert/identity";
import { formAlert, signInPath, submitSignIn } from "../../fixtures/auth-pages";
import { bearer, createPAT, emailFor, login, password, recordOf, register } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import { answerTo, registerOnboarded } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";
import { nerveUsers } from "../../fixtures/users";

// A12, deactivating an account (M2 design 2, decision 3), with the
// administrator's nerve users activate and deactivate.

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

test("A12 (page): the general page deactivates the account once confirmed; the session ends, sign-in says so, and nerve users activate lets it in again", async ({
  api,
  db,
  signedInPage,
}, testInfo) => {
  const email = emailFor(testInfo);
  const tokens = await registerOnboarded(api, email);
  const pat = await createPAT(api, tokens.access_token);
  // Another session of the account, which ends too.
  await login(api, email);
  const page = await signedInPage(tokens);
  const watch = await watchPage(page);
  await page.goto("/settings/profile/general");
  const before = await accountOf(db, email);
  const tokensBefore = await tokensOf(db, before.id);

  // The confirmation says what deactivating does, as nerve does it (decision 3).
  await page.getByRole("button", { name: "Deactivate account" }).click();
  await expect(page.getByText("ask an administrator of this server to reactivate it", { exact: false })).toBeVisible();
  const deactivated = await answerTo(page, "POST", "/api/v0/me/deactivate", () =>
    page.getByRole("button", { name: "Confirm" }).click()
  );
  expect(deactivated.status()).toBe(204);

  // The page is back at sign-in, which comes back to the general page, and the browser keeps no session.
  await expect(page).toHaveURL(signInPath("/settings/profile/general"));
  await expect(page.getByText("Your account is deactivated.")).toBeVisible();
  expect(await recordOf(page)).toBeNull();
  await expectDeactivated(db, before, tokensBefore);

  // Signing in again: the account is deactivated.
  expect(await submitSignIn(page, email, password)).toBe(403);
  await expect(formAlert(page)).toHaveText("This account is deactivated.");

  // The administrator activates it: it signs in, into onboarding, which deactivating started over.
  expect(await nerveUsers(db, ["activate", "--email", email])).toBe(
    `activated ${email}: 1 API tokens are usable again\n`
  );
  expect(await submitSignIn(page, email, password)).toBe(200);
  await expect(page).toHaveURL("/onboarding");
  expect((await api.GET("/api/v0/me", { headers: bearer(pat.token) })).response.status).toBe(200);

  expect(watch.apiFailures).toEqual(["403 POST /api/v0/auth/login"]);
  expect(watch.pageErrors).toEqual([]);
  await expectQuietConsole(page, watch, {
    errors: ["Failed to load resource: the server responded with a status of 403 (Forbidden)"],
    warnings: [EMOJI_CHECK_WARNING],
  });
});

test("A12 (page): a deactivation nerve fails says why; the account and the page's session stay", async ({
  api,
  db,
  signedInPage,
}, testInfo) => {
  const email = emailFor(testInfo);
  const page = await signedInPage(await registerOnboarded(api, email));
  const watch = await watchPage(page);
  await page.goto("/settings/profile/general");
  await expect(page.getByRole("button", { name: "Deactivate account" })).toBeVisible();
  const before = await accountStateOf(db, email);
  const held = await recordOf(page);
  expect(held, "the page's session").not.toBeNull();
  // nerve fails the deactivation.
  await page.route("**/api/v0/me/deactivate", (route) =>
    route.fulfill({
      status: 500,
      contentType: "application/problem+json",
      body: JSON.stringify({
        type: "about:blank",
        title: "Internal Server Error",
        status: 500,
        code: "internal_error",
      }),
    })
  );

  await page.getByRole("button", { name: "Deactivate account" }).click();
  const refused = await answerTo(page, "POST", "/api/v0/me/deactivate", () =>
    page.getByRole("button", { name: "Confirm" }).click()
  );
  expect(refused.status()).toBe(500);

  // The toast says why, by the problem's code; the confirmation stays open, to confirm again or cancel; the page
  // stays signed in, on the general page; nothing changed.
  await expect(page.getByText("Something went wrong on the server. Please try again.")).toBeVisible();
  await expect(page.getByRole("button", { name: "Confirm" })).toBeVisible();
  await expect(page).toHaveURL("/settings/profile/general");
  expect(await recordOf(page)).toEqual(held);
  expect(await accountStateOf(db, email)).toEqual(before);

  expect(watch.apiFailures).toEqual(["500 POST /api/v0/me/deactivate"]);
  expect(watch.oldApiRequests).toEqual([]);
  expect(watch.pageErrors).toEqual([]);
  await expectQuietConsole(page, watch, {
    warnings: [EMOJI_CHECK_WARNING],
    errors: ["Failed to load resource: the server responded with a status of 500 (Internal Server Error)"],
  });
});

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
