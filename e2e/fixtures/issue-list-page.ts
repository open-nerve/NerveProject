import { expect, type Locator, type Page, type TestInfo } from "@playwright/test";
import type { ProjectPreferences } from "@nerve/api-client";
import {
  changeProjectPreferences,
  createProject,
  createWorkspace,
  readProjectPreferences,
  slugFor,
  type Api,
} from "./api";
import { accountId, emailFor, password } from "./auth";
import { signInPath, submitSignIn } from "./auth-pages";
import { EMOJI_CHECK_WARNING, expectQuietConsole, requestsElsewhere, watchPage, type PageWatch } from "./browser";
import { registerOnboarded } from "./settings-pages";
import { stateIds } from "./states";

/** Independent account and real project for the list page stories; no fabricated page response. */
export async function issueListProject(api: Api, info: TestInfo) {
  const email = emailFor(info, "list-member");
  const tokens = await registerOnboarded(api, email);
  const member = { email, token: tokens.access_token, id: await accountId(api, tokens.access_token) };
  const slug = slugFor(info);
  await createWorkspace(api, member.token, { name: "Lists", slug });
  const project = await createProject(api, member.token, slug, { name: "Web", identifier: "WEB" });
  const states = await stateIds(api, member.token, project.id);
  return { member, slug, project, states, path: `/${slug}/projects/${project.id}/issues` };
}
export type ListProject = Awaited<ReturnType<typeof issueListProject>>;

/** Set full saved preferences through the real endpoint, before loading the page that must honor them. */
export async function saveListPreferences(api: Api, seed: ListProject, patch: Partial<ProjectPreferences>) {
  const preferences = { ...(await readProjectPreferences(api, seed.member.token, seed.project.id)), ...patch };
  expect(await changeProjectPreferences(api, seed.member.token, seed.project.id, preferences)).toEqual({
    status: 200,
    preferences,
  });
  return preferences;
}

/** Sign in through the form, observing from the first navigation and keeping every list query. */
export async function openIssueList(page: Page, seed: ListProject) {
  const watch = await watchPage(page);
  const queries: URLSearchParams[] = [];
  page.on("request", (request) => {
    const url = new URL(request.url());
    if (request.method() === "GET" && url.pathname === `/api/v0/projects/${seed.project.id}/issues`)
      queries.push(url.searchParams);
  });
  await page.goto(signInPath(seed.path));
  expect(await submitSignIn(page, seed.member.email, password)).toBe(200);
  await expect(page).toHaveURL(seed.path);
  return { watch, queries };
}
export function listGroup(page: Page, title: string): Locator {
  return page
    .locator('[class~="group/list-header"]')
    .filter({ has: page.getByText(title, { exact: true }) })
    .locator("../..");
}
/** Outer row identities remain in the DOM when their property contents are outside the virtual viewport. */
export async function expectListOrder(group: Locator, ids: readonly string[]) {
  // A navigation can finish before the application has mounted this group.
  // Wait for that UI boundary before checking the loaded rows and their exact order.
  await group.waitFor({ state: "attached" });
  await expect
    .poll(() => group.locator('[id^="issue_"]').evaluateAll((rows) => rows.map((row) => row.id.split("_")[1])))
    .toEqual(ids);
}
export async function expectListPageQuiet(
  page: Page,
  watch: PageWatch,
  browserName: string,
  loads: number,
  expected: { failures?: string[]; errors?: string[] } = {}
) {
  expect(watch.apiFailures).toEqual(expected.failures ?? []);
  expect(watch.oldApiRequests).toEqual([]);
  expect(watch.pageErrors).toEqual([]);
  expect(watch.cspViolations).toEqual([]);
  expect(requestsElsewhere(page, watch)).toEqual([]);
  await expectQuietConsole(page, watch, {
    warnings: browserName === "chromium" ? Array.from({ length: loads }, () => EMOJI_CHECK_WARNING) : [],
    errors: expected.errors ?? [],
  });
}
