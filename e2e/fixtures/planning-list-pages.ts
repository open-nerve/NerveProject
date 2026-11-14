import { expect, type Browser, type Locator, type Page, type TestInfo } from "@playwright/test";

import {
  addProjectMembers,
  changeRole,
  createProject,
  createWorkspace,
  inviteAndAccept,
  projectMembershipOf,
  projectMemberWrites,
  slugFor,
  type Api,
} from "./api";
import { accountId, bearer, emailFor, login, type AuthTokens } from "./auth";
import { consoleProbe, watchPage, type PageWatch } from "./browser";
import { cyclePath, entriesOf, turnCycles } from "./cycle-pages";
import { createCycle, cycleCalls, type CycleCreate } from "./cycles";
import { createIssue, type Issue, type IssueCreate } from "./issues";
import { expectListPageQuiet } from "./issue-list-page";
import { modulePath, turnModules } from "./module-pages";
import { createModule, moduleCalls } from "./modules";
import { registerOnboarded } from "./settings-pages";
import { stateIds } from "./states";
import { anotherBrowser } from "./workspace-pages";

// A cycle's or a module's page's work items (M6 P5b spec 2.1, 2.6, 2.10; P6b spec 2.2, 2.7, 2.10), as a person uses
// them: the world of their stories, one of each domain over the same people and project, the work items a layout
// shows, a row's menu, the counts the page gives, and the reads a row's dialog makes. The two domains' pages lay out
// their work items alike, so one helper finds them on either. The stories build their world through the API and read
// nerve's own answers.

/** One of the world's people: his address, his id, and the tokens of his sign-in, which his browser takes. */
interface Person {
  email: string;
  id: string;
  tokens: AuthTokens;
}

async function person(api: Api, testInfo: TestInfo, label: string): Promise<Person> {
  const email = emailFor(testInfo, label);
  const tokens = await registerOnboarded(api, email);
  return { email, id: await accountId(api, tokens.access_token), tokens };
}

/**
 * Acme's world of a list page's stories, before its domain is turned on: the admin's workspace Acme and its project
 * Web, and mem, a member of both, whose browser the stories drive. item makes a work item of Web as the admin. guest
 * signs a person of label up who joins Acme and Web as a member, makes his own work item as made says (of the page's
 * cycle or module), and is then Web's guest, whom nerve shows his own work items alone there (Web's guests do not see
 * every work item, M4 design 3.4); Acme's admin as well when workspaceAdmin is true, made so after his guest role, which
 * the promotion keeps (a workspace's admin joins a project as its admin alone).
 */
async function listPageWorld(api: Api, testInfo: TestInfo) {
  const [admin, mem] = [await person(api, testInfo, "admin"), await person(api, testInfo, "mem")];
  const slug = slugFor(testInfo);
  const token = admin.tokens.access_token;
  await createWorkspace(api, token, { name: "Acme", slug });
  await inviteAndAccept(api, token, slug, { email: mem.email, token: mem.tokens.access_token }, 15);
  const web = await createProject(api, token, slug, { name: "Web", identifier: "WEB" });
  await addProjectMembers(api, token, web.id, [{ member_id: mem.id, role: 15 }]);
  const states = await stateIds(api, token, web.id);
  const item = (name: string, body: Partial<IssueCreate> = {}): Promise<Issue> =>
    createIssue(api, token, web.id, { name, ...body });
  const guest = async (label: string, made: IssueCreate, workspaceAdmin: boolean) => {
    const one = await person(api, testInfo, label);
    await inviteAndAccept(api, token, slug, { email: one.email, token: one.tokens.access_token }, 15);
    await addProjectMembers(api, token, web.id, [{ member_id: one.id, role: 15 }]);
    await createIssue(api, one.tokens.access_token, web.id, made);
    const membership = await projectMembershipOf(api, token, web.id, one.id);
    expect(await projectMemberWrites(api, web.id).change(token, membership, 5), `${label}, Web's guest`).toEqual({
      status: 200,
    });
    if (workspaceAdmin) await changeRole(api, token, slug, one.id, 20);
    return one;
  };
  return {
    admin,
    mem,
    slug,
    web,
    token,
    states,
    item,
    guest,
    projectPath: `/${slug}/projects/${web.id}/issues`,
  };
}

/**
 * Acme's world of a cycle page's stories (listPageWorld), Web's cycles on. cycle makes a draft cycle of Web as the
 * admin; end ends a cycle in January 2025 (its work items joined it before), and archives it when asked. guest's own
 * work item, of name, is in the cycle of cycleId; path is the address of a cycle's page.
 */
export async function cycleListWorld(api: Api, testInfo: TestInfo) {
  const world = await listPageWorld(api, testInfo);
  const { token, web } = world;
  await turnCycles(api, token, web, true);
  const cycle = (name: string, body: Partial<CycleCreate> = {}) => createCycle(api, token, web.id, { name, ...body });
  const end = async (cycleId: string, archive = false) => {
    const cycles = cycleCalls(api);
    expect(
      await cycles.update(token, cycleId, { start_date: "2025-01-01", end_date: "2025-01-14" }),
      "the cycle ends in January 2025"
    ).toEqual({ status: 200 });
    if (archive) expect(await cycles.archive(token, cycleId), "the cycle archived").toEqual({ status: 200 });
  };
  return {
    ...world,
    cycle,
    end,
    guest: (label: string, name: string, cycleId: string, workspaceAdmin = false) =>
      world.guest(label, { name, cycle_id: cycleId }, workspaceAdmin),
    path: (cycleId: string) => cyclePath(world.slug, web.id, cycleId),
  };
}

/**
 * Acme's world of a module page's stories (listPageWorld), Web's modules on (a new project's are off). module makes a
 * module of Web, in the backlog, as the admin; archive completes a module and archives it, nerve archiving only a
 * completed or cancelled one (its work items joined it before: nerve refuses that one joins an archived module, not
 * that one stays in it). guest's own work item, of name, is in the module of moduleId; path is the address of a
 * module's page.
 */
export async function moduleListWorld(api: Api, testInfo: TestInfo) {
  const world = await listPageWorld(api, testInfo);
  const { token, web } = world;
  await turnModules(api, token, web, true);
  const module = (name: string) => createModule(api, token, web.id, { name });
  const archive = async (moduleId: string) => {
    const modules = moduleCalls(api);
    expect(
      [await modules.update(token, moduleId, { status: "completed" }), await modules.archive(token, moduleId)],
      "the module completed and archived"
    ).toEqual([{ status: 200 }, { status: 200 }]);
  };
  return {
    ...world,
    module,
    archive,
    guest: (label: string, name: string, moduleId: string, workspaceAdmin = false) =>
      world.guest(label, { name, module_ids: [moduleId] }, workspaceAdmin),
    path: (moduleId: string) => modulePath(world.slug, web.id, moduleId),
  };
}

/** n as two digits, as a day's month and date are written. */
function twoDigits(n: number): string {
  return String(n).padStart(2, "0");
}

/**
 * A weekday of this month, YYYY-MM-DD: a cycle's or a module's page's calendar shows its work items due then (its
 * month, weekends hidden by default) whatever day the story runs.
 */
export function weekdayThisMonth(): string {
  const day = new Date();
  day.setDate(15);
  while (day.getDay() === 0 || day.getDay() === 6) day.setDate(day.getDate() + 1);
  return `${day.getFullYear()}-${twoDigits(day.getMonth() + 1)}-${twoDigits(day.getDate())}`;
}

/**
 * Opens path, a cycle's or a module's page, in the test's page, signed in as who and watched from its first navigation,
 * and closes the page's progress sidebar, which lies over the right of the work items as Plane's does.
 */
export async function openListPage(
  signedInPage: (tokens: AuthTokens) => Promise<Page>,
  who: Person,
  path: string
): Promise<{ page: Page; watch: PageWatch }> {
  const page = await signedInPage(who.tokens);
  const watch = await watchPage(page);
  await page.goto(path);
  await closeSidebar(page);
  return { page, watch };
}

/**
 * Loads path again for who: opens it as openListPage does, in a browser of his own signed in anew, runs check on that
 * page, which then ends quiet (expectListPageQuiet), and closes the browser. The page reads what nerve keeps now by a
 * first load of its document. Not by page.reload(): Firefox gives a document loaded after another in the same browser
 * the fonts that one loaded, and about ten seconds later reports the font the root preloads as unused, whatever the
 * time between the two loads (M6 P5b T16).
 */
export async function loadedAnew(
  browser: Browser,
  baseURL: string,
  api: Api,
  who: Person,
  path: string,
  check: (page: Page, watch: PageWatch) => Promise<void>
): Promise<void> {
  const other = await anotherBrowser(browser, baseURL, await login(api, who.email));
  try {
    const watch = await watchPage(other.page);
    await other.page.goto(path);
    await closeSidebar(other.page);
    await check(other.page, watch);
    await expectListPageQuiet(other.page, watch);
  } finally {
    await other.close();
  }
}

/**
 * expectListPageQuiet for each layout a story walks on one page: each check before has logged its console probe, which
 * this one expects first.
 */
export function quietEachTime(page: Page, watch: PageWatch): () => Promise<void> {
  let checked = 0;
  return async () => {
    const probes = Array<string>(checked).fill(consoleProbe);
    checked += 1;
    await expectListPageQuiet(page, watch, { errors: probes, warnings: probes });
  };
}

/** Closes the cycle's or the module's progress sidebar by the header's toggle, while it is open. */
async function closeSidebar(page: Page): Promise<void> {
  const toggle = page.getByRole("button", { name: "Toggle details", exact: true });
  await expect(toggle).toHaveAttribute("aria-pressed", /true|false/);
  if ((await toggle.getAttribute("aria-pressed")) === "true") await toggle.click();
  await expect(toggle).toHaveAttribute("aria-pressed", "false");
}

/** The button of the menu of the row of the work item of name, in any layout: named by the work item. */
export function rowOptions(page: Page, name: string): Locator {
  return page.getByRole("button", { name: `Options: ${name}`, exact: true });
}

/** The names of the work items whose rows, cards or blocks page shows, sorted: each row has its menu. */
export async function shownItems(page: Page): Promise<string[]> {
  const labels = await page
    .locator('button[aria-label^="Options: "]')
    .evaluateAll((buttons) => buttons.map((button) => button.getAttribute("aria-label") ?? ""));
  return [...new Set(labels.map((label) => label.slice("Options: ".length)))].toSorted();
}

/** The entries of the menu of the row of the work item of name, opened by its button and closed again by Escape. */
export async function rowEntries(page: Page, name: string): Promise<string[]> {
  await rowOptions(page, name).click();
  const entries = await entriesOf(page.getByRole("menu"));
  await page.keyboard.press("Escape");
  await expect(page.getByRole("menu")).toHaveCount(0);
  return entries;
}

/** Chooses entry from the menu of the row of the work item of name. */
export async function chooseFromRow(page: Page, name: string, entry: string): Promise<void> {
  await rowOptions(page, name).click();
  await page.getByRole("menu").getByRole("menuitem", { name: entry, exact: true }).click();
}

/**
 * The button of the header of page that lays its work items out as name says, named as the header names it: "List",
 * "Board", "Calendar" or "Table" (not the layout of a modules list, module-pages.ts's layoutButton).
 */
export function workItemLayoutButton(page: Page, name: "List" | "Board" | "Calendar" | "Table"): Locator {
  return page.getByRole("button", { name: `${name} Layout`, exact: true });
}

/**
 * The count of work items the header of a cycle's or a module's page shows next to the name of its cycle or module,
 * the header's switcher, as nerve gives it.
 */
export function headerCount(page: Page, name: string): Locator {
  return page
    .getByRole("button", { name, exact: true })
    .locator("xpath=ancestor::div[.//*[contains(@class,'rounded-xl')]][1]")
    .locator("div.rounded-xl");
}

/**
 * Waits for the reads a row's edit or copy dialog makes of the work item of id once act opens it: the dialog's own read
 * of the work item, the check of it its sub-issues' read starts (M4 P12 D22), and the read of the list's first page
 * again that check makes, each answered.
 */
export async function dialogReads(page: Page, id: string, projectId: string, act: () => Promise<void>): Promise<void> {
  const reads: string[] = [];
  const record = (response: { url: () => string; request: () => { method: () => string } }) => {
    const { pathname } = new URL(response.url());
    if (response.request().method() !== "GET") return;
    if (pathname === `/api/v0/issues/${id}`) reads.push("work item");
    if (pathname === `/api/v0/issues/${id}/sub-issues`) reads.push("sub-issues");
    if (pathname === `/api/v0/projects/${projectId}/issues`) reads.push("list");
  };
  page.on("response", record);
  try {
    await act();
    // from the dialog's read on: a read of the list the page made before it is not the dialog's
    await expect
      .poll(() => reads.slice(reads.indexOf("work item")))
      .toEqual(["work item", "sub-issues", "work item", "list"]);
  } finally {
    page.off("response", record);
  }
}

/**
 * The queries of the reads of the work items of the project of projectId that page sends from now on, as nerve gets
 * them: a cycle's or a module's page's list is its project's with the cycle or the module (P5b spec 2.1, P6b spec 2.3).
 */
export function listQueries(page: Page, projectId: string): URLSearchParams[] {
  const queries: URLSearchParams[] = [];
  page.on("request", (request) => {
    const url = new URL(request.url());
    if (request.method() === "GET" && url.pathname === `/api/v0/projects/${projectId}/issues`)
      queries.push(url.searchParams);
  });
  return queries;
}

/** The row of the work item of id in a list layout. */
export function listRow(page: Page, id: string): Locator {
  return page.locator(`[id^="issue_${id}_"]`);
}

/** The ids of the rows of the table page shows, in their order: each row's cell of its work item names it. */
export function tableOrder(page: Page): Promise<string[]> {
  return page
    .getByRole("table")
    .locator("tbody > tr")
    .evaluateAll((rows) => rows.flatMap((row) => row.querySelector('td[id^="issue-"]')?.id.slice(6) ?? []));
}

/** Deletes the attachment of id through the API, as the caller of token, who may change its work item. */
export async function deleteAttachment(api: Api, token: string, id: string): Promise<void> {
  const { error, response } = await api.DELETE("/api/v0/assets/{id}", {
    params: { path: { id } },
    headers: bearer(token),
  });
  expect(response.status, `delete the attachment ${id}: ${JSON.stringify(error)}`).toBe(204);
}

/** The list's own button of a new work item (its quick add), not the sidebar's of the workspace. */
export function quickAddButton(page: Page): Locator {
  return page.locator(
    "xpath=//button[normalize-space(.)='New work item'][not(ancestor::aside or ancestor::*[@role='complementary'])]"
  );
}

/**
 * The bodies, as JSON, of the requests of method to path that page sends from now on, in their order: the array grows
 * as they leave. Chromium shows a fetch's body only while the path is routed, so there it is routed through untouched;
 * WebKit and Firefox show it to the request event, and a route there would make WebKit fetch the modules the app had
 * already imported once more and report them unused (M4 P14 review, I2).
 */
export async function bodiesSent(page: Page, browserName: string, method: string, path: string): Promise<unknown[]> {
  if (browserName === "chromium")
    await page.route(
      (url) => url.pathname === path,
      (route) => route.fallback()
    );
  const bodies: unknown[] = [];
  page.on("request", (request) => {
    if (request.method() === method && new URL(request.url()).pathname === path) bodies.push(request.postDataJSON());
  });
  return bodies;
}
