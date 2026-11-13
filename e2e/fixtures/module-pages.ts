import { expect, type Locator, type Page, type TestInfo } from "@playwright/test";

import { changeProject, type Api, type Project } from "./api";
import type { ModuleRow } from "./assert/planning";
import { cycleMenu, cyclePagesWorld } from "./cycle-pages";

// A project's module pages (M6 P6a spec 2.6, 2.7, 2.8, 2.11), as a person uses them: their addresses, what a story
// finds on them by role and name, the world of their stories and the rows of the database they make. What the module
// pages show as the cycle pages do (a toast, a row's menu, the calendar of a range, the Tab stops of a dialog, the API
// requests a page sends) is found by cycle-pages.ts's helpers. A page story and the API story of its number assert the
// same rows of the database (assert/planning.ts).

/**
 * Acme's world of the module pages' stories: the cycle pages' people and project (cyclePagesWorld), the admin's
 * workspace Acme and its project Web; mem and nia, members of both; gus, a guest of both; and wanda, Acme's admin and
 * Web's guest, whom nerve shows only her own work items there, so she writes none of its modules, as gus does not
 * (P6a spec R5). Web's cycles are off, a new project's; its modules are on unless modulesOn is false.
 */
export async function modulePagesWorld(api: Api, testInfo: TestInfo, modulesOn = true) {
  const world = await cyclePagesWorld(api, testInfo, false);
  if (modulesOn) await turnModules(api, world.token, world.web, true);
  return world;
}

/** Turns the modules of project on or off, as its admin of token. */
export async function turnModules(api: Api, token: string, project: Project, on: boolean): Promise<void> {
  const turned = await changeProject(api, token, project.id, { module_view: on });
  expect(turned.status, `${project.name}'s modules turned ${on ? "on" : "off"}: ${JSON.stringify(turned)}`).toBe(200);
}

/** The address of the modules list of the project of projectId in the workspace of slug. */
export function modulesPath(slug: string, projectId: string): string {
  return `/${slug}/projects/${projectId}/modules`;
}

/** The address of the page of the module of id of that project. */
export function modulePath(slug: string, projectId: string, id: string): string {
  return `${modulesPath(slug, projectId)}/${id}`;
}

/** The address of that project's archive of modules. */
export function archivedModulesPath(slug: string, projectId: string): string {
  return `/${slug}/projects/${projectId}/archives/modules`;
}

/**
 * Goes to the modules list of the project of projectId, which page shows, by its link in the sidebar's project: within
 * the app, as a person does, not by a load of another document (a second load in one page makes Firefox report the font
 * its root preloads as unused, M6 P5b T16; openCyclesList).
 */
export async function openModulesList(page: Page, slug: string, projectId: string): Promise<void> {
  const list = modulesPath(slug, projectId);
  await page
    .getByRole("link", { name: "Modules", exact: true })
    .and(page.locator(`a[href="${list}"]`))
    .first()
    .click();
  await expect(page).toHaveURL(list);
}

/**
 * The ids of the modules whose pages the links within lead to, each once, in the order of the links: a list's rows, an
 * archive's.
 */
export async function linkedModules(within: Locator): Promise<string[]> {
  const hrefs = await within
    .locator('a[href*="/modules/"]')
    .evaluateAll((links) => links.map((link) => link.getAttribute("href") ?? ""));
  return [...new Set(hrefs.map((href) => href.slice(href.lastIndexOf("/") + 1)))];
}

/**
 * The row of the module of id, named name, on page's list or archive: its link, its facts (its days, its status, its
 * lead) and its menu, named by the module as a cycle's is (common.item_options).
 */
export function moduleRow(page: Page, id: string, name: string): Locator {
  return page
    .locator("div.relative")
    .filter({ has: page.locator(`a[href$="/modules/${id}"]`) })
    .filter({ has: cycleMenu(page, name) })
    .last();
}

/**
 * A module's row of the database as a page story makes it (assert/planning.ts's ModuleRow): of a name and a status
 * alone, at its project's first place, made and written last by creator, as to says otherwise.
 */
export function madeModule(name: string, status: string, creator: string, to: Partial<ModuleRow> = {}): ModuleRow {
  return {
    name,
    description: "",
    start: null,
    target: null,
    status,
    lead: null,
    members: [],
    sort_order: 65535,
    archived: false,
    deleted: false,
    creator,
    by: creator,
    ...to,
  };
}

/** rows with the module of name changed as to says. */
export function changedModule(rows: ModuleRow[], name: string, to: Partial<ModuleRow>): ModuleRow[] {
  return rows.map((m) => (m.name === name ? { ...m, ...to } : m));
}

/** What a project's modules list says while the project has none. */
export function noModulesYet(page: Page): Locator {
  return page.getByText("Map your project goals to Modules and track easily.", { exact: true });
}

/**
 * Checks that the list of members open on page offers names alone, each once, after its avatar's initial: in a
 * module's dialog and sidebar, the members of its project who are not its guests (P6a spec R6).
 */
export async function expectMembersOffered(page: Page, ...names: string[]): Promise<void> {
  const options = page.getByRole("option");
  await expect(options).toHaveCount(names.length);
  expect(await Promise.all(names.map((name) => options.filter({ hasText: name }).count()))).toEqual(names.map(() => 1));
}
