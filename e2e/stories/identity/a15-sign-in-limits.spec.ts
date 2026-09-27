import { createApi } from "../../fixtures/api";
import { countIdentity, expectNothingAdded } from "../../fixtures/assert/identity";
import { formAlert, submitSignIn } from "../../fixtures/auth-pages";
import { emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// A15, login limits (M2 design 2, 3.10).

/** A nerve with low login limits: login_ip 3, login_ip_email 2, each regaining one unit a minute. */
const lowLoginLimits = {
  NERVE_RATELIMIT__LOGIN_IP__PER_MINUTE: "1",
  NERVE_RATELIMIT__LOGIN_IP__BURST: "3",
  NERVE_RATELIMIT__LOGIN_IP_EMAIL__PER_MINUTE: "1",
  NERVE_RATELIMIT__LOGIN_IP_EMAIL__BURST: "2",
};

test("A15 (page): the sign-in page says when there were too many attempts", async ({
  page,
  db,
  nerveWith,
}, testInfo) => {
  const limited = await nerveWith(lowLoginLimits);
  const email = emailFor(testInfo);
  await register(createApi(limited.baseURL), email);
  const before = await countIdentity(db);
  const attempt = async (address: string, want: number) => {
    expect(await submitSignIn(page, address, "Wr0ng-password"), address).toBe(want);
    await expect(formAlert(page), address).toHaveText(
      want === 429 ? "Too many attempts. Please try again later." : "The email or the password is wrong."
    );
  };

  await page.goto(`${limited.baseURL}/`);
  // One address fails up to login_ip_email's burst, then is refused; another address may still try, until
  // login_ip refuses every address.
  await attempt(email, 401);
  await attempt(email, 401);
  await attempt(email, 429);
  await attempt(emailFor(testInfo, "other"), 401);
  await attempt(emailFor(testInfo, "third"), 429);

  await expectNothingAdded(db, before);
});

test("A15 (API): logins are limited per client IP and address, then per client IP", async ({
  db,
  nerveWith,
}, testInfo) => {
  const limited = createApi((await nerveWith(lowLoginLimits)).baseURL);
  const email = emailFor(testInfo);
  await register(limited, email);
  const before = await countIdentity(db);
  const attempt = async (address: string, want: number) => {
    const { response, error } = await limited.POST("/api/v0/auth/login", {
      body: { email: address, password: "Wr0ng-password" },
    });
    expect(response.status, address).toBe(want);
    if (want === 429) {
      expect(error?.code).toBe("rate_limited");
      expect(response.headers.get("Retry-After")).toBe("60");
    }
  };

  // One address fails up to login_ip_email's burst, then is refused.
  await attempt(email, 401);
  await attempt(email, 401);
  await attempt(email, 429);
  // That refusal took nothing from login_ip: another address gets its last
  // unit, and then login_ip refuses every address.
  await attempt(emailFor(testInfo, "other"), 401);
  await attempt(emailFor(testInfo, "third"), 429);

  await expectNothingAdded(db, before);
});
