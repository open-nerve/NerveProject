import type { Browser, Page } from "@playwright/test";

import { countIdentity, expectNothingAdded, expectSignedIn } from "../../fixtures/assert/identity";
import { formAlert, signInPath, submitSignIn } from "../../fixtures/auth-pages";
import { bearer, emailFor, login, password, recordOf, register } from "../../fixtures/auth";
import { watchPage } from "../../fixtures/browser";
import { expect, test } from "../../fixtures/test";

// A3, signing in (M2 design 2), with next_path (3.18).

/** A page in a browser context of its own: signed out, whatever the other pages did. */
async function freshPage(browser: Browser, baseURL: string): Promise<Page> {
  const context = await browser.newContext({ baseURL });
  return context.newPage();
}

test("A3 (page): signing in comes back to the page asked for, and only to a page of this site", async ({
  browser,
  api,
  db,
  nerve,
}, testInfo) => {
  const email = emailFor(testInfo, "Alice");
  const tokens = await register(api, email);
  // Onboarded, so that the settings open once signed in.
  const onboarded = await api.PATCH("/api/v0/me/profile", {
    body: { is_onboarded: true },
    headers: bearer(tokens.access_token),
  });
  expect(onboarded.response.status).toBe(200);

  // Signed out, a page behind the sign-in goes to the sign-in page; the page comes back after it, with
  // its query and fragment.
  const asked = "/settings/profile/general?tab=x#y";
  const page = await freshPage(browser, nerve.baseURL);
  await page.goto(asked);
  await expect(page).toHaveURL(signInPath(asked));
  expect(signInPath(asked)).toBe("/?next_path=%2Fsettings%2Fprofile%2Fgeneral%3Ftab%3Dx%23y");
  expect(await submitSignIn(page, email, password)).toBe(200);
  await expect(page).toHaveURL(asked);
  const userAgent = await page.evaluate(() => navigator.userAgent);
  await expectSignedIn(db, {
    email,
    refreshToken: (await recordOf(page))?.refresh_token ?? "",
    userAgent,
    ip: "127.0.0.1",
  });
  await page.context().close();

  // A next_path that could lead elsewhere is dropped: the account's default page instead. That page,
  // /create-workspace, is one M2 reaches, so it asks no older API as it mounts (M2 design 3.1): the
  // workspace addresses answer 404 until M3.
  await Promise.all(
    ["//evil.example", "/\\evil.example", "javascript:alert(1)", "/\t/evil.example"].map(async (nextPath) => {
      const other = await freshPage(browser, nerve.baseURL);
      const watch = await watchPage(other);
      await other.goto(`/?next_path=${encodeURIComponent(nextPath)}`);
      expect(await submitSignIn(other, email, password), nextPath).toBe(200);
      await expect(other, nextPath).toHaveURL("/create-workspace");
      await expect(other.locator("#workspaceName"), nextPath).toBeVisible();
      expect(watch.oldApiRequests, nextPath).toEqual([]);
      expect(watch.apiFailures, nextPath).toEqual([]);
      expect(watch.pageErrors, nextPath).toEqual([]);
      await other.context().close();
    })
  );
});

test("A3 (page): a wrong password and an unknown address get the same message, and no session", async ({
  page,
  api,
  db,
}, testInfo) => {
  const email = emailFor(testInfo, "Alice");
  await register(api, email);
  const before = await countIdentity(db);

  await page.goto("/");
  const expectRefused = async (address: string, pw: string) => {
    expect(await submitSignIn(page, address, pw), address).toBe(401);
    await expect(formAlert(page)).toHaveText("The email or the password is wrong.");
    await expect(page).toHaveURL("/");
    await expect(page.getByLabel("Email", { exact: true })).toHaveValue(address);
  };
  await expectRefused(email, "Wr0ng-password");
  await expectRefused(emailFor(testInfo, "nobody"), password);
  await expectNothingAdded(db, before);
});

test("A3 (API): a caller signs in; a wrong password and an unknown address answer alike", async ({
  api,
  db,
}, testInfo) => {
  const email = emailFor(testInfo, "Alice");
  await register(api, email);
  const userAgent = "nerve-e2e/A3";

  // The address in another case and with blanks around it still signs in.
  const tokens = await login(api, ` ${email.toUpperCase()} `, { "User-Agent": userAgent });

  expect(tokens.token_type).toBe("Bearer");
  await expectSignedIn(db, { email, refreshToken: tokens.refresh_token, userAgent, ip: "127.0.0.1" });
  const me = await api.GET("/api/v0/me", { headers: { Authorization: `Bearer ${tokens.access_token}` } });
  expect(me.response.status).toBe(200);

  const before = await countIdentity(db);
  const refusals = await Promise.all([
    api.POST("/api/v0/auth/login", { body: { email, password: "Wr0ng-password" } }),
    api.POST("/api/v0/auth/login", { body: { email: emailFor(testInfo, "nobody"), password } }),
  ]);
  for (const { response, error } of refusals) {
    expect(response.status).toBe(401);
    expect(error?.code).toBe("identity.invalid_credentials");
    expect(error?.detail).toBe("The e-mail address or the password is incorrect.");
  }

  // A body without a password breaks the contract: the platform's 400.
  const missing = await api.POST("/api/v0/auth/login", {
    // @ts-expect-error -- the request leaves out a required field on purpose
    body: { email },
  });
  expect(missing.response.status).toBe(400);
  expect(missing.error?.code).toBe("bad_request");
  expect(missing.error?.errors?.map((e) => ({ field: e.field, code: e.code }))).toEqual([
    { field: "password", code: "required" },
  ]);
  await expectNothingAdded(db, before);
});
