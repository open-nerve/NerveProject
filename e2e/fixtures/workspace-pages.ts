import { expect, type Browser, type Locator, type Page, type Response } from "@playwright/test";

import { password, signInContext, type AuthTokens } from "./auth";
import { submitSignIn } from "./auth-pages";
import { watchPage, type PageWatch } from "./browser";
import { answerTo, sentTo } from "./settings-pages";

// The workspace's pages (M3 design 7.5), as a person uses them.

/**
 * A page of a browser of its own, signed in with tokens at the nerve of baseURL: another person's, beside the test's
 * page. Its context closes when the test's function returned by it is called.
 */
export async function anotherBrowser(
  browser: Browser,
  baseURL: string,
  tokens: AuthTokens
): Promise<{ page: Page; close: () => Promise<void> }> {
  const context = await browser.newContext({ baseURL });
  await signInContext(context, baseURL, tokens);
  return { page: await context.newPage(), close: () => context.close() };
}

/**
 * Signs the account of email in through the sign-in form, in a browser of its own at the nerve of baseURL, watched
 * from its first page: resolves once nerve has let it in. Its context closes when the function returned is called.
 */
export async function signInAnew(
  browser: Browser,
  baseURL: string,
  email: string
): Promise<{ page: Page; watch: PageWatch; close: () => Promise<void> }> {
  const context = await browser.newContext({ baseURL });
  const page = await context.newPage();
  const watch = await watchPage(page);
  await page.goto("/");
  expect(await submitSignIn(page, email, password), `the sign-in of ${email}`).toBe(200);
  return { page, watch, close: () => context.close() };
}

/**
 * Opens the workspace of id through the workspace menu of page, which shows a workspace: resolves with what the page
 * sent as the workspace opened last, and nerve's answer.
 */
export async function switchWorkspace(page: Page, id: string): Promise<{ body: unknown; answer: Response }> {
  await page.getByRole("button", { name: "Open workspace switcher" }).click();
  return sentTo(page, "PATCH", "/api/v0/me/profile", () => page.locator(`a[id="${id}"]`).click());
}

/**
 * Confirms the deletion of the workspace whose name is name on its general page, which page shows: the admin opens the
 * deletion, types the name and the words that confirm it, and confirms. Resolves once the confirmation is clicked, not
 * waiting for nerve's answer.
 */
export async function confirmDeletion(page: Page, name: string): Promise<void> {
  await page.getByRole("button", { name: "Delete", exact: true }).click();
  await page.locator("#workspaceName").fill(name);
  await page.locator("#confirmDelete").fill("delete my workspace");
  await page.getByRole("button", { name: "Confirm", exact: true }).click();
}

/**
 * Deletes the workspace of slug, whose name is name, from its general page, which page shows (confirmDeletion).
 * Resolves with nerve's answer to the deletion.
 */
export function deleteFromGeneralPage(page: Page, slug: string, name: string): Promise<Response> {
  return answerTo(page, "DELETE", `/api/v0/workspaces/${slug}`, () => confirmDeletion(page, name));
}

/** The row of the members page, which page shows, of the member whose address is email. */
export function memberRow(page: Page, email: string): Locator {
  return page.locator("tr", { hasText: email });
}

/**
 * Picks the role named to in the role select of the row of email on the members page, which page shows and where the
 * select shows the role named from: resolves with what the page sent to the membership of id, and nerve's answer.
 */
export async function pickRole(
  page: Page,
  email: string,
  id: string,
  role: { from: string; to: string }
): Promise<{ body: unknown; answer: Response }> {
  await memberRow(page, email).getByRole("button", { name: role.from, exact: true }).click();
  return sentTo(page, "PATCH", `/api/v0/workspace-members/${id}`, () =>
    page.getByRole("option", { name: role.to, exact: true }).click()
  );
}

/**
 * Ends a membership from the members page, which page shows: opens the menu of the row of email, picks its entry
 * (Leave on the caller's own row, Remove on another's), and confirms it in the dialog that asks. Resolves with nerve's
 * answer to the request of method to path that the page sends.
 */
export async function endMembership(
  page: Page,
  email: string,
  entry: "Leave" | "Remove",
  request: { method: string; path: string }
): Promise<Response> {
  // the row's menu is its first button: the role's select, when there is one, comes after it
  await memberRow(page, email).locator("button").first().click();
  await page.getByRole("button", { name: entry, exact: true }).click();
  return answerTo(page, request.method, request.path, () =>
    page.getByRole("dialog").getByRole("button", { name: entry, exact: true }).click()
  );
}
