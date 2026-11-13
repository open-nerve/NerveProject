// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ComponentType, type ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { MemoryRouter, Route, Routes, useParams } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Project } from "@nerve/api-client";
import { listedModules } from "@/components/modules/fake-module-lists";
import { useModuleDisplay } from "@/components/modules/use-module-display";
import { signedIn } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { moduleRowOf } from "@/store/planning/fake-modules";
import { ModuleFilterStore } from "@/store/planning/module_filter.store";
import { moduleOf } from "@/store/project/fake-projects";
import type { ProjectStore } from "@/store/project/project.store";
import { callerInWeb, its, type Caller } from "@/store/user/fake-permissions";
import type { IUserPermissionStore } from "@/store/user/permissions.store";
import Layout from "./layout";
import Page from "./page";

// A project's modules list page (P6a spec 2.6, R3, R4, R5): its layout, its header and its body, at the route's
// address. Turned off, it says so, whose way to the project's features is its admin's; else it shows nerve's list of
// the project's modules not archived (the read the project's wrapper makes, which the page shares), and waits for it as
// the list is laid out, says why nerve could not give it with a retry, or how to set the first module, which those who
// write the project's modules may (canWriteModules) by the project's dialogs' host. Everyone who sees the list orders,
// filters and lays it out, and sees the filters applied to it above it. The caller's role is the permission store's own
// (fake-permissions.ts); the modules a stand-in store's, the filters a session's; the list itself and the header's view
// a stand-in that says whose it is.

const held = vi.hoisted(() => ({
  modules: undefined as unknown as ReturnType<typeof import("@/components/modules/fake-module-lists").listedModules>,
  filters: undefined as unknown as import("@/store/planning/module_filter.store").ModuleFilterStore,
  permissions: undefined as IUserPermissionStore | undefined,
  projects: undefined as ProjectStore | undefined,
  toggle: vi.fn(),
  setLayout: undefined as ((layout: "list" | "board") => void) | undefined,
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "x" }) }));
vi.mock("@/hooks/store/use-modules", () => ({ useModules: () => held.modules }));
vi.mock("@/hooks/store/use-module-filters", () => ({ useModuleFilters: () => held.filters }));
vi.mock("@/hooks/store/use-command-palette", () => ({
  useCommandPalette: () => ({ toggleCreateModuleModal: held.toggle }),
}));
vi.mock("@/hooks/store/user", () => ({
  useUserPermissions: () => held.permissions,
  useUser: () => ({ data: { id: "me" } }),
}));
vi.mock("@/hooks/store/use-project", () => ({
  useProject: () => ({
    loader: "loaded",
    getProjectById: (id: string): Project | undefined => held.projects?.getProjectById(id),
  }),
}));
vi.mock("@/components/breadcrumbs/common", () => ({ CommonProjectBreadcrumbs: () => null }));
vi.mock("@/components/core/app-header", () => ({
  AppHeader: (props: { header: ReactNode; mobileHeader?: ReactNode }) => (
    <>
      <header>{props.header}</header>
      <nav>{props.mobileHeader}</nav>
    </>
  ),
}));
vi.mock("@/components/modules/modules-list-view", () => ({
  ModulesListView: (props: { projectId: string }) => <p>the modules of {props.projectId}</p>,
}));
vi.mock("@/components/modules/module-view-header", () => ({
  ModuleViewHeader: (props: { projectId: string }) => <div aria-label={`view of ${props.projectId}`} />,
}));
vi.mock("@/components/ui/loader/cycle-module-list-loader", () => ({
  CycleModuleListLayoutLoader: () => <p role="status">rows loading</p>,
}));
vi.mock("@/components/ui/loader/cycle-module-board-loader", () => ({
  CycleModuleBoardLayoutLoader: () => <p role="status">cards loading</p>,
}));
vi.mock("@/components/modules/modal", () => ({ CreateUpdateModuleModal: () => null }));

/** A route module's component, given its route's params as the router gives them. */
type RouteProps = { params: Readonly<Record<string, string | undefined>> };
const routeOf = (Component: unknown) => Component as ComponentType<RouteProps>;
function RouteOf({ Component }: { Component: ComponentType<RouteProps> }) {
  const params = useParams();
  return <Component params={params} />;
}
/** How the person lays the route's project's modules out, as the header's buttons do. */
function Layouts() {
  const { projectId = "" } = useParams();
  held.setLayout = useModuleDisplay(projectId).setLayout;
  return null;
}

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
  signedIn();
  localStorage.clear();
  held.toggle.mockReset();
  held.modules = listedModules();
  held.filters = new ModuleFilterStore();
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

const settle = () => act(async () => vi.advanceTimersByTimeAsync(100));
const text = () => host.textContent ?? "";
const button = (name: string) =>
  [...host.querySelectorAll("button")].find((one) => (one.getAttribute("aria-label") ?? one.textContent) === name);
const loading = () => host.querySelector('[role="status"]')?.textContent;

/** The caller of a role in acme and in web, at web's modules list, web being as project says. */
async function opened(caller: Caller, project: Partial<Project> = {}) {
  const { permissions, projects, acme, web } = await callerInWeb(
    caller.workspaceRole,
    caller.projectRole,
    caller.guestViewAll,
    project
  );
  Object.assign(held, { permissions, projects });
  await act(async () =>
    root.render(
      <SWRConfig value={{ provider: () => new Map() }}>
        <MemoryRouter initialEntries={[`/${acme.slug}/projects/${web.id}/modules`]}>
          <Routes>
            <Route
              path=":workspaceSlug/projects/:projectId/modules"
              element={
                <>
                  <Layouts />
                  <RouteOf Component={routeOf(Layout)} />
                </>
              }
            >
              <Route index element={<RouteOf Component={routeOf(Page)} />} />
            </Route>
            <Route path=":workspaceSlug/settings/projects/:projectId/features/modules" element={<p>the features</p>} />
          </Routes>
        </MemoryRouter>
      </SWRConfig>
    )
  );
  await settle();
  return { acme, web };
}

it.each([
  { caller: its.admin, may: true },
  { caller: its.member, may: false },
])(
  "shows $caller.who the project's modules turned off, reading none, whose features its admin alone manages",
  async ({ caller, may }) => {
    await opened(caller, { module_view: false });
    expect(text()).toContain("disabled_project.empty_state.module.title");
    expect(held.modules.asked()).toEqual([]);
    const manage = button("disabled_project.empty_state.module.primary_button.text");
    expect(manage?.disabled).toBe(!may);
    await act(async () => manage?.click());
    expect(text().includes("the features")).toBe(may);
  }
);

it("waits for nerve's list of the project's modules, the read the project's wrapper makes too", async () => {
  const { web } = await opened(its.member);
  expect(held.modules.asked()).toEqual([`modules of ${web.id}`]);
  expect(loading()).toBe("rows loading");
  expect(text()).not.toContain("the modules of");
});

it("waits as the person laid the project's modules out", async () => {
  await opened(its.member);
  await act(async () => held.setLayout?.("board"));
  expect(loading()).toBe("cards loading");
});

it("says why nerve could not list them, waiting no more, and lists them again on retry", async () => {
  const { web } = await opened(its.member);
  await held.modules.refuse("modules", refusal(503, "server_busy"));
  await settle();
  expect(host.querySelector('[role="alert"]')?.textContent).toContain("errors.server_busy");
  expect(loading()).toBeUndefined();
  await act(async () => button("common.retry")?.click());
  await settle();
  expect(held.modules.asked()).toEqual([`modules of ${web.id}`, `modules of ${web.id}`]);
  await held.modules.answer("modules", [moduleRowOf(moduleOf(web, "Auth"))]);
  await settle();
  expect(host.querySelector('[role="alert"]')).toBeNull();
  expect(text()).toContain(`the modules of ${web.id}`);
});

it.each([
  { caller: its.admin, may: true },
  { caller: its.member, may: true },
  { caller: its.guest, may: false },
  { caller: its.guestAdmin, may: false },
  { caller: its.guestAdminSeesAll, may: true },
])(
  "shows $caller.who, the project having no modules, how to set its first, which he may if he writes them",
  async ({ caller, may }) => {
    await opened(caller);
    await held.modules.answer("modules", []);
    await settle();
    expect(text()).toContain("project_empty_state.modules.title");
    expect(text()).not.toContain("the modules of");
    const first = button("project_empty_state.modules.cta_primary");
    expect(first?.disabled).toBe(!may);
    await act(async () => first?.click());
    expect(held.toggle.mock.calls).toEqual(may ? [[true]] : []);
  }
);

it.each([
  { caller: its.admin, may: true },
  { caller: its.member, may: true },
  { caller: its.guest, may: false },
  { caller: its.guestAdmin, may: false },
  { caller: its.guestAdminSeesAll, may: true },
])(
  "gives $caller.who the list's order, filters and layouts, and a new module if he writes the project's modules",
  async ({ caller, may }) => {
    const { acme, web } = await opened(caller);
    await held.modules.answer("modules", [moduleRowOf(moduleOf(web, "Auth"))]);
    await settle();
    expect(text()).toContain(`the modules of ${web.id}`);
    const crumb = [...host.querySelectorAll("header a")].find((link) => link.textContent === "modules");
    expect(crumb?.getAttribute("href")).toBe(`/${acme.slug}/projects/${web.id}/modules`);
    expect(host.querySelector(`header [aria-label="view of ${web.id}"]`)).not.toBeNull();
    expect([...host.querySelectorAll("nav button")].map((one) => one.textContent)).toEqual(["common.layout"]);
    const add = [...host.querySelectorAll("header button")].find((one) =>
      one.textContent?.includes("project_module.add_module")
    );
    expect(add !== undefined).toBe(may);
    await act(async () => (add as HTMLButtonElement | undefined)?.click());
    expect(held.toggle.mock.calls).toEqual(may ? [[true]] : []);
  }
);

// nerve refuses every write of the project's modules while they are turned off (planning.modules_disabled), its
// admin's too; and there is no list to order, filter or lay out
it("gives no one the list's view nor a new module while the project's modules are turned off", async () => {
  const { web } = await opened(its.admin, { module_view: false });
  expect(host.querySelector(`[aria-label="view of ${web.id}"]`)).toBeNull();
  expect(host.querySelectorAll("nav button")).toHaveLength(0);
  expect(text()).not.toContain("project_module.add_module");
});

it("shows the filters applied to the list above it, the list's own, which clear there", async () => {
  const { web } = await opened(its.guest);
  await held.modules.answer("modules", [moduleRowOf(moduleOf(web, "Auth"))]);
  await settle();
  expect(button("common.clear_all")).toBeUndefined();
  await act(async () => held.filters.updateFilters(web.id, "default", { status: ["planned"] }));
  expect(text()).toContain("project_modules.status.planned");
  expect(text()).toContain(`the modules of ${web.id}`);
  // the archive's filters are its own: not the list's
  await act(async () => held.filters.updateFilters(web.id, "archived", { start_date: ["1_weeks;after;fromnow"] }));
  expect(text()).not.toContain("1 week from now");
  await act(async () => button("common.clear_all")?.click());
  expect(held.filters.getFilters(web.id, "default")).toEqual({});
  expect(held.filters.getFilters(web.id, "archived")).toEqual({ start_date: ["1_weeks;after;fromnow"] });
  expect(button("common.clear_all")).toBeUndefined();
});
