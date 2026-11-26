/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { components } from "@nerve/api-client";
import { expect, type Locator, type Page, type TestInfo } from "@playwright/test";

import {
  addProjectMembers,
  changeProject,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
  type Api,
} from "./api";
import { bearer } from "./auth";
import { demoteToGuest } from "./intake-pages";
import { PROJECT_VIEWS, valued } from "./mounts";
import { person } from "./notification-pages";
import { createProjectView } from "./views";

// A project's views pages (M7 P6a spec 2.3–2.7, 2.11), as a person uses them: their addresses and reads, what a story
// finds on them by role and name, the draft's controls that Update view stops, and the world of their stories. What
// they share with the cycle, the notification and the intake pages' stories (recordApi, toast, dismissToasts,
// menuEntries, failedLoads, seenBy; person, demoteToGuest; holdAnswer) they import from there, not copied here; the
// views' calls and rows are views.ts's.

type ViewCreate = components["schemas"]["ViewCreate"];
type Project = components["schemas"]["Project"];

/**
 * The world of the views pages' stories: alice's workspace Acme and its project Web, its views turned on, of which bob
 * is a member and gus a guest, who sees his own work items of it alone, as a new project's guests do, and so his own
 * views alone; each named, alice "Alice", bob "Bob", gus "Gus". gus is a member until guest() makes him Web's guest,
 * so that a view of his own can be made first. view makes a view of Web as the one given.
 */
export async function viewWorld(api: Api, testInfo: TestInfo) {
  const [alice, bob, gus] = await Promise.all([
    person(api, testInfo, "alice", "Alice"),
    person(api, testInfo, "bob", "Bob"),
    person(api, testInfo, "gus", "Gus"),
  ]);
  const slug = slugFor(testInfo);
  await createWorkspace(api, alice.token, { name: "Acme", slug });
  await inviteAndAccept(api, alice.token, slug, { email: bob.email, token: bob.token }, 15);
  await inviteAndAccept(api, alice.token, slug, { email: gus.email, token: gus.token }, 15);
  const project = async (name: string, identifier: string, on: boolean, members = [bob, gus]) => {
    const made = await createProject(api, alice.token, slug, { name, identifier });
    if (members.length > 0)
      await addProjectMembers(
        api,
        alice.token,
        made.id,
        members.map(({ id }) => ({ member_id: id, role: 15 as const }))
      );
    if (on) await turnViews(api, alice.token, made, true);
    return made;
  };
  const web = await project("Web", "WEB", true);
  return {
    alice,
    bob,
    gus,
    slug,
    web,
    project,
    view: (who: { token: string }, body: ViewCreate, projectId = web.id) =>
      createProjectView(api, who.token, projectId, body),
    guest: (projectOf: Project = web) => demoteToGuest(api, alice.token, projectOf, gus.id),
  };
}

/** Turns the views of project on or off, as its admin of token. */
export async function turnViews(api: Api, token: string, project: Project, on: boolean): Promise<void> {
  const turned = await changeProject(api, token, project.id, { issue_views_view: on });
  expect(turned.status, `${project.name}'s views turned ${on ? "on" : "off"}: ${JSON.stringify(turned)}`).toBe(200);
}

/** Deletes the label of id through the API, as the caller of token, an admin or a member of its project. */
export async function deleteLabel(api: Api, token: string, id: string): Promise<void> {
  const { error, response } = await api.DELETE("/api/v0/labels/{label_id}", {
    params: { path: { label_id: id } },
    headers: bearer(token),
  });
  expect(response.status, `delete the label ${id}: ${JSON.stringify(error)}`).toBe(204);
}

/** The address of the views list of the project of projectId in the workspace of slug. */
export function viewsPath(slug: string, projectId: string): string {
  return `/${slug}/projects/${projectId}/views`;
}

/** The address of the page of the view of id of that project. */
export function viewPath(slug: string, projectId: string, id: string): string {
  return `${viewsPath(slug, projectId)}/${id}`;
}

/** The read of the views of the project of projectId, as recordApi records it (PROJECT_VIEWS). */
export function viewsRead(projectId: string): string {
  const [read = ""] = PROJECT_VIEWS;
  return valued(read, { [projectId]: "{project}" });
}

/** The read of the view of id, as recordApi records it. */
export function viewRead(id: string): string {
  return `GET /api/v0/views/${id}`;
}

/** Of requests (recordApi's), the reads of the views: the project's list and each view's. */
export function viewReads(requests: readonly string[]): string[] {
  return requests.filter((request) => /^GET \/api\/v0\/(projects\/[^/?]+\/views|views\/[^/?]+)$/.test(request));
}

/** Of requests (recordApi's), the reads of the work items of the project of projectId, by their queries. */
export function listReads(requests: readonly string[], projectId: string): URLSearchParams[] {
  const read = `GET /api/v0/projects/${projectId}/issues?`;
  return requests
    .filter((request) => request.startsWith(read))
    .map((request) => new URLSearchParams(request.slice(read.length)));
}

/**
 * Checks that query holds each of the words of saved (savedQuery's, or a grouped view's own), a list's query: as nerve
 * takes them, the alternatives of a key joined by commas. The query holds more (its limit, its layout's options).
 */
export function expectQueryHolds(
  query: URLSearchParams,
  saved: Readonly<Record<string, string | readonly string[] | undefined>>
): void {
  const words = Object.fromEntries(
    Object.entries(saved).map(([key, value]) => {
      if (value === undefined) throw new Error(`the words of ${key} are not the story's`);
      return [key, typeof value === "string" ? value : value.join(",")];
    })
  );
  expect(Object.fromEntries(query), "the list's query").toMatchObject(words);
}

/** The names of the views the views list shows, in its order: each row's menu is named by its view. */
export async function listedViews(page: Page): Promise<string[]> {
  const labels = await pageMain(page)
    .locator('button[aria-label$=" options"]')
    .evaluateAll((buttons) => buttons.map((button) => button.getAttribute("aria-label") ?? ""));
  return labels.map((label) => label.slice(0, -" options".length));
}

/** The page's own region: the header with its crumbs and the content, not the app's sidebar. */
export function pageMain(page: Page): Locator {
  return page.locator("main main");
}

/** The crumb of the project's views in the page's header, which links to its list. */
export function viewsCrumb(page: Page, slug: string, projectId: string): Locator {
  return pageMain(page).locator(`a[href="${viewsPath(slug, projectId)}"]`);
}

/** The buttons of a view's page's header that lay its work items out, by name: "List", "Board", "Calendar", "Table". */
export function layoutButtons(page: Page): Locator {
  return pageMain(page).getByRole("button", { name: /^(List|Board|Calendar|Table) Layout$/ });
}

/** A button of the filters row, by its name: "Clear all", "Save view", "Update view", "Confirming". */
export function rowButton(page: Page, name: string): Locator {
  return pageMain(page).getByRole("button", { name, exact: true });
}

/**
 * Checks the ways a view's page changes its draft, with the wide header (R29; P6a rulings, T8 ①): while Update view is
 * out (enabled false), each of the conditions without its remove button, no Add filter anywhere (the row's, or the
 * header's filters toggle's), Clear all and Save view stopped, and the header's layouts and display stopped; once
 * nerve has answered, each working again, the row's Add filter back.
 */
export async function expectDraftControls(page: Page, enabled: boolean, conditions: number): Promise<void> {
  const state = enabled ? "toBeEnabled" : "toBeDisabled";
  await expect(pageMain(page).getByRole("button", { name: "Remove filter", exact: true })).toHaveCount(
    enabled ? conditions : 0
  );
  await expect(pageMain(page).getByRole("button", { name: "Add filter", exact: true })).toHaveCount(enabled ? 1 : 0);
  await expect(rowButton(page, "Clear all"))[state]();
  await expect(rowButton(page, "Save view"))[state]();
  await expect(layoutButtons(page)).toHaveCount(4);
  await Promise.all((await layoutButtons(page).all()).map((button) => expect(button)[state]()));
  await expect(pageMain(page).getByRole("button", { name: "Display", exact: true }))[state]();
}
