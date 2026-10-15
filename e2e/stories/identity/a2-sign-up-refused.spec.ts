import type { Page } from "@playwright/test";

import { createApi, type Api } from "../../fixtures/api";
import { countIdentity, expectNothingAdded } from "../../fixtures/assert/identity";
import { fillSignUp, formAlert, submitSignUp } from "../../fixtures/auth-pages";
import { emailFor, password, register } from "../../fixtures/auth";
import { recordFocuses, watchPage } from "../../fixtures/browser";
import { expect, test } from "../../fixtures/test";

// A2, sign-up refused (M2 design 2).

test("A2 (page): a refused sign-up says why in place, keeps the address and adds nothing", async ({
  page,
  api,
  db,
  nerve,
  nerveWith,
}, testInfo) => {
  const email = emailFor(testInfo);
  await register(api, email);
  const before = await countIdentity(db);
  const newEmail = emailFor(testInfo, "new");
  const emailInput = page.getByLabel("Email", { exact: true });
  const watch = await watchPage(page);

  await page.goto("/sign-up");
  // An address the page's rule refuses, confirmed by Enter (nothing goes, the form not filled in): not read invalid while
  // the field has the focus, its message shown once the field is left, and read invalid then, the message its
  // description. An address only an email input's own rule refuses (a quoted name) is not read invalid at all: the
  // page's rule is the field's only one.
  await emailInput.fill("abc");
  await emailInput.press("Enter");
  await expect(emailInput).not.toHaveAttribute("aria-invalid");
  await emailInput.press("Tab");
  await expect(emailInput).toHaveAttribute("aria-invalid", "true");
  await expect(emailInput).toHaveAccessibleDescription("Email is invalid");
  await emailInput.fill('"ab"@example.com');
  await emailInput.press("Enter");
  await emailInput.press("Tab");
  await expect(emailInput).not.toHaveAttribute("aria-invalid");
  // The spaces around an address are dropped, as an email input drops them.
  await emailInput.fill(" abc@example.com ");
  await expect(emailInput).toHaveValue("abc@example.com");

  // The address is taken: the message is above the form, the page and the address stay.
  expect(await submitSignUp(page, email.toUpperCase(), password)).toBe(409);
  await expect(formAlert(page)).toHaveText("An account with this email already exists.");
  await expect(page).toHaveURL("/sign-up");
  await expect(emailInput).toHaveValue(email.toUpperCase());

  // A weak password: the rules show under the field, and the page sends nothing (counted at the end).
  await fillSignUp(page, newEmail, "password");
  const upperCaseRule = page.getByText("Min 1 upper-case letter", { exact: true });
  await expect(page.getByText("8–128 characters", { exact: true })).toBeVisible();
  await expect(upperCaseRule).toBeVisible();
  await page.getByRole("button", { name: "Create account", exact: true }).click();
  await expect(formAlert(page)).toHaveText("Try setting-up a strong password to proceed");
  // A valid password: the rules go.
  await page.getByLabel("Set a password", { exact: true }).fill("N3wPassw0rd!");
  await expect(upperCaseRule).toHaveCount(0);

  // Common passwords, which only nerve knows: the message is under the field, none above the form. The field takes the
  // focus, already read invalid, its message its description (as recorded when it takes it).
  const expectTooCommon = async (common: string) => {
    const focuses = await recordFocuses(page);
    expect(await submitSignUp(page, newEmail, common), common).toBe(422);
    await expect(page.getByText("This password is too common")).toBeVisible();
    await expect(formAlert(page)).toHaveCount(0);
    await expect(emailInput).toHaveValue(newEmail);
    await expect(page.locator("#password")).toBeFocused();
    expect((await focuses()).at(-1)).toMatchObject({
      id: "password",
      invalid: "true",
      description: expect.stringMatching(/^This password is too common/),
    });
  };
  await expectTooCommon("Password1!");
  await expectTooCommon("Password1!~");

  // With sign-up off the header has no sign-up link, which it has with sign-up on; and nerve refuses
  // every address alike, a taken one too.
  const signUpLink = page.getByRole("link", { name: "Sign up", exact: true });
  await showSignIn(page, nerve.baseURL);
  await expect(signUpLink).toBeVisible();
  const closed = await nerveWith({ NERVE_AUTH__SIGNUP_ENABLED: "false" });
  await showSignIn(page, closed.baseURL);
  await expect(signUpLink).toHaveCount(0);
  await page.goto(`${closed.baseURL}/sign-up`);
  const expectClosed = async (address: string) => {
    expect(await submitSignUp(page, address, password), address).toBe(403);
    await expect(formAlert(page)).toHaveText("Sign-up is closed.");
  };
  await expectClosed(newEmail);
  await expectClosed(email);

  // One registration went to nerve per submission above but the weak password's: the taken address, the
  // two common passwords, the two addresses on the closed nerve. The CSP blocked nothing on either nerve.
  expect(watch.apiRequests.filter((request) => request === "POST /api/v0/auth/register")).toHaveLength(5);
  expect(watch.cspViolations).toEqual([]);
  await expectNothingAdded(db, before);
});

/**
 * Opens the sign-in page of the nerve at baseURL, and waits until it has the instance's settings and shows its form,
 * which the app renders only once it holds them (InstanceWrapper): a check made after this that the page lacks
 * something sees the page those settings decide, not one still waiting for them.
 */
async function showSignIn(page: Page, baseURL: string): Promise<void> {
  await Promise.all([
    page.waitForResponse((res) => res.url() === `${baseURL}/api/v0/instance` && res.ok(), { timeout: 10_000 }),
    page.goto(`${baseURL}/`),
  ]);
  await expect(page.getByRole("button", { name: "Go to workspace" })).toBeVisible();
}

async function expectRefused(
  api: Api,
  body: { email: string; password: string },
  want: { status: number; code: string; fields?: { field: string; code: string }[] }
): Promise<void> {
  const { response, error } = await api.POST("/api/v0/auth/register", { body });
  const label = `${body.email} ${body.password}`;
  expect(response.status, label).toBe(want.status);
  expect(error?.code, label).toBe(want.code);
  if (want.fields) {
    expect(
      error?.errors?.map((e) => ({ field: e.field, code: e.code })),
      label
    ).toEqual(want.fields);
  }
}

test("A2 (API): a refused sign-up answers why and adds nothing", async ({ api, db, nerveWith }, testInfo) => {
  const email = emailFor(testInfo);
  await register(api, email);
  const before = await countIdentity(db);
  const newEmail = emailFor(testInfo, "new");
  const invalid = (pw: string, code: string) =>
    expectRefused(
      api,
      { email: newEmail, password: pw },
      { status: 422, code: "validation_failed", fields: [{ field: "password", code }] }
    );

  await Promise.all([
    // The address is taken, whatever its case.
    expectRefused(api, { email: email.toUpperCase(), password }, { status: 409, code: "identity.email_taken" }),
    // A weak password, and common ones.
    invalid("password", "weak_password"),
    invalid("Password1!", "common_password"),
    invalid("Password1!~", "common_password"),
  ]);

  // With sign-up off, every address gets the same answer, a taken one too.
  const closed = createApi((await nerveWith({ NERVE_AUTH__SIGNUP_ENABLED: "false" })).baseURL);
  await Promise.all(
    [newEmail, email].map((address) =>
      expectRefused(closed, { email: address, password }, { status: 403, code: "identity.signup_disabled" })
    )
  );

  await expectNothingAdded(db, before);
});
