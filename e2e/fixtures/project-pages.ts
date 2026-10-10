import type { Page, Response } from "@playwright/test";

import { answerTo } from "./settings-pages";
import { memberRow } from "./workspace-pages";

// A project's pages (M3 design 7.6), as a person uses them.

/** A request the page sends: its method and its path, as answerTo and holdAnswer take them. */
type Sent = { method: string; path: string };

/** The request that removes the project membership of id. */
export function removalOf(id: string): Sent {
  return { method: "DELETE", path: `/api/v0/project-members/${id}` };
}

/** The request that ends the caller's membership of the project of id. */
export function leavingOf(id: string): Sent {
  return { method: "POST", path: `/api/v0/projects/${id}/leave` };
}

/**
 * Ends a membership from a project's members page, which page shows: opens the menu of the row of email, picks its
 * entry (Leave on the caller's own row, Remove on another's), and confirms it in the dialog that asks. Resolves with
 * nerve's answer to the request the page sends.
 */
export async function endProjectMembership(
  page: Page,
  email: string,
  entry: "Leave" | "Remove",
  request: Sent
): Promise<Response> {
  // the row's menu is its first button: the role's select, when there is one, comes after it
  await memberRow(page, email).locator("button").first().click();
  await page.getByRole("menuitem", { name: entry }).click();
  return answerTo(page, request.method, request.path, () =>
    page.getByRole("dialog").getByRole("button", { name: entry, exact: true }).click()
  );
}
