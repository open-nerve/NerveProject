import { expect, type Locator, type Page, type TestInfo } from "@playwright/test";

import {
  addProjectMembers,
  changeProject,
  createProject,
  createWorkspace,
  inviteAndAccept,
  projectMembershipOf,
  projectMemberWrites,
  slugFor,
  type Api,
  type Project,
} from "./api";
import { submitIntakeIssue } from "./intake";
import type { Issue } from "./issues";
import { person } from "./notification-pages";
import { expectPopupBesideButton } from "./settings-pages";

// A project's intake page (M7 P5a spec 2.5–2.8, 2.12), as a person uses it: its address and its list's reads, what a
// story finds on it by role and name, and the world of its stories. What its stories share with the cycle and the
// notification pages' (recordApi, toast, dismissToasts, tabKey, tabStops, failedLoads, seenBy; person, zonedDay,
// zonedMoment) they import from cycle-pages.ts and notification-pages.ts, not copied here; their database reads are
// assert/intake.ts's.

/**
 * The world of the intake page's stories: alice's workspace Acme and its project Web, its intake turned on, of which
 * bob is a member; gus, its guest, who sees his own work items of it alone, as a new project's guests do; and odo, a
 * member of it until a story makes him a guest (demoteToGuest), once he has submitted. Each is a member of Acme as he
 * is of Web, and named: alice "Alice", bob "Bob", gus "Gus", odo "Odo".
 */
export async function intakeWorld(api: Api, testInfo: TestInfo) {
  const [alice, bob, gus, odo] = await Promise.all([
    person(api, testInfo, "alice", "Alice"),
    person(api, testInfo, "bob", "Bob"),
    person(api, testInfo, "gus", "Gus"),
    person(api, testInfo, "odo", "Odo"),
  ]);
  const slug = slugFor(testInfo);
  await createWorkspace(api, alice.token, { name: "Acme", slug });
  await inviteAndAccept(api, alice.token, slug, { email: bob.email, token: bob.token }, 15);
  await inviteAndAccept(api, alice.token, slug, { email: gus.email, token: gus.token }, 5);
  await inviteAndAccept(api, alice.token, slug, { email: odo.email, token: odo.token }, 15);
  const web = await createProject(api, alice.token, slug, { name: "Web", identifier: "WEB" });
  await addProjectMembers(api, alice.token, web.id, [
    { member_id: bob.id, role: 15 },
    { member_id: gus.id, role: 5 },
    { member_id: odo.id, role: 15 },
  ]);
  const turned = await changeProject(api, alice.token, web.id, { intake_view: true });
  expect(turned.status, `Web's intake turned on: ${JSON.stringify(turned)}`).toBe(200);
  return { alice, bob, gus, odo, slug, web };
}

/** Makes the account of memberId a guest of project, as its admin of token. */
export async function demoteToGuest(api: Api, token: string, project: Project, memberId: string): Promise<void> {
  const membership = await projectMembershipOf(api, token, project.id, memberId);
  expect(
    await projectMemberWrites(api, project.id).change(token, membership, 5),
    `${memberId} made a guest of ${project.name}`
  ).toEqual({ status: 200 });
}

/**
 * Submits bodies to the intake of the project of projectId as the caller of token, one after another, and gives the
 * work items in that order: the intake's list, the newest first, lists them the other way round.
 */
export async function submitInOrder(
  api: Api,
  token: string,
  projectId: string,
  bodies: Parameters<typeof submitIntakeIssue>[3][]
): Promise<Issue[]> {
  return bodies.reduce<Promise<Issue[]>>(
    async (previous, body) => [...(await previous), await submitIntakeIssue(api, token, projectId, body)],
    Promise.resolve([])
  );
}

/** The address of the intake page of the project of projectId in the workspace of slug. */
export function intakePath(slug: string, projectId: string): string {
  return `/${slug}/projects/${projectId}/intake`;
}

/**
 * The read of a page of the intake of the project of projectId as recordApi records it, its query as the page sends it
 * (P5a spec 2.2's listQuery: status, priority, assignee_id, created_by_id, label_id, created_at, updated_at, order_by,
 * limit and cursor, in that order, each array one parameter of comma-separated values).
 */
export function intakeListRead(projectId: string, query: string): string {
  return `GET /api/v0/projects/${projectId}/intake-issues?${query}`;
}

/** The reads of an intake's list among requests (recordApi's), in their order. */
export function intakeListReads(requests: readonly string[]): string[] {
  return requests.filter((request) => request.includes("/intake-issues"));
}

/**
 * Checks the intake page's two tabs: the open one named open (with its list's total while it is chosen, P5a R31), and
 * the closed one, never with a total; chosen, the one of chosen.
 */
export async function expectTabs(page: Page, open: string, chosen: "open" | "closed"): Promise<void> {
  const selected = (tab: Locator, yes: boolean) => expect(tab).toHaveAttribute("aria-selected", String(yes));
  await selected(page.getByRole("tab", { name: open, exact: true }), chosen === "open");
  await selected(page.getByRole("tab", { name: "Closed", exact: true }), chosen === "closed");
}

/**
 * The intake's entry of the project of projectId in the app's sidebar: its link, named by the intake and, while it has
 * some, its pending work items as nerve counted them (inbox_issue.a11y.pending_count).
 */
export function intakeEntry(page: Page, slug: string, projectId: string, pending = 0): Locator {
  return page
    .getByRole("complementary", { name: "Main sidebar", exact: true })
    .getByRole("link", { name: pending === 0 ? "Intake" : `Intake, ${pending} pending`, exact: true })
    .and(page.locator(`a[href="${intakePath(slug, projectId)}"]`));
}

/** The rows of the list of the intake page's tab, in its order: each a link named by its work item's identifier and name. */
export function intakeRows(page: Page): Locator {
  return page.getByRole("region", { name: "Intake work items", exact: true }).getByRole("link");
}

/** The names of the rows of the list of the intake page's tab, in its order ("WEB-3 Fix", say). */
export async function shownRows(page: Page): Promise<string[]> {
  return intakeRows(page).evaluateAll((links) => links.map((link) => link.getAttribute("aria-label") ?? ""));
}

/**
 * What the intake page's detail reads of the work item of issueId as it selects it, each read once at least, sorted:
 * the work item, its activity, comments, links, reactions, relations and sub-work items, and its description's versions
 * when the caller may read them, an admin or a member (P5a spec 2.7, 2.12).
 */
export function detailReads(issueId: string, versions = true): string[] {
  const reads = ["", "/activities", "/comments", "/links", "/reactions", "/relations", "/sub-issues"];
  return [...reads, ...(versions ? ["/description-versions"] : [])]
    .map((read) => `GET /api/v0/issues/${issueId}${read}`)
    .toSorted();
}

/** The reads of the work item of issueId among requests (recordApi's), each once, sorted. */
export function readsOf(requests: readonly string[], issueId: string): string[] {
  return [...new Set(requests.filter((request) => request.startsWith(`GET /api/v0/issues/${issueId}`)))].toSorted();
}

/** The intake page's detail: the region around the work item it selects. */
export function intakeDetail(page: Page): Locator {
  return page.getByRole("region", { name: "Intake work item", exact: true });
}

/**
 * The filters' button of the list of the intake page's tab, named Filters by its label: the activity of the work item
 * the page selects has a button that shows the same word.
 */
function filtersButton(page: Page): Locator {
  return page.getByLabel("Filters", { exact: true }).and(page.getByRole("button"));
}

/**
 * Opens the filters' popover of the list of the intake page's tab by a press of its button and gives it, once it shows
 * beside the button (a popover that opens where the page begins, or out of the page, is one a person never sees).
 */
export async function openFilters(page: Page): Promise<Locator> {
  const button = filtersButton(page);
  await button.click();
  await expect(button).toHaveAttribute("aria-expanded", "true");
  const panel = page.locator(`[id="${await button.getAttribute("aria-controls")}"]`);
  await expectPopupBesideButton(panel, button);
  return panel;
}

/**
 * The option named option of the filter titled title in the filters' popover panel: among the buttons that follow the
 * filter's toggle, which is named by the title and, once some are chosen, how many ("Priority (1)").
 */
export function filterOption(panel: Locator, title: string, option: string | RegExp): Locator {
  return panel
    .getByRole("button", { name: new RegExp(`^${title}( \\(\\d+\\))?$`) })
    .locator("xpath=../..")
    .getByRole("button", typeof option === "string" ? { name: option, exact: true } : { name: option });
}

/**
 * Chooses item from the order's menu of the list of the intake page's tab, by the pointer: its button, named by the
 * order the list has, opens it beside itself, and the item chosen closes it.
 */
export async function chooseOrder(page: Page, item: string): Promise<void> {
  const button = page.getByRole("button", { name: /^Order by / });
  await button.click();
  const menu = page.getByRole("menu");
  await expectPopupBesideButton(menu, button);
  await menu.getByRole("menuitem", { name: item, exact: true }).click();
  await expect(menu).toHaveCount(0);
}
