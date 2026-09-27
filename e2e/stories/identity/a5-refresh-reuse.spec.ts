import { randomBytes } from "node:crypto";

import { expectRevoked, sessionOf } from "../../fixtures/assert/identity";
import { signInPath } from "../../fixtures/auth-pages";
import { emailFor, recordOf, refresh, register } from "../../fixtures/auth";
import { expectQuietConsole, watchPage } from "../../fixtures/browser";
import { expect, test } from "../../fixtures/test";

// A5, a reused refresh token (M2 design 2).

/** The error Chromium logs to the console for each request a page sends that nerve answers with 401. */
const refusedResourceError = "Failed to load resource: the server responded with a status of 401 (Unauthorized)";

test("A5 (page): when a copy of the page's refresh token was used, the page's next refresh ends the session", async ({
  api,
  db,
  signedInPage,
}, testInfo) => {
  const page = await signedInPage(await register(api, emailFor(testInfo)));
  const watch = await watchPage(page);
  await page.goto("/onboarding");
  await expect(page.getByText("Create your profile.")).toBeVisible();

  // Someone with a copy of the page's refresh token uses it first: nerve rotates the session to them.
  const held = (await recordOf(page))?.refresh_token ?? "";
  await refresh(api, held);

  // The page's next refresh, as it loads again, presents the retired token: nerve ends the session, and
  // the page goes to the sign-in page, which comes back here.
  await page.reload();
  await expect(page).toHaveURL(signInPath("/onboarding"));
  await expect(page.getByRole("button", { name: "Go to workspace" })).toBeVisible();
  expect(await recordOf(page)).toBeNull();
  await expectRevoked(db, held, "reuse_detected");
  // The one request that failed is that refresh, which Chromium reports in the console as well; nothing
  // else went wrong on the way to the sign-in page.
  expect(watch.apiFailures).toEqual(["401 POST /api/v0/auth/refresh"]);
  expect(watch.pageErrors).toEqual([]);
  await expectQuietConsole(page, watch, { errors: [refusedResourceError] });
});

/** token with its secret and tag replaced by random bytes: the session and the generation are real, the rest is not. */
function forgedFrom(token: string): string {
  const prefix = "nrv_rt_";
  const raw = Buffer.from(token.slice(prefix.length), "base64url");
  randomBytes(48).copy(raw, 20);
  return prefix + raw.toString("base64url");
}

test("A5 (API): a retired refresh token revokes its session; a forged older generation does not", async ({
  api,
  db,
}, testInfo) => {
  const first = await register(api, emailFor(testInfo));
  const second = await refresh(api, first.refresh_token);
  const refused = async (token: string) => {
    const { response, error } = await api.POST("/api/v0/auth/refresh", { body: { refresh_token: token } });
    expect(response.status).toBe(401);
    expect(error?.code).toBe("identity.refresh_token_invalid");
  };

  // A forged older generation proves nothing: 401, the session goes on.
  const before = await sessionOf(db, second.refresh_token);
  await refused(forgedFrom(first.refresh_token));
  expect(await sessionOf(db, second.refresh_token)).toEqual(before);

  // The real retired token comes back: someone else holds a copy.
  await refused(first.refresh_token);
  await expectRevoked(db, first.refresh_token, "reuse_detected");

  // Every token of the session fails from now on.
  await refused(second.refresh_token);
  const me = await api.GET("/api/v0/me", { headers: { Authorization: `Bearer ${second.access_token}` } });
  expect(me.response.status).toBe(401);
});
