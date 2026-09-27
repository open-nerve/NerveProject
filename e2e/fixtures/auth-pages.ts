import { expect, type Page } from "@playwright/test";

// The sign-in and sign-up forms (M2 design 7.3), as a person fills them.

/** The address of the sign-in page that comes back to path after signing in (M2 design 3.18). */
export function signInPath(path: string): string {
  return `/?next_path=${encodeURIComponent(path)}`;
}

/** The message above a form, which says why nerve refused it. */
export function formAlert(page: Page) {
  return page.getByRole("alert");
}

/**
 * Fills the sign-in form, which page shows, and submits it: resolves with the status of nerve's answer to
 * the one login it sends. The address is typed anew, so the message of the last try is gone before this
 * one is sent, even when both have the same values.
 */
export async function submitSignIn(page: Page, email: string, password: string): Promise<number> {
  await page.getByLabel("Email", { exact: true }).clear();
  await page.getByLabel("Email", { exact: true }).fill(email);
  await page.getByLabel("Password", { exact: true }).fill(password);
  await expect(formAlert(page)).toHaveCount(0);
  const [response] = await Promise.all([
    page.waitForResponse((res) => new URL(res.url()).pathname === "/api/v0/auth/login", { timeout: 10_000 }),
    page.getByRole("button", { name: "Go to workspace" }).click(),
  ]);
  return response.status();
}

/** Fills the sign-up form, which page shows, with password typed twice. */
export async function fillSignUp(page: Page, email: string, password: string): Promise<void> {
  await page.getByLabel("Email", { exact: true }).fill(email);
  await page.getByLabel("Set a password", { exact: true }).fill(password);
  await page.getByLabel("Confirm password", { exact: true }).fill(password);
  await expect(formAlert(page)).toHaveCount(0);
}

/** Fills the sign-up form and submits it: resolves with the status of nerve's answer to the registration. */
export async function submitSignUp(page: Page, email: string, password: string): Promise<number> {
  await fillSignUp(page, email, password);
  const [response] = await Promise.all([
    page.waitForResponse((res) => new URL(res.url()).pathname === "/api/v0/auth/register", { timeout: 10_000 }),
    page.getByRole("button", { name: "Create account" }).click(),
  ]);
  return response.status();
}
