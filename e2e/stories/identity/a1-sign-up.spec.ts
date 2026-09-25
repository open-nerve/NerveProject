import { expectRegistered } from "../../fixtures/assert/identity";
import { emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// A1, a new account (M2 design 2). The page version joins in M2/P4.

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
