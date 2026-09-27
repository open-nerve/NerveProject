import type { Page } from "@playwright/test";

import { accountOf, onboardingStepsOf } from "../../fixtures/assert/identity";
import { submitSignUp } from "../../fixtures/auth-pages";
import { bearer, createPAT, emailFor, password, register } from "../../fixtures/auth";
import { expectQuietConsole, watchPage, type PageWatch } from "../../fixtures/browser";
import type { Database } from "../../fixtures/db";
import { saveProfileStep } from "../../fixtures/onboarding-pages";
import { expect, test } from "../../fixtures/test";

// A10, the profile step of onboarding (M2 design 2).

/**
 * Takes the profile step that page shows the new account of email, a lowercased address: the step saves
 * the name, and nothing went wrong on the page since watch began.
 */
async function takeProfileStep(page: Page, watch: PageWatch, db: Database, email: string): Promise<void> {
  await expect(page.getByText("Create your profile.")).toBeVisible();
  await page.getByLabel("Name").fill("Ada");
  expect(await saveProfileStep(page)).toBe(200);

  // The next step shows; nerve has the name and the one step done.
  await expect(page.getByText("Create your workspace")).toBeVisible();
  expect(await onboardingStepsOf(db, email)).toEqual({
    profile_complete: true,
    workspace_create: false,
    workspace_invite: false,
    workspace_join: false,
  });
  expect((await accountOf(db, email)).first_name).toBe("Ada");
  // Nothing went wrong on the way (M2 design 3.1): no API call failed, none went to an older API, such as
  // M3's workspaces and invitations, no exception or rejection was left unhandled, the CSP blocked
  // nothing, the console has no error or warning.
  expect(watch.apiFailures).toEqual([]);
  expect(watch.oldApiRequests).toEqual([]);
  expect(watch.pageErrors).toEqual([]);
  expect(watch.cspViolations).toEqual([]);
  await expectQuietConsole(page, watch);
}

test("A10 (page): a new account's first visit of /onboarding goes well, and its profile step saves the name", async ({
  api,
  db,
  signedInPage,
}, testInfo) => {
  const email = emailFor(testInfo);
  const page = await signedInPage(await register(api, email));
  const watch = await watchPage(page);

  // A full load: /onboarding mounts before the app has the account.
  await page.goto("/onboarding");
  await takeProfileStep(page, watch, db, email);
});

test("A10 (page): signing up goes on to /onboarding within the app, which asks no older API", async ({
  page,
  db,
}, testInfo) => {
  const email = emailFor(testInfo);
  const watch = await watchPage(page);

  await page.goto("/sign-up");
  expect(await submitSignUp(page, email, password)).toBe(201);
  // The app goes on by itself once it has the new account, so /onboarding mounts with the account there:
  // the document is still the one loaded for /sign-up.
  await expect(page).toHaveURL("/onboarding");
  expect(await page.evaluate(() => performance.getEntriesByType("navigation").map((entry) => entry.name))).toEqual([
    new URL("/sign-up", page.url()).href,
  ]);
  await takeProfileStep(page, watch, db, email);
});

test("A10 (API): the profile step sets the name and one step, which the others keep beside", async ({
  api,
  db,
}, testInfo) => {
  const email = emailFor(testInfo);
  const pat = await createPAT(api, (await register(api, email)).access_token);

  const named = await api.PATCH("/api/v0/me", { body: { first_name: "Ada" }, headers: bearer(pat.token) });
  expect(named.response.status).toBe(200);
  expect((await accountOf(db, email)).first_name).toBe("Ada");

  // Another step is done already, so keeping it differs from resetting it
  // to its default.
  const joined = await api.PATCH("/api/v0/me/profile", {
    body: { onboarding_step: { workspace_join: true } },
    headers: bearer(pat.token),
  });
  expect(joined.response.status).toBe(200);

  // One key: it is merged in, the other three keep their values (M2 design 3.14).
  const stepped = await api.PATCH("/api/v0/me/profile", {
    body: { onboarding_step: { profile_complete: true } },
    headers: bearer(pat.token),
  });
  expect(stepped.response.status).toBe(200);
  const merged = { profile_complete: true, workspace_create: false, workspace_invite: false, workspace_join: true };
  expect(await onboardingStepsOf(db, email)).toEqual(merged);

  // An unknown key, here misspelt, breaks the contract: the platform's 400,
  // and nothing changes.
  const unknown = await api.PATCH("/api/v0/me/profile", {
    body: { onboarding_step: { profile_completed: true } },
    headers: bearer(pat.token),
  });
  expect(unknown.response.status).toBe(400);
  expect(unknown.error?.errors?.map((e) => ({ field: e.field, code: e.code }))).toEqual([
    { field: "onboarding_step.profile_completed", code: "not_allowed" },
  ]);
  expect(await onboardingStepsOf(db, email)).toEqual(merged);
});
