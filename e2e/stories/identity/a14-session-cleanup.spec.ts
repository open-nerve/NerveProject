import { sessionOf } from "../../fixtures/assert/identity";
import { emailFor, login, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// A14, expired sessions are cleaned up (M2 design 2, 3.15): River's periodic
// job runs every auth.session_cleanup_interval, 2 s in the test
// configuration. There is no page version.

test("A14: the cleanup job deletes an expired session and keeps the live one", async ({ api, db }, testInfo) => {
  const email = emailFor(testInfo);
  const expired = await register(api, email);
  const live = await login(api, email);
  const { id } = await sessionOf(db, expired.refresh_token);
  await db.query("UPDATE auth_sessions SET expires_at = now() - interval '1 minute' WHERE id = $1", [id]);

  await expect
    .poll(async () => (await db.query("SELECT id FROM auth_sessions WHERE id = $1", [id])).length, {
      message: "the expired session is deleted",
      timeout: 15_000,
    })
    .toBe(0);
  expect((await sessionOf(db, live.refresh_token)).revoked_at).toBeNull();
});
