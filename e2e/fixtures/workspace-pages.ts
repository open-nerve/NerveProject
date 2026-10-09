import type { Browser, Page, Response } from "@playwright/test";

import { signInContext, type AuthTokens } from "./auth";
import { answerTo } from "./settings-pages";

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
