// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Project } from "@nerve/api-client";
import { settle, where } from "@/components/common/fake-quick-actions";
import { listViews } from "@/components/views/fake-view-pages";
import { problem, type FakeNerve } from "@/lib/auth/fake-nerve";
import type { ProjectStore } from "@/store/project/project.store";
import { its, type Caller } from "@/store/user/fake-permissions";
import type { IUserPermissionStore } from "@/store/user/permissions.store";
import { ME, viewOf, viewRequests, viewsTab } from "@/store/views/fake-views";
import type { ViewStore } from "@/store/views/view.store";
import { ListPages, standInMembers } from "./fake-list-page";

// A project's views list page (P6a spec 2.2, 2.3, R4, R5, R20): its layout, its header and its body, at the route's
// address. Turned off, it says so and reads nothing, whose way to the project's features is its admin's; else it reads
// nerve's list of the project's views, waits for it, says why nerve could not give it with a retry, or how to make the
// first view, which only those who may create one may, as "Add view" in its header; and its search, filters and order
// are the page's, which end as the project changes. The caller's role is the permission store's own
// (fake-permissions.ts); the views a views' store's over a fake nerve (fake-view-pages.tsx); the dialogs' host of the
// project (toggleCreateViewModal) a stand-in that keeps what it was asked.

const held = vi.hoisted(() => ({
  views: undefined as unknown as ViewStore,
  permissions: undefined as IUserPermissionStore | undefined,
  projects: undefined as ProjectStore | undefined,
  members: undefined as unknown as ReturnType<typeof import("./fake-list-page").standInMembers>,
  toggle: vi.fn(),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "x" }) }));
vi.mock("@/hooks/store/use-views", () => ({ useViews: () => held.views }));
// the projects as the caller's workspace lists them, its projects fetched
vi.mock("@/hooks/store/use-project", () => ({
  useProject: () => ({ loader: "loaded", getProjectById: (id: string) => held.projects?.getProjectById(id) }),
}));
vi.mock("@/hooks/store/use-member", () => ({ useMember: () => held.members }));
vi.mock("@/hooks/store/user", () => ({
  useUserPermissions: () => held.permissions,
  useUser: () => ({ data: { id: "u-me" } }),
}));
vi.mock("@/hooks/store/use-command-palette", () => ({
  useCommandPalette: () => ({ toggleCreateViewModal: held.toggle }),
}));
vi.mock("@/hooks/use-route-workspace", () => ({ useRouteWorkspace: () => null }));
vi.mock("@/components/assets/profile-image", () => ({ ProfileAvatar: () => null }));
vi.mock("@/components/breadcrumbs/common", () => ({ CommonProjectBreadcrumbs: () => null }));
vi.mock("@/components/core/app-header", () => ({
  AppHeader: (props: { header: ReactNode; mobileHeader?: ReactNode }) => (
    <>
      <header>{props.header}</header>
      <nav>{props.mobileHeader}</nav>
    </>
  ),
}));
vi.mock("@/components/ui/loader/view-list-loader", () => ({
  ViewListLoader: () => <p role="status">views loading</p>,
}));

let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(() => {
  vi.useFakeTimers();
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
  held.toggle.mockReset();
  held.members = standInMembers();
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

const text = () => host.textContent ?? "";
/** The buttons in the part of the page of selector (none: the whole page). */
const buttons = (within = "") => [...host.querySelectorAll<HTMLButtonElement>(`${within} button`)];
const button = (name: string, within = "") =>
  buttons(within).find((one) => (one.getAttribute("aria-label") ?? one.textContent) === name);
const loading = () => host.querySelector('[role="status"]')?.textContent;
/** The names of the views the list shows, in its order. */
const rows = () => [...host.querySelectorAll("a[href*='/views/v-']")].map((link) => link.textContent);
/** What the caller asked nerve: each request's method and path. */
const asked = (nerve: FakeNerve) => nerve.calls.map(({ method, path }) => `${method} ${path}`);
/** Types query into the header's search, as a person does. */
async function search(query: string) {
  const input = host.querySelector<HTMLInputElement>("header input");
  await act(async () => {
    if (!input) return;
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")?.set?.call(input, query);
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
}

/** The caller of a role in acme and in its projects, at web's views list, web being as fields say. */
async function opened(caller: Caller, fields: Partial<Project> = {}) {
  const tab = await viewsTab(caller, fields);
  Object.assign(held, { views: tab.views, permissions: tab.permissions, projects: tab.projects });
  await act(async () => root.render(<ListPages path={`/${tab.acme.slug}/projects/${tab.web.id}/views`} />));
  await settle();
  return tab;
}

it.each([
  { caller: its.admin, may: true },
  { caller: its.member, may: false },
])(
  "shows $caller.who the project's views turned off, reading none, whose features its admin alone manages",
  async ({ caller, may }) => {
    const { nerve } = await opened(caller, { issue_views_view: false });
    expect(text()).toContain("disabled_project.empty_state.view.title");
    expect(asked(nerve)).toEqual([]);
    // no list to search, filter or order, and no new view: nerve refuses it while they are off
    expect(host.querySelector("header input")).toBeNull();
    expect(text()).not.toContain("project_views.add_view");
    expect(buttons("nav")).toEqual([]);
    const manage = button("disabled_project.empty_state.view.primary_button.text");
    expect(manage?.disabled).toBe(!may);
    await act(async () => manage?.click());
    expect(text().includes("the features")).toBe(may);
  }
);

it("reads nerve's list of the project's views once, waiting for it", async () => {
  const { nerve, web } = await opened(its.guest);
  expect(asked(nerve)).toEqual([`GET ${viewRequests.list(web)}`]);
  expect(loading()).toBe("views loading");
  await listViews(nerve, web, [viewOf(web, "Bugs", ME)]);
  expect(loading()).toBeUndefined();
  expect(rows()).toEqual(["Bugs"]);
  expect(document.title).toBe("WEB - views");
  const crumb = [...host.querySelectorAll("header a")].find((link) => link.textContent === "views");
  expect(crumb?.getAttribute("href")).toBe(`/acme/projects/${web.id}/views`);
});

it("says why nerve could not list them, waiting no more, and lists them again on retry", async () => {
  const { nerve, web } = await opened(its.member);
  await act(async () => nerve.to(viewRequests.list(web))[0]?.answer(problem(503, "server_busy")));
  await settle();
  expect(host.querySelector('[role="alert"]')?.textContent).toContain("errors.server_busy");
  expect(loading()).toBeUndefined();
  await act(async () => button("common.retry")?.click());
  await settle();
  expect(asked(nerve)).toEqual([`GET ${viewRequests.list(web)}`, `GET ${viewRequests.list(web)}`]);
  await listViews(nerve, web, [viewOf(web, "Bugs", ME)]);
  expect(host.querySelector('[role="alert"]')).toBeNull();
  expect(rows()).toEqual(["Bugs"]);
});

it.each([
  { caller: its.admin, may: true },
  { caller: its.member, may: true },
  { caller: its.memberAdmin, may: true },
  { caller: its.guest, may: false },
  { caller: its.guestSeesAll, may: false },
])(
  "shows $caller.who, the project having no views, how to make the first, which he may if he may create one",
  async ({ caller, may }) => {
    const { nerve, web } = await opened(caller);
    await listViews(nerve, web, []);
    expect(text()).toContain("project_empty_state.views.title");
    const first = button("project_empty_state.views.cta_primary");
    expect(first?.disabled).toBe(!may);
    await act(async () => first?.click());
    expect(held.toggle.mock.calls).toEqual(may ? [[true]] : []);
  }
);

it.each([
  { caller: its.admin, may: true },
  { caller: its.member, may: true },
  { caller: its.guestAdmin, may: true },
  { caller: its.guest, may: false },
  { caller: its.guestSeesAll, may: false },
])("gives $caller.who the list's search, filters and order, and a new view if he may create one", async (each) => {
  const { nerve, web } = await opened(each.caller);
  await listViews(nerve, web, [viewOf(web, "Bugs", ME)]);
  expect(host.querySelector("header input")).not.toBeNull();
  expect(button("common.filters", "header")).toBeDefined();
  expect(button("common.filters", "nav")).toBeDefined();
  const add = button("project_views.add_view", "header");
  expect(add !== undefined).toBe(each.may);
  await act(async () => add?.click());
  expect(held.toggle.mock.calls).toEqual(each.may ? [[true]] : []);
});

it("says no view matches the search, the project having views", async () => {
  const { nerve, web } = await opened(its.member);
  await listViews(nerve, web, [viewOf(web, "Bugs", ME)]);
  await search("Release");
  expect(rows()).toEqual([]);
  expect(text()).toContain("common_empty_state.search.title");
});

it("keeps the page's search for its project alone: another project's list opens with none", async () => {
  const { nerve, web, ops } = await opened(its.member);
  await listViews(nerve, web, [viewOf(web, "Bugs", ME), viewOf(web, "Mine", ME)]);
  await search("Bugs");
  expect(rows()).toEqual(["Bugs"]);
  await act(async () => where.navigate?.(`/acme/projects/${ops.id}/views`));
  await settle();
  await listViews(nerve, ops, [viewOf(ops, "Bugs", ME), viewOf(ops, "Infra", ME)]);
  expect(host.querySelector<HTMLInputElement>("header input")?.value).toBe("");
  expect(rows()).toEqual(["Bugs", "Infra"]);
});
