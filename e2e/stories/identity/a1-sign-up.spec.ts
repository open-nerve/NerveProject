import { expectRegistered } from "../../fixtures/assert/identity";
import { submitSignUp } from "../../fixtures/auth-pages";
import { emailFor, password, recordOf, register } from "../../fixtures/auth";
import { followAccessToken } from "../../fixtures/browser";
import { expect, test } from "../../fixtures/test";

// A1, a new account (M2 design 2).

test("A1 (page): a visitor signs up and lands on the profile step of onboarding", async ({ page, db }, testInfo) => {
  const email = emailFor(testInfo, "Alice");
  // What the page shows or sends where a token could leak: the console, and the addresses it asks for.
  const shown: string[] = [];
  const sentAccessToken = followAccessToken(page);
  page.on("console", (message) => shown.push(message.text()));
  page.on("request", (request) => shown.push(request.url()));

  await page.goto("/sign-up");
  expect(await submitSignUp(page, email, password)).toBe(201);

  await expect(page).toHaveURL("/onboarding");
  await expect(page.getByText("Create your profile.")).toBeVisible();
  // The session is the token manager's record, and nothing else: no cookie (M2 design 7.1).
  expect(await page.context().cookies()).toEqual([]);
  const record = await recordOf(page);
  expect(Object.keys(record ?? {}).toSorted()).toEqual(["login_id", "refresh_token"]);
  expect(record?.login_id).toMatch(/^[0-9a-f]{32}$/);
  // The tokens are nowhere else: not in sessionStorage, another localStorage key, an address or the
  // console. Their bodies are looked for, so that a copy under another prefix or in quotes counts too.
  const accessToken = sentAccessToken();
  expect(accessToken, "a request carried the access token").not.toBe("");
  const bodies = [record?.refresh_token.replace(/^nrv_rt_/, "") ?? "", accessToken.split(".")[2] ?? ""];
  expect(bodies.map((body) => body.length > 20)).toEqual([true, true]);
  const stored = await page.evaluate(() =>
    [...Object.entries(sessionStorage), ...Object.entries(localStorage).filter(([key]) => key !== "nerve.auth")].map(
      ([key, value]) => `${key}=${value}`
    )
  );
  const leaks = [...stored, page.url(), ...shown].filter((text) => bodies.some((body) => text.includes(body)));
  expect(leaks).toEqual([]);
  const userAgent = await page.evaluate(() => navigator.userAgent);
  await expectRegistered(db, { email, refreshToken: record?.refresh_token ?? "", userAgent, ip: "127.0.0.1" });
});

test("A1 (API): a caller signs up and gets a session", async ({ api, db }, testInfo) => {
  const email = emailFor(testInfo, "Alice");
  const userAgent = "nerve-e2e/A1";

  const tokens = await register(api, email, { "User-Agent": userAgent });

  expect(tokens.token_type).toBe("Bearer");
  expect(tokens.access_token_expires_in).toBe(15 * 60);
  await expectRegistered(db, { email, refreshToken: tokens.refresh_token, userAgent, ip: "127.0.0.1" });

  // The access token works at once.
  const me = await api.GET("/api/v0/me", { headers: { Authorization: `Bearer ${tokens.access_token}` } });
  expect(me.response.status).toBe(200);
  expect(me.data?.email).toBe(email.toLowerCase());
});
