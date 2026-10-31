import type { Page } from "@playwright/test";
import { expect } from "@playwright/test";
import { password } from "./auth";
import { signInPath, submitSignIn } from "./auth-pages";
import { watchPage } from "./browser";
import type { ListProject } from "./issue-list-page";
import type { Issue } from "./issues";

/** Opens the actual numbered detail route, observing requests and console from the first navigation. */
export async function openIssueDetail(page: Page, seed: ListProject, issue: Pick<Issue, "sequence_id">) {
  const watch = await watchPage(page);
  const path = `/${seed.slug}/browse/${seed.project.identifier}-${issue.sequence_id}`;
  await page.goto(signInPath(path));
  expect(await submitSignIn(page, seed.member.email, password)).toBe(200);
  await expect(page).toHaveURL(path);
  await expect(page.getByRole("textbox", { name: "Work item title", exact: true })).toBeVisible();
  return watch;
}
