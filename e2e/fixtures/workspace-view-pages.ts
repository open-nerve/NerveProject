/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { components } from "@nerve/api-client";
import { expect, type Locator, type Page, type TestInfo } from "@playwright/test";

import {
  addProjectMembers,
  archiveProject,
  changeProject,
  changeRole,
  createLabel,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
  type Api,
} from "./api";
import { focusedName } from "./intake-focus";
import { demoteToGuest } from "./intake-pages";
import { createIssue, issueSubscribed } from "./issues";
import { WORKSPACE_LABELS, WORKSPACE_VIEWS, valued } from "./mounts";
import { person } from "./notification-pages";
import { pageMain, rowButton } from "./view-pages";
import { createWorkspaceView, viewDefaults } from "./views";

// A workspace's views pages (M7/P6b spec 2.5, 2.7–2.10, 2.15), as a person uses them: their addresses and reads, what a
// story finds on them by role and name, the controls Update view stops, and the world of their stories. What they share
// with the project's views pages (pageMain, rowButton), the cycle, the notification and the settings pages' stories
// (recordApi, toast, failedLoads, seenBy; person; holdAnswer, sentHeld) they import from there, not copied here; the
// views' calls and rows are views.ts's.

type ViewCreate = components["schemas"]["ViewCreate"];

/**
 * The world of the workspace's views pages' stories: alice's workspace Acme, of which bob is a member and gus a guest;
 * each named, alice "Alice", bob "Bob", gus "Gus". Its projects: Web, bob's and gus's, whose guests see their own work
 * items alone; Ops, bob's and gus's, whose guests see every work item; Lab, alice's alone; Shelf, bob's, archived.
 * Web's label Bug and Ops' Infra, and Lab's Hidden. Their work items, alice's but where said: in Web, Crash, urgent,
 * Bug, bob's to do; Typo, bob's own, low; Note, gus's own, made while he was a member (a guest makes none); in Ops,
 * Deploy, Infra, to which bob subscribes; Lab's Secret, and Shelf's Shelved, bob's to do: bob's list shows Crash,
 * Deploy, Note and Typo, gus's Deploy and Note.
 */
export async function workspaceViewWorld(api: Api, testInfo: TestInfo) {
  const [alice, bob, gus] = await Promise.all([
    person(api, testInfo, "alice", "Alice"),
    person(api, testInfo, "bob", "Bob"),
    person(api, testInfo, "gus", "Gus"),
  ]);
  const slug = slugFor(testInfo);
  const acme = await createWorkspace(api, alice.token, { name: "Acme", slug });
  await inviteAndAccept(api, alice.token, slug, { email: bob.email, token: bob.token }, 15);
  await inviteAndAccept(api, alice.token, slug, { email: gus.email, token: gus.token }, 15);
  const [web, ops, lab, shelf] = await Promise.all(
    (
      [
        ["Web", "WEB"],
        ["Ops", "OPS"],
        ["Lab", "LAB"],
        ["Shelf", "SHF"],
      ] as const
    ).map(([name, identifier]) => createProject(api, alice.token, slug, { name, identifier }))
  );
  if (!web || !ops || !lab || !shelf) throw new Error("missing projects");
  const bobAndGus = [
    { member_id: bob.id, role: 15 as const },
    { member_id: gus.id, role: 15 as const },
  ];
  await Promise.all([
    addProjectMembers(api, alice.token, web.id, bobAndGus),
    addProjectMembers(api, alice.token, ops.id, bobAndGus),
    addProjectMembers(api, alice.token, shelf.id, [{ member_id: bob.id, role: 15 }]),
    changeProject(api, alice.token, ops.id, { guest_view_all_features: true }),
  ]);
  const [bug, infra] = await Promise.all([
    createLabel(api, alice.token, web.id, { name: "Bug" }),
    createLabel(api, alice.token, ops.id, { name: "Infra" }),
    createLabel(api, alice.token, lab.id, { name: "Hidden" }),
  ]);
  if (!bug || !infra) throw new Error("missing labels");
  const [crash, typo, note, deploy] = await Promise.all([
    createIssue(api, alice.token, web.id, {
      name: "Crash",
      priority: "urgent",
      label_ids: [bug.id],
      assignee_ids: [bob.id],
    }),
    createIssue(api, bob.token, web.id, { name: "Typo", priority: "low" }),
    createIssue(api, gus.token, web.id, { name: "Note" }),
    createIssue(api, alice.token, ops.id, { name: "Deploy", label_ids: [infra.id] }),
    createIssue(api, alice.token, lab.id, { name: "Secret" }),
    createIssue(api, alice.token, shelf.id, { name: "Shelved", assignee_ids: [bob.id] }),
  ]);
  if (!crash || !typo || !note || !deploy) throw new Error("missing work items");
  expect(await issueSubscribed(api, bob.token, deploy.id), "bob subscribes to Deploy").toEqual({ status: 204 });
  // gus, who made Note as a member, then the guest of Web, of Ops and of Acme
  await Promise.all([demoteToGuest(api, alice.token, web, gus.id), demoteToGuest(api, alice.token, ops, gus.id)]);
  await changeRole(api, alice.token, slug, gus.id, 5);
  await archiveProject(api, alice.token, shelf.id);
  return { alice, bob, gus, slug, acme, web, ops, lab, bug, infra, crash, typo, note, deploy };
}

/**
 * The smaller world of the stories of the workspace's views list and of its views' pages' keys and late answers (H14,
 * H20): alice's workspace Acme, of which bob is a member and gus a guest; each named. Its project Web, alice's and
 * bob's, holds Crash, urgent, and Typo, low. view makes a view of Acme's own as the one of token, in the spreadsheet
 * as the web app makes one (workspaceViewContent) unless body says otherwise.
 */
export async function workspaceViewsWorld(api: Api, testInfo: TestInfo) {
  const [alice, bob, gus] = await Promise.all([
    person(api, testInfo, "alice", "Alice"),
    person(api, testInfo, "bob", "Bob"),
    person(api, testInfo, "gus", "Gus"),
  ]);
  const slug = slugFor(testInfo);
  const acme = await createWorkspace(api, alice.token, { name: "Acme", slug });
  await inviteAndAccept(api, alice.token, slug, { email: bob.email, token: bob.token }, 15);
  await inviteAndAccept(api, alice.token, slug, { email: gus.email, token: gus.token }, 5);
  const web = await createProject(api, alice.token, slug, { name: "Web", identifier: "WEB" });
  await addProjectMembers(api, alice.token, web.id, [{ member_id: bob.id, role: 15 }]);
  await Promise.all([
    createIssue(api, alice.token, web.id, { name: "Crash", priority: "urgent" }),
    createIssue(api, alice.token, web.id, { name: "Typo", priority: "low" }),
  ]);
  return {
    alice,
    bob,
    gus,
    slug,
    acme,
    web,
    view: (who: { token: string }, body: ViewCreate, of = slug) =>
      createWorkspaceView(api, who.token, of, { ...workspaceViewContent, ...body }),
  };
}

/**
 * Records whether page's document shows text at any moment from now on, even for one the page takes away at once (a
 * view's page that would say its view does not exist on its way elsewhere), where a look made later would miss it:
 * the function returned tells.
 */
export async function recordShown(page: Page, text: string): Promise<() => Promise<boolean>> {
  const record = await page.evaluateHandle((wanted) => {
    const seen = { shown: document.body.textContent?.includes(wanted) ?? false };
    new MutationObserver((mutations) => {
      const shown = mutations.some((mutation) =>
        mutation.type === "characterData"
          ? (mutation.target.textContent?.includes(wanted) ?? false)
          : Array.from(mutation.addedNodes).some((node) => node.textContent?.includes(wanted))
      );
      if (shown) seen.shown = true;
    }).observe(document.body, { subtree: true, childList: true, characterData: true });
    return seen;
  }, text);
  return async () => (await record.jsonValue()).shown;
}

/**
 * Waits for the reads page has out to be answered (recordApi's), as a story does before it moves or changes nerve: all
 * of them, or all but held, those whose answers the story holds.
 */
export async function settled(page: Page, recorded: () => { pending: number }, held = 0): Promise<void> {
  await expect.poll(() => recorded().pending, "the reads out answered").toBe(held);
}

/** The address of the workspace's views list of slug. */
export function workspaceViewsPath(slug: string): string {
  return `/${slug}/workspace-views`;
}

/** The address of the page of a view of the workspace of slug: a default view's key, or a view's id. */
export function workspaceViewPath(slug: string, id: string): string {
  return `${workspaceViewsPath(slug)}/${id}`;
}

/** The read of the workspace's label catalog of slug, as recordApi records it (WORKSPACE_LABELS). */
export function catalogRead(slug: string): string {
  const [read = ""] = WORKSPACE_LABELS;
  return valued(read, { [slug]: "{slug}" });
}

/** The read of the workspace's own views of slug, as recordApi records it (WORKSPACE_VIEWS). */
export function workspaceViewsRead(slug: string): string {
  const [read = ""] = WORKSPACE_VIEWS;
  return valued(read, { [slug]: "{slug}" });
}

/** Of requests (recordApi's), the reads of the work items of the workspace of slug, by their queries. */
export function workspaceListReads(requests: readonly string[], slug: string): URLSearchParams[] {
  const read = `GET /api/v0/workspaces/${slug}/issues?`;
  return requests
    .filter((request) => request.startsWith(read))
    .map((request) => new URLSearchParams(request.slice(read.length)));
}

/**
 * Of requests (recordApi's), all but the reads of the projects of projectIds, which it checks are made at most once
 * each: a row whose project the project store does not hold yet as the list answers (the workspace's projects not
 * answered then) has its project read on its own (issue-list-read.ts), as the two reads' race decides.
 */
export function withoutRowProjects(requests: readonly string[], projectIds: readonly string[]): string[] {
  const reads = new Set(projectIds.map((id) => `GET /api/v0/projects/${id}`));
  const projects = requests.filter((request) => reads.has(request));
  expect(projects, "each row's project read at most once").toEqual([...new Set(projects)]);
  return requests.filter((request) => !reads.has(request));
}

/**
 * The whole query of a default view's page's first read, its display the browser's default (M7/P6b spec 2.4, 2.5):
 * the newest first, sub-issues shown, a page of a hundred, its own term last; no grouping, no other word.
 */
export function defaultQuery(own: Record<string, string> = {}, order = "-created_at"): Record<string, string> {
  return { order_by: order, sub_issue: "true", ...own, limit: "100" };
}

/**
 * What a view the web app makes of the workspace's own holds besides its words (WORKSPACE_VIEW_CONTENT; M7/P6b spec
 * 2.11, R8): the display nerve gives a view made without one, in the spreadsheet's layout; every property shown.
 */
export const workspaceViewContent = {
  display_filters: { ...viewDefaults.display_filters, layout: "spreadsheet" },
  display_properties: viewDefaults.display_properties,
} satisfies Pick<ViewCreate, "display_filters" | "display_properties">;

/**
 * The properties a workspace view's page's Display shows and hides, in its order (M7/P6b spec 2.4, re-97 m1): the
 * table's, without Cycle or Modules, whose columns a page of no project never shows.
 */
export const TABLE_PROPERTIES = [
  "ID",
  "Assignee",
  "Start date",
  "Due date",
  "Labels",
  "Priority",
  "State",
  "Sub-work item count",
  "Link",
  "Attachments",
  "Created on",
  "Updated on",
];

/** Notes where page's focus is (focusedName, its first 60 characters) under step: each engine's, read in its report. */
export async function noteFocus(page: Page, testInfo: TestInfo, step: string): Promise<void> {
  testInfo.annotations.push({ type: "focus", description: `${step}: ${(await focusedName(page)).slice(0, 60)}` });
}

/** The names of the options the list open on page offers, in its order: a condition's values, the switcher's views. */
export async function offered(page: Page): Promise<string[]> {
  const options = page.getByRole("listbox").getByRole("option");
  await expect(options.first()).toBeVisible();
  return (await options.allTextContents()).map((text) => text.trim());
}

/** The button of the header's switcher, named by the view the page shows. */
export function switcher(page: Page, name: string): Locator {
  return pageMain(page).getByRole("button", { name, exact: true });
}

/** The page's Display button, in its header. */
function displayButton(page: Page): Locator {
  return pageMain(page).getByRole("button", { name: "Display", exact: true });
}

/** The buttons of the table's column headers, each of which orders the table by its column. */
function columnButtons(page: Page): Locator {
  return page.getByRole("table").getByRole("columnheader").getByRole("button");
}

/** The header of the table's column of name: "Labels", "Priority"… */
export function columnHeader(page: Page, name: string): Locator {
  return page.getByRole("table").getByRole("columnheader", { name, exact: true });
}

/** Orders the table by its column of name, as entry ("A Z", "Urgent None"…) of its header's menu says. */
export async function orderBy(page: Page, column: string, entry: string): Promise<void> {
  await columnHeader(page, column).getByRole("button").click();
  await page.getByRole("menuitem", { name: entry, exact: true }).click();
}

/**
 * What the labels cell of each of the table's rows shows, by its work item's name (its button, "Labels: {name}"): the
 * names of its labels, or "Select labels" for none.
 */
export async function rowLabels(page: Page): Promise<Record<string, string>> {
  const shown = await page
    .getByRole("table")
    .locator('button[aria-label^="Labels: "]')
    .evaluateAll((buttons) =>
      buttons.map((button) => [button.getAttribute("aria-label") ?? "", button.textContent?.trim() ?? ""] as const)
    );
  return Object.fromEntries(shown.map(([label, text]) => [label.slice("Labels: ".length), text]));
}

/**
 * Checks the ways a workspace view's page changes its draft (M7/P6b spec 2.9, as P6a R29): while Update view is out
 * (enabled false), each of the conditions without its remove button, no Add filter anywhere (the row's, or the
 * header's filters toggle's), Clear all and Save view stopped, the display and every column's order stopped; once
 * nerve has answered, each working again, the row's Add filter back.
 */
export async function expectPageControls(page: Page, enabled: boolean, conditions: number): Promise<void> {
  const state = enabled ? "toBeEnabled" : "toBeDisabled";
  await expect(pageMain(page).getByRole("button", { name: "Remove filter", exact: true })).toHaveCount(
    enabled ? conditions : 0
  );
  await expect(pageMain(page).getByRole("button", { name: "Add filter", exact: true })).toHaveCount(enabled ? 1 : 0);
  await expect(rowButton(page, "Clear all"))[state]();
  await expect(rowButton(page, "Save view"))[state]();
  await expect(displayButton(page))[state]();
  const columns = await columnButtons(page).all();
  expect(columns.length, "the table's columns").toBeGreaterThan(0);
  await Promise.all(columns.map((button) => expect(button)[state]()));
}
