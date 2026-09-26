import { countIdentity, expectNothingAdded, expectSignedIn } from "../../fixtures/assert/identity";
import { emailFor, login, password, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// A3, signing in (M2 design 2). The page version, with next_path, joins in M2/P4.

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
