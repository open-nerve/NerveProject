import type { Page } from "@playwright/test";

import { accountOf, expectRevoked, onboardingStepsOf, sessionOf } from "../../fixtures/assert/identity";
import { signInPath, submitSignIn } from "../../fixtures/auth-pages";
import {
  anotherTabSignsIn,
  bearer,
  emailFor,
  password,
  recordOf,
  refresh,
  register,
  shownNameOf,
} from "../../fixtures/auth";
import { expectQuietConsole, followAccessToken, watchPage, type PageWatch } from "../../fixtures/browser";
import { saveProfileStep } from "../../fixtures/onboarding-pages";
import { expect, test } from "../../fixtures/test";

// A6, signing out and switching accounts (M2 design 2, 7.1).

/** The onboarding steps of an account that has taken none. */
const noStepDone = { profile_complete: false, workspace_create: false, workspace_invite: false, workspace_join: false };

/** Opens /onboarding in tab, which shows the account of email in its header. */
async function openOnboarding(tab: Page, email: string): Promise<void> {
  await tab.goto("/onboarding");
  await expect(tab.getByText("Create your profile.")).toBeVisible();
  await expect(accountMenu(tab, email)).toBeVisible();
}

/** The account menu of the onboarding header, which shows the display name of the account signed in. */
function accountMenu(tab: Page, email: string) {
  return tab.getByRole("button", { name: shownNameOf(email) });
}

/** Signs out through the account menu: "Wrong e-mail address?", then "Switch account" (M2 has no workspace menu). */
async function signOutThroughMenu(tab: Page, email: string): Promise<void> {
  await accountMenu(tab, email).click();
  await tab.getByRole("menuitem", { name: "Wrong e-mail address?" }).click();
  await tab.getByRole("button", { name: "Switch account" }).click();
}

/** Nothing went wrong in the tabs of watched: no API call failed, nothing was left unhandled, the console is quiet. */
async function expectNothingWentWrong(watched: [Page, PageWatch][]): Promise<void> {
  await Promise.all(
    watched.map(async ([tab, watch]) => {
      expect(watch.apiFailures).toEqual([]);
      expect(watch.pageErrors).toEqual([]);
      await expectQuietConsole(tab, watch);
    })
  );
}

test("A6 (page): signing out in one tab signs every tab out", async ({ api, db, context, signedInPage }, testInfo) => {
  const email = emailFor(testInfo);
  const tokens = await register(api, email);
  // A theme the tabs show once the profile arrives, and no longer without a session.
  const themed = await api.PATCH("/api/v0/me/profile", {
    body: { theme: "dark" },
    headers: bearer(tokens.access_token),
  });
  expect(themed.response.status).toBe(200);
  const tabA = await signedInPage(tokens);
  const watchA = await watchPage(tabA);
  const sentAccessToken = followAccessToken(tabA);
  await openOnboarding(tabA, email);
  const tabB = await context.newPage();
  const watchB = await watchPage(tabB);
  await openOnboarding(tabB, email);
  await expect(tabA.locator("html")).toHaveAttribute("data-theme", "dark");
  await expect(tabB.locator("html")).toHaveAttribute("data-theme", "dark");
  const held = (await recordOf(tabA))?.refresh_token ?? "";
  // The access token tab A sent last works until the sign-out.
  const accessToken = sentAccessToken();
  expect(accessToken, "tab A sent an access token").not.toBe("");
  expect((await api.GET("/api/v0/me", { headers: bearer(accessToken) })).response.status).toBe(200);

  await signOutThroughMenu(tabA, email);

  // Both tabs are on the sign-in page, which comes back to where they were, in the default theme; the record is gone.
  await Promise.all(
    [tabA, tabB].map(async (tab) => {
      await expect(tab).toHaveURL(signInPath("/onboarding"));
      await expect(tab.getByRole("button", { name: "Go to workspace" })).toBeVisible();
      await expect(tab.locator("html")).toHaveAttribute("data-theme", "light");
    })
  );
  expect(await recordOf(tabB)).toBeNull();
  await expectRevoked(db, held, "logout");
  // The session's access token fails on the next request.
  const me = await api.GET("/api/v0/me", { headers: bearer(accessToken) });
  expect(me.response.status).toBe(401);
  await expectNothingWentWrong([
    [tabA, watchA],
    [tabB, watchB],
  ]);
});

test("A6 (page): another tab signs another account in without signing out: every tab goes on as that account", async ({
  api,
  db,
  context,
  signedInPage,
}, testInfo) => {
  const x = emailFor(testInfo, "x");
  const y = emailFor(testInfo, "y");
  const tabA = await signedInPage(await register(api, x));
  const watchA = await watchPage(tabA);
  await register(api, y);
  await openOnboarding(tabA, x);
  const xHeld = (await recordOf(tabA))?.refresh_token ?? "";

  // Tab B, on a file of the site, keeps a sign-in of Y as the token manager does (anotherTabSignsIn): a new record,
  // written under the refresh lock, with a new login_id.
  await anotherTabSignsIn(context, api, y);

  // Tab A follows: it shows Y, and what it writes from now on is Y's.
  await expect(accountMenu(tabA, y)).toBeVisible();
  await expect(accountMenu(tabA, x)).toHaveCount(0);
  await tabA.getByLabel("Name", { exact: true }).fill("Yvonne");
  expect(await saveProfileStep(tabA)).toBe(200);
  await expect(tabA.getByText("Create your workspace", { exact: true })).toBeVisible();
  // Both of the step's writes went to Y, the name and then the step done; X's account is as it was.
  expect((await accountOf(db, y.toLowerCase())).first_name).toBe("Yvonne");
  expect(await onboardingStepsOf(db, y.toLowerCase())).toEqual({ ...noStepDone, profile_complete: true });
  expect((await accountOf(db, x.toLowerCase())).first_name).toBe("");
  expect(await onboardingStepsOf(db, x.toLowerCase())).toEqual(noStepDone);
  // X's session is left as it was: nobody signed it out.
  expect((await sessionOf(db, xHeld)).revoked_at).toBeNull();
  await expectNothingWentWrong([[tabA, watchA]]);
});

test("A6 (page): another tab signs out, then signs another account in: every tab comes in as that account", async ({
  api,
  db,
  context,
  signedInPage,
}, testInfo) => {
  const x = emailFor(testInfo, "x");
  const y = emailFor(testInfo, "y");
  const tabA = await signedInPage(await register(api, x));
  const watchA = await watchPage(tabA);
  await register(api, y);
  await openOnboarding(tabA, x);
  const tabB = await context.newPage();
  const watchB = await watchPage(tabB);
  await openOnboarding(tabB, x);
  const xHeld = (await recordOf(tabB))?.refresh_token ?? "";

  await signOutThroughMenu(tabB, x);
  await expect(tabA).toHaveURL(signInPath("/onboarding"));
  await expect(tabB).toHaveURL(signInPath("/onboarding"));
  await expectRevoked(db, xHeld, "logout");

  // Tab B signs Y in; tab A sees the record appear and comes in as Y too, back to where it was.
  expect(await submitSignIn(tabB, y, password)).toBe(200);
  await Promise.all(
    [tabB, tabA].map(async (tab) => {
      await expect(tab).toHaveURL("/onboarding");
      await expect(accountMenu(tab, y)).toBeVisible();
    })
  );
  await expectNothingWentWrong([
    [tabA, watchA],
    [tabB, watchB],
  ]);
});

test("A6 (API): logout ends the session; the previous generation's logout changes nothing", async ({
  api,
  db,
}, testInfo) => {
  const first = await register(api, emailFor(testInfo));
  const second = await refresh(api, first.refresh_token);
  const logout = async (token: string) => {
    const { response } = await api.POST("/api/v0/auth/logout", { body: { refresh_token: token } });
    expect(response.status).toBe(204);
  };

  // The previous generation: 204, and the session goes on (M2 design 3.5).
  const before = await sessionOf(db, second.refresh_token);
  await logout(first.refresh_token);
  expect(await sessionOf(db, second.refresh_token)).toEqual(before);

  // The current one ends the session; its access token fails on the next request.
  await logout(second.refresh_token);
  await expectRevoked(db, second.refresh_token, "logout");
  const me = await api.GET("/api/v0/me", { headers: { Authorization: `Bearer ${second.access_token}` } });
  expect(me.response.status).toBe(401);

  // Once more: the same answer, nothing changes.
  const ended = await sessionOf(db, second.refresh_token);
  await logout(second.refresh_token);
  expect(await sessionOf(db, second.refresh_token)).toEqual(ended);
});
